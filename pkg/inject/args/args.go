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

	"github.com/kubernetes-sigs/kubebuilder/pkg/admission"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
    "k8s.io/client-go/informers"
)

// InjectArgs are the common arguments for initializing controllers and admission hooks
type InjectArgs struct {
	// Config is the rest config to talk to an API server
	Config *rest.Config

	// KubernetesClientSet is a clientset to talk to Kuberntes apis
	KubernetesClientSet *kubernetes.Clientset

	// KubernetesInformers contains a Kubernetes informers factory
	KubernetesInformers informers.SharedInformerFactory

	// ControllerManager is the controller manager
	ControllerManager *controller.ControllerManager

	// AdmissionManager is the admission webhook manager
	AdmissionHandler *admission.AdmissionManager
}

// CreateInjectArgs returns new arguments for initializing objects
func CreateInjectArgs(config *rest.Config) InjectArgs {
    cs := kubernetes.NewForConfigOrDie(config)
	return InjectArgs{
		Config:              config,
		KubernetesClientSet: cs,
        KubernetesInformers: informers.NewSharedInformerFactory(cs, 2 * time.Minute),
		ControllerManager:   &controller.ControllerManager{},
		AdmissionHandler:    &admission.AdmissionManager{},
	}
}
