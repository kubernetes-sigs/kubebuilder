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
	"log"

	apiextv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1beta1"
)

type uninstaller struct {
	rbac   rbacv1client.RbacV1Interface
	core   corev1client.CoreV1Interface
	apiext apiextv1beta1client.CustomResourceDefinitionInterface
	apps   appsv1client.AppsV1Interface
	apireg apiregistrationv1beta1.APIServiceInterface
}

// NewUninstaller returns a new uninstaller
func NewUninstaller(config *rest.Config) *uninstaller {
	cs := kubernetes.NewForConfigOrDie(config)
	ae := apiextv1beta1client.NewForConfigOrDie(config)
	ar := apiregistrationv1beta1.NewForConfigOrDie(config)
	return &uninstaller{
		cs.RbacV1(),
		cs.CoreV1(),
		ae.CustomResourceDefinitions(),
		cs.AppsV1(),
		ar.APIServices(),
	}
}

// Uninstall uninstalls the components installed by the InstallStrategy
func (i *uninstaller) Uninstall(strategy InstallStrategy) error {
	if err := i.uninstallNamespace(strategy); err != nil {
		return err
	}
	if err := i.uninstallClusterRole(strategy); err != nil {
		return err
	}
	if err := i.uninstallClusterRoleBinding(strategy); err != nil {
		return err
	}
	if err := i.uninstallCrds(strategy); err != nil {
		return err
	}
	if err := i.uninstallAPIServices(strategy); err != nil {
		return err
	}
	log.Printf("Finished uninstalling.")
	return nil
}

func (i *uninstaller) uninstallCrds(strategy InstallStrategy) error {
	if len(strategy.GetCRDs()) == 0 {
		return nil
	}
	for _, crd := range strategy.GetCRDs() {
		value, err := i.apiext.Get(crd.Name, v1.GetOptions{})
		if err == nil && value != nil {
			if err = i.apiext.Delete(crd.Name, &v1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *uninstaller) uninstallNamespace(strategy InstallStrategy) error {
	ns := strategy.GetNamespace()
	if ns == nil {
		return nil
	}
	if foundNs, err := i.core.Namespaces().Get(ns.Name, v1.GetOptions{}); err == nil && foundNs != nil {
		return i.core.Namespaces().Delete(ns.Name, &v1.DeleteOptions{})
	}
	return nil
}

func (i *uninstaller) uninstallClusterRole(strategy InstallStrategy) error {
	role := strategy.GetClusterRole()
	if role == nil {
		return nil
	}
	if foundRole, err := i.rbac.ClusterRoles().Get(role.Name, v1.GetOptions{}); err == nil && foundRole != nil {
		return i.rbac.ClusterRoles().Delete(foundRole.Name, &v1.DeleteOptions{})
	}
	return nil
}

func (i *uninstaller) uninstallClusterRoleBinding(strategy InstallStrategy) error {
	rolebinding := strategy.GetClusterRoleBinding()
	if rolebinding == nil {
		return nil
	}
	if foundBinding, err := i.rbac.ClusterRoleBindings().Get(rolebinding.Name, v1.GetOptions{}); err == nil && foundBinding != nil {
		return i.rbac.ClusterRoleBindings().Delete(rolebinding.Name, &v1.DeleteOptions{})
	}
	return nil
}

func (i *uninstaller) uninstallAPIServices(strategy InstallStrategy) error {
	for _, apiservice := range strategy.GetAPIServices() {
		if found, err := i.apireg.Get(apiservice.Name, v1.GetOptions{}); err == nil && found != nil {
			if err = i.apireg.Delete(found.Name, &v1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}
