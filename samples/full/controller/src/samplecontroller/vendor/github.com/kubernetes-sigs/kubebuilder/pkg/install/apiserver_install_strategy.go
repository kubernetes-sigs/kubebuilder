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
	"sync"
)

// ApiserverInstallStrategy installs APIs using apiserver aggregation.  Creates a StatefulSet for
// etcd storage.
type ApiserverInstallStrategy struct {
	// Name is the name of the installation
	Name string

	// ApiserverImage is the container image for the aggregated apiserver
	ApiserverImage string

	// ControllerManagerImage is the container image to use for the controller
	ControllerManagerImage string

	// DocsImage is the container image to use for hosting reference documentation
	DocsImage string

	// APIMeta contains the generated API metadata from the pkg/apis
	APIMeta APIMeta

	// Certs are the certs for installing the aggregated apiserver
	Certs    *Certs
	certOnce sync.Once
}

// GetServiceAccount returns the default ServiceAccount
func (s *ApiserverInstallStrategy) GetServiceAccount() string {
	return "default"
}

// GetCRDs returns the generated CRDs from APIMeta.GetCRDs()
func (s *ApiserverInstallStrategy) GetCRDs() []extensionsv1beta1.CustomResourceDefinition {
	return []extensionsv1beta1.CustomResourceDefinition{}
}

// GetNamespace returns the strategy name suffixed "with -system"
func (s *ApiserverInstallStrategy) GetNamespace() *corev1.Namespace {
	ns := &corev1.Namespace{}
	ns.Name = fmt.Sprintf("%v-system", s.Name)
	return ns
}

// GetClusterRole returns a ClusterRule with the generated rules by APIMeta.GetPolicyRules
func (s *ApiserverInstallStrategy) GetClusterRole() *rbacv1.ClusterRole {
	ns := s.GetNamespace()
	role := &rbacv1.ClusterRole{}
	role.Namespace = ns.Name
	role.Name = fmt.Sprintf("%v-role", ns.Name)
	role.Rules = s.APIMeta.GetPolicyRules()
	role.Rules = append(role.Rules,
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		})
	role.Rules = append(role.Rules,
		rbacv1.PolicyRule{
			APIGroups: []string{"authorization.k8s.io"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		})

	return role
}

// GetClusterRoleBinding returns a binding for the ServiceAccount and ClusterRole
func (s *ApiserverInstallStrategy) GetClusterRoleBinding() *rbacv1.ClusterRoleBinding {
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
func (s *ApiserverInstallStrategy) GetDeployments() []*appsv1.Deployment {
	// Controller ControllerManager
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

	// Apiserver
	apiserver := &appsv1.Deployment{}
	apiserver.Name = fmt.Sprintf("%v-apiserver", s.Name)
	apiserver.Namespace = fmt.Sprintf("%v-system", s.Name)
	apiserver.Labels = map[string]string{
		"app":       "apiserver",
		"api":       s.Name,
		"apiserver": "true",
	}
	replicas := int32(1)
	apiserver.Spec.Replicas = &replicas
	apiserver.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: apiserver.Labels,
	}
	apiserver.Spec.Template.Labels = apiserver.Labels
	apiserver.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:            "apiserver",
			Image:           s.ApiserverImage,
			ImagePullPolicy: "Always",
			Command:         []string{"/root/apiserver"},
			Args: []string{
				fmt.Sprintf("--etcd-servers=http://%s-etcd:2379", s.Name),
				"--tls-cert-file=/apiserver.local.config/certificates/tls.crt",
				"--tls-private-key-file=/apiserver.local.config/certificates/tls.key",
				"--audit-log-path=-",
				"--audit-log-maxage=0",
				"--audit-log-maxbackup=0",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "apiserver-certs",
					MountPath: "/apiserver.local.config/certificates",
					ReadOnly:  true,
				},
			},
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
	apiserver.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "apiserver-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("apiserver-certs"),
				},
			},
		},
	}
	deps := []*appsv1.Deployment{controllerManager, apiserver}
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

// GetStatefulSets returns and StatefulSet resource for an etcd instance
func (s *ApiserverInstallStrategy) GetStatefulSets() []*appsv1.StatefulSet {
	etcd := &appsv1.StatefulSet{}
	etcd.Name = "etcd"
	etcd.Namespace = s.GetNamespace().Name
	etcd.Spec.ServiceName = "etcd"
	etcd.Labels = map[string]string{
		"app": "etcd",
	}
	replicas := int32(1)
	etcd.Spec.Replicas = &replicas
	terminationGracePeriodSeconds := int64(10)
	etcd.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: etcd.Labels,
	}
	etcd.Spec.Template.Labels = etcd.Labels
	etcd.Spec.Template.Spec.TerminationGracePeriodSeconds = &terminationGracePeriodSeconds
	etcd.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:            "etcd",
			Image:           "quay.io/coreos/etcd:latest",
			ImagePullPolicy: "Always",
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
			Env: []corev1.EnvVar{
				{
					Name:  "ETCD_DATA_DIR",
					Value: "/etcd-data-dir",
				},
			},
			Command: []string{
				"/usr/local/bin/etcd",
			},
			Args: []string{
				"--listen-client-urls=http://0.0.0.0:2379",
				"--advertise-client-urls=http://localhost:2379",
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 2379,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "etcd-data-dir",
					MountPath: "/etcd-data-dir",
				},
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(2379),
						Path: "/health",
					},
				},
				FailureThreshold:    1,
				InitialDelaySeconds: 10,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      2,
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(2379),
						Path: "/health",
					},
				},
				FailureThreshold:    3,
				InitialDelaySeconds: 10,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      2,
			},
		},
	}
	etcd.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "etcd-data-dir",
				Annotations: map[string]string{
					"volume.beta.kubernetes.io/storage-class": "standard",
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						"storage": resource.MustParse("10Gi"),
					},
				},
			},
		},
	}
	return []*appsv1.StatefulSet{etcd}
}

func (s *ApiserverInstallStrategy) GetServices() []*corev1.Service {
	ns := s.GetNamespace().Name
	etcdService := &corev1.Service{}
	etcdService.Name = fmt.Sprintf("%s-etcd", s.Name)
	etcdService.Namespace = ns
	etcdService.Labels = map[string]string{
		"app": "etcd",
	}
	etcdService.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "etcd",
			Port:       2379,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(2379),
		},
	}
	etcdService.Spec.Selector = map[string]string{
		"app": "etcd",
	}

	apiserverService := &corev1.Service{}
	apiserverService.Name = fmt.Sprintf("%s-apiserver", s.Name)
	apiserverService.Namespace = ns
	apiserverService.Labels = map[string]string{
		"apiserver": "true",
	}
	apiserverService.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "apiserver",
			Port:       443,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(443),
		},
	}
	apiserverService.Spec.Selector = map[string]string{
		"apiserver": "true",
		"app":       "apiserver",
		"api":       s.Name,
	}
	services := []*corev1.Service{etcdService, apiserverService}

	if len(s.DocsImage) > 0 {
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
		services = append(services, docsService)
	}

	return services
}

func (s *ApiserverInstallStrategy) GetAPIServices() []*apiregistrationv1beta1.APIService {
	s.initCert()
	ns := s.GetNamespace()
	result := []*apiregistrationv1beta1.APIService{}
	for _, gv := range s.APIMeta.GetGroupVersions() {
		a := &apiregistrationv1beta1.APIService{}
		a.Name = gv.Version + "." + gv.Group
		a.Labels = map[string]string{
			"api":       s.Name,
			"apiserver": "true",
		}
		a.Spec.Group = gv.Group
		a.Spec.Version = gv.Version
		a.Spec.GroupPriorityMinimum = 2000
		a.Spec.VersionPriority = 10
		a.Spec.Service = &apiregistrationv1beta1.ServiceReference{
			Name:      fmt.Sprintf("%s-apiserver", s.Name),
			Namespace: ns.Name,
		}
		a.Spec.CABundle = s.Certs.CACrt
		result = append(result, a)
	}
	return result
}

func (s *ApiserverInstallStrategy) initCert() {
	s.certOnce.Do(func() {
		srv := fmt.Sprintf("%s-apiserver", s.Name)
		ns := s.GetNamespace().Name
		s.Certs = CreateCerts(srv, ns)
	})
}

func (s *ApiserverInstallStrategy) GetSecrets() []*corev1.Secret {
	s.initCert()
	tls := &corev1.Secret{
		Data: map[string][]byte{
			"tls.crt": s.Certs.ClientCrt,
			"tls.key": s.Certs.ClientKey,
		}}
	tls.Name = "apiserver-certs"
	tls.Namespace = s.GetNamespace().Name
	return []*corev1.Secret{tls}
}
func (s *ApiserverInstallStrategy) GetConfigMaps() []*corev1.ConfigMap { return nil }
func (s *ApiserverInstallStrategy) BeforeInstall() error               { return nil }
func (s *ApiserverInstallStrategy) AfterInstall() error                { return nil }
