/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package args

import (
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

// InjectArgs are the common arguments for initializing controllers and admission hooks
type InjectArgs struct {
	// Config is the rest config to talk to an API server
	Config *rest.Config

	// KubernetesClientSet is a clientset to talk to Kuberntes apis
	KubernetesClientSet kubernetes.Interface

	// KubernetesInformers contains a Kubernetes informers factory
	KubernetesInformers informers.SharedInformerFactory

	// ControllerManager is the controller manager
	ControllerManager *controller.ControllerManager

	// EventBroadcaster
	EventBroadcaster record.EventBroadcaster
}

// CreateRecorder returns a new recorder
func (iargs InjectArgs) CreateRecorder(name string) record.EventRecorder {
	return iargs.EventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: name})
}

// CreateInjectArgs returns new arguments for initializing objects
func CreateInjectArgs(config *rest.Config) InjectArgs {
	cs := kubernetes.NewForConfigOrDie(config)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: cs.CoreV1().Events("")})
	return InjectArgs{
		Config:              config,
		KubernetesClientSet: cs,
		KubernetesInformers: informers.NewSharedInformerFactory(cs, 2*time.Minute),
		ControllerManager:   &controller.ControllerManager{},
		EventBroadcaster:    eventBroadcaster,
	}
}

// Injector is used by code generators to register code generated objects
type Injector struct {
	// CRDs are CRDs that may be created / updated at startup
	CRDs []*apiextensionsv1beta1.CustomResourceDefinition

	// PolicyRules are RBAC policy rules that may be installed with the controller
	PolicyRules []rbacv1.PolicyRule

	// GroupVersions are the api group versions in the CRDs
	GroupVersions []schema.GroupVersion

	// Runnables objects run with RunArguments
	Runnables []Runnable

	// RunFns are functions run with RunArguments
	RunFns []RunFn

	// ControllerManager is used to register Informers and Controllers
	ControllerManager *controller.ControllerManager
}

// Run will run all of the registered RunFns and Runnables
func (i Injector) Run(a run.RunArguments) error {
	for _, r := range i.Runnables {
		go r.Run(a)
	}
	for _, r := range i.RunFns {
		go r(a)
	}
	return nil
}

// RunFn can be registered with an Injector and run
type RunFn func(arguments run.RunArguments) error

// Runnable can be registered with an Injector and run
type Runnable interface {
	Run(arguments run.RunArguments) error
}
