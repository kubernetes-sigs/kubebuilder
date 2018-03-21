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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

// CRDInstallStrategy installs APIs into a cluster using CRDs
type CRDInstallStrategy struct {
	// Name is the installation
	Name string

	// ControllerManagerImage is the container image to use for the controller
	ControllerManagerImage string

	// DocsImage is the container image to use for hosting reference documentation
	DocsImage string

	// APIMeta contains the generated API metadata from the pkg/apis
	APIMeta APIMeta
}

// GetServiceAccount returns the default ServiceAccount
func (s *CRDInstallStrategy) GetServiceAccount() string {
	return "default"
}

// GetCRDs returns the generated CRDs from APIMeta.GetCRDs()
func (s *CRDInstallStrategy) GetCRDs() []*extensionsv1beta1.CustomResourceDefinition {
	return s.APIMeta.GetCRDs()
}

// GetNamespace returns the strategy name suffixed "with -system"
func (s *CRDInstallStrategy) GetNamespace() *corev1.Namespace {
	ns := &corev1.Namespace{}
	ns.Name = fmt.Sprintf("%v-system", s.Name)
	return ns
}

// GetClusterRole returns a ClusterRule with the generated rules by APIMeta.GetPolicyRules
func (s *CRDInstallStrategy) GetClusterRole() *rbacv1.ClusterRole {
	ns := s.GetNamespace()
	role := &rbacv1.ClusterRole{}
	role.Namespace = ns.Name
	role.Name = fmt.Sprintf("%v-role", ns.Name)
	role.Rules = s.APIMeta.GetPolicyRules()
	return role
}

// GetClusterRoleBinding returns a binding for the ServiceAccount and ClusterRole
func (s *CRDInstallStrategy) GetClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	rolebinding := &rbacv1.ClusterRoleBinding{}
	ns := s.GetNamespace()
	rolebinding.Namespace = ns.Name
	rolebinding.Name = fmt.Sprintf("%v-rolebinding", rolebinding.Namespace)

	// Bind the Namesapce default ServiceAccount to the system role for the controller
	rolebinding.Subjects = []rbacv1.Subject{
		{
			Name:      s.GetServiceAccount(),
			Namespace: ns.Name,
			Kind:      "ServiceAccount",
		},
	}
	rolebinding.RoleRef = rbacv1.RoleRef{
		Name:     fmt.Sprintf("%v-role", ns.Name),
		Kind:     "ClusterRole",
		APIGroup: "rbac.authorization.k8s.io",
	}
	return rolebinding
}

// GetDeployments returns a Deployment to run the Image
func (s *CRDInstallStrategy) GetDeployments() []*appsv1.Deployment {
	controllerManager := &appsv1.Deployment{}
	controllerManager.Namespace = fmt.Sprintf("%v-system", s.Name)
	controllerManager.Name = fmt.Sprintf("%v-controller-manager", s.Name)
	controllerManager.Labels = map[string]string{
		"app": "controller-manager",
		"api": s.Name,
	}
	controllerManager.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: controllerManager.Labels,
	}
	controllerManager.Spec.Template.Labels = controllerManager.Labels
	controllerManager.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:    "controller-manager",
			Image:   s.ControllerManagerImage,
			Command: []string{"/root/controller-manager"},
			Args:    []string{"--install-crds=false"},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("20Mi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("30Mi"),
				},
			},
		},
	}

	deps := []*appsv1.Deployment{controllerManager}
	if len(s.DocsImage) > 0 {
		docs := &appsv1.Deployment{}
		docs.Namespace = fmt.Sprintf("%v-system", s.Name)
		docs.Name = fmt.Sprintf("%v-docs", s.Name)
		docs.Labels = map[string]string{
			"app": "docs",
			"api": s.Name,
		}
		docs.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: docs.Labels,
		}
		docs.Spec.Template.Labels = docs.Labels
		docs.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:  "docs",
				Image: s.DocsImage,
			},
		}
		deps = append(deps, docs)
	}

	return deps
}

func (s *CRDInstallStrategy) GetServices() []*corev1.Service {
	ns := s.GetNamespace().Name
	docsService := &corev1.Service{}
	docsService.Name = fmt.Sprintf("%s-docs", s.Name)
	docsService.Namespace = ns
	docsService.Labels = map[string]string{
		"app": "docs",
	}
	docsService.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "docs",
			Port:       80,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(80),
		},
	}
	docsService.Spec.Selector = map[string]string{
		"app": "docs",
	}

	return []*corev1.Service{docsService}
}

func (s *CRDInstallStrategy) GetSecrets() []*corev1.Secret                         { return nil }
func (s *CRDInstallStrategy) GetConfigMaps() []*corev1.ConfigMap                   { return nil }
func (s *CRDInstallStrategy) GetStatefulSets() []*appsv1.StatefulSet               { return nil }
func (s *CRDInstallStrategy) BeforeInstall() error                                 { return nil }
func (s *CRDInstallStrategy) AfterInstall() error                                  { return nil }
func (s *CRDInstallStrategy) GetAPIServices() []*apiregistrationv1beta1.APIService { return nil }
