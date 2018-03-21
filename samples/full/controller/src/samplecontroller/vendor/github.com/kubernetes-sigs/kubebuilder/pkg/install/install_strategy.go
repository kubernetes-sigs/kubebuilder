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

package install

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

// DefaultInstallStrategy is the default install strategy to use
type DefaultInstallStrategy = CRDInstallStrategy

// InstallStrategy defines what resources should be created as part of installing
// an API extension.
type InstallStrategy interface {
	// GetCRDs returns a list of CRDs to create
	GetCRDs() []*extensionsv1beta1.CustomResourceDefinition

	// GetNamespace returns the namespace to install the controller-manager into.
	GetNamespace() *corev1.Namespace

	// GetServiceAccount returns the name of the ServiceAccount to use
	GetServiceAccount() string

	// GetClusterRole returns a ClusterRole to create
	GetClusterRole() *rbacv1.ClusterRole

	// GetClusterRoleBinding returns a GetClusterRoleBinding to create
	GetClusterRoleBinding() *rbacv1.ClusterRoleBinding

	// GetDeployments returns the Deployments to create
	GetDeployments() []*appsv1.Deployment

	// GetStatefulSets the StatefulSets to create
	GetStatefulSets() []*appsv1.StatefulSet

	// GetSecrets returns the Secrets to create
	GetSecrets() []*corev1.Secret

	// GetConfigMaps returns the ConfigMaps to create
	GetConfigMaps() []*corev1.ConfigMap

	// GetServices returns the Services to create
	GetServices() []*corev1.Service

	// GetAPIServices returns the APIServices to create
	GetAPIServices() []*apiregistrationv1beta1.APIService

	// BeforeInstall is run before installing other components
	BeforeInstall() error

	// AfterInstall is run after installing other components
	AfterInstall() error
}

// EmptyInstallStrategy is a Strategy that doesn't create anything but can be embedded in
// another InstallStrategy.
type EmptyInstallStrategy struct{}

func (s EmptyInstallStrategy) AfterInstall() error                                  { return nil }
func (s EmptyInstallStrategy) BeforeInstall() error                                 { return nil }
func (s EmptyInstallStrategy) GetAPIServices() []*apiregistrationv1beta1.APIService { return nil }
func (s EmptyInstallStrategy) GetClusterRole() *rbacv1.ClusterRole                  { return nil }
func (s EmptyInstallStrategy) GetClusterRoleBinding() *rbacv1.ClusterRoleBinding    { return nil }
func (s EmptyInstallStrategy) GetConfigMaps() []*corev1.ConfigMap                   { return nil }
func (s EmptyInstallStrategy) GetCRDs() []*extensionsv1beta1.CustomResourceDefinition {
	return []*extensionsv1beta1.CustomResourceDefinition{}
}
func (s EmptyInstallStrategy) GetDeployments() []*appsv1.Deployment   { return nil }
func (s EmptyInstallStrategy) GetNamespace() *corev1.Namespace        { return nil }
func (s EmptyInstallStrategy) GetSecrets() []*corev1.Secret           { return nil }
func (s EmptyInstallStrategy) GetServiceAccount() string              { return "" }
func (s EmptyInstallStrategy) GetServices() []*corev1.Service         { return nil }
func (s EmptyInstallStrategy) GetStatefulSets() []*appsv1.StatefulSet { return nil }

// APIMeta returns metadata about the APIs
type APIMeta interface {
	// GetCRDs returns the CRDs
	GetCRDs() []*extensionsv1beta1.CustomResourceDefinition

	// GetPolicyRules returns the PolicyRules to apply to the ServiceAccount running the controller
	GetPolicyRules() []rbacv1.PolicyRule

	// GetGroupVersions returns the GroupVersions of the CRDs or aggregated APIs
	GetGroupVersions() []schema.GroupVersion
}
