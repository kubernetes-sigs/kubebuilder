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
	"reflect"
	"time"

	apiextv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1beta1"
)

type installer struct {
	rbac   rbacv1client.RbacV1Interface
	core   corev1client.CoreV1Interface
	apiext apiextv1beta1client.CustomResourceDefinitionInterface
	apps   appsv1client.AppsV1Interface
	apireg apiregistrationv1beta1.APIServiceInterface
}

// NewInstaller returns a new installer
func NewInstaller(config *rest.Config) *installer {
	cs := kubernetes.NewForConfigOrDie(config)
	ae := apiextv1beta1client.NewForConfigOrDie(config)
	ar := apiregistrationv1beta1.NewForConfigOrDie(config)
	return &installer{
		cs.RbacV1(),
		cs.CoreV1(),
		ae.CustomResourceDefinitions(),
		cs.AppsV1(),
		ar.APIServices(),
	}
}

// Install installs the components provided by the InstallStrategy
func (i *installer) Install(strategy InstallStrategy) error {
	if err := strategy.BeforeInstall(); err != nil {
		return err
	}
	if err := i.installCrds(strategy); err != nil {
		return err
	}
	if err := i.installNamespace(strategy); err != nil {
		return err
	}
	if err := i.installServiceAccount(strategy); err != nil {
		return err
	}
	if err := i.installClusterRole(strategy); err != nil {
		return err
	}
	if err := i.installClusterRoleBinding(strategy); err != nil {
		return err
	}
	if err := i.installSecrets(strategy); err != nil {
		return err
	}
	if err := i.installConfigMaps(strategy); err != nil {
		return err
	}
	if err := i.installDeployments(strategy); err != nil {
		return err
	}
	if err := i.installStatefulSets(strategy); err != nil {
		return err
	}
	if err := i.installAPIServices(strategy); err != nil {
		return err
	}
	if err := i.installServices(strategy); err != nil {
		return err
	}
	if err := strategy.AfterInstall(); err != nil {
		return err
	}
	log.Printf("Finished installing.")
	return nil
}

func (i *installer) installCrds(strategy InstallStrategy) error {
	if len(strategy.GetCRDs()) == 0 {
		return nil
	}
	for _, crd := range strategy.GetCRDs() {
		value, err := i.apiext.Get(crd.Name, v1.GetOptions{})
		// Create case
		if err != nil || value == nil {
			log.Printf("Creating CRD %v\n", crd.Name)
			_, err = i.apiext.Create(crd)
			if err != nil {
				return err
			}
			continue
		}
		// Update case
		if !reflect.DeepEqual(value.Spec, crd.Spec) {
			log.Printf("Updating CRD %v\n", crd.Name)
			value.Spec = crd.Spec
			_, err = i.apiext.Update(value)
			if err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

func (i *installer) installNamespace(strategy InstallStrategy) error {
	ns := strategy.GetNamespace()
	if ns == nil {
		return nil
	}
	if foundNs, err := i.core.Namespaces().Get(ns.Name, v1.GetOptions{}); err != nil || foundNs == nil {
		log.Printf("Creating Namespace %v\n", ns.Name)
		_, err = i.core.Namespaces().Create(ns)
		return err
	}
	return nil
}

func (i *installer) installServiceAccount(strategy InstallStrategy) error {
	accountName := strategy.GetServiceAccount()
	if len(accountName) == 0 {
		return nil
	}
	ns := strategy.GetNamespace()
	var ret error
	// Wait for the namespace SA to be created
	for cnt := 0; cnt < 5; cnt++ {
		sa, err := i.core.ServiceAccounts(ns.Name).Get(accountName, v1.GetOptions{})
		ret = err
		if err == nil || sa != nil {
			break
		}
		time.Sleep(time.Second * 2)
	}
	return ret
}

func (i *installer) installClusterRole(strategy InstallStrategy) error {
	role := strategy.GetClusterRole()
	if role == nil {
		return nil
	}
	// Create case
	if foundRole, err := i.rbac.ClusterRoles().Get(role.Name, v1.GetOptions{}); err == nil && foundRole != nil {
		log.Printf("Updating ClusterRole %v\n", role.Name)
		foundRole.Rules = role.Rules
		_, err = i.rbac.ClusterRoles().Update(foundRole)
		return err
	}
	// Update case
	log.Printf("Creating ClusterRole %v\n", role.Name)
	_, err := i.rbac.ClusterRoles().Create(role)
	return err
}

func (i *installer) installClusterRoleBinding(strategy InstallStrategy) error {
	rolebinding := strategy.GetClusterRoleBinding()
	if rolebinding == nil {
		return nil
	}
	if foundBinding, err := i.rbac.ClusterRoleBindings().Get(rolebinding.Name, v1.GetOptions{}); err != nil || foundBinding == nil {
		log.Printf("Creating ClusterRoleBinding %v\n", rolebinding.Name)
		_, err = i.rbac.ClusterRoleBindings().Create(rolebinding)
		return err
	}
	return nil
}

func (i *installer) installDeployments(strategy InstallStrategy) error {
	ns := strategy.GetNamespace()
	for _, deployment := range strategy.GetDeployments() {
		if foundDeployment, err := i.apps.Deployments(ns.Name).Get(deployment.Name, v1.GetOptions{}); err == nil && foundDeployment != nil {
			log.Printf("Updating Deployment %v\n", deployment.Name)
			foundDeployment.Spec = deployment.Spec
			_, err := i.apps.Deployments(ns.Name).Update(foundDeployment)
			return err
		}
		log.Printf("Creating Deployment %v\n", deployment.Name)
		if _, err := i.apps.Deployments(ns.Name).Create(deployment); err != nil {
			return err
		}
	}
	return nil
}

func (i *installer) installStatefulSets(strategy InstallStrategy) error {
	ns := strategy.GetNamespace()
	for _, statefulSet := range strategy.GetStatefulSets() {
		if found, err := i.apps.StatefulSets(ns.Name).Get(statefulSet.Name, v1.GetOptions{}); err == nil && found != nil {
			log.Printf("Updating StatefulSet %v\n", statefulSet.Name)
			found.Spec = statefulSet.Spec
			_, err := i.apps.StatefulSets(ns.Name).Update(found)
			return err
		}
		log.Printf("Creating StatefulSet %v\n", statefulSet.Name)
		if _, err := i.apps.StatefulSets(ns.Name).Create(statefulSet); err != nil {
			return err
		}
	}
	return nil
}

func (i *installer) installServices(strategy InstallStrategy) error {
	ns := strategy.GetNamespace()
	for _, service := range strategy.GetServices() {
		if found, err := i.core.Services(ns.Name).Get(service.Name, v1.GetOptions{}); err == nil && found != nil {
			log.Printf("Updating Service %v\n", service.Name)
			found.Spec = service.Spec
			_, err := i.core.Services(ns.Name).Update(found)
			return err
		}
		log.Printf("Creating Service %v\n", service.Name)
		if _, err := i.core.Services(ns.Name).Create(service); err != nil {
			return err
		}
	}
	return nil
}

func (i *installer) installSecrets(strategy InstallStrategy) error {
	ns := strategy.GetNamespace()
	for _, secret := range strategy.GetSecrets() {
		if found, err := i.core.Secrets(ns.Name).Get(secret.Name, v1.GetOptions{}); err == nil && found != nil {
			log.Printf("Updating Secret %v\n", secret.Name)
			found.Data = secret.Data
			_, err := i.core.Secrets(ns.Name).Update(found)
			return err
		}
		log.Printf("Creating Secret %v\n", secret.Name)
		if _, err := i.core.Secrets(ns.Name).Create(secret); err != nil {
			return err
		}
	}
	return nil
}

func (i *installer) installConfigMaps(strategy InstallStrategy) error {
	ns := strategy.GetNamespace()
	for _, configmap := range strategy.GetConfigMaps() {
		if found, err := i.core.ConfigMaps(ns.Name).Get(configmap.Name, v1.GetOptions{}); err == nil && found != nil {
			log.Printf("Updating ConfigMap %v\n", configmap.Name)
			found.Data = configmap.Data
			_, err := i.core.ConfigMaps(ns.Name).Update(found)
			return err
		}
		log.Printf("Creating ConfigMap %v\n", configmap.Name)
		if _, err := i.core.ConfigMaps(ns.Name).Create(configmap); err != nil {
			return err
		}
	}
	return nil
}

func (i *installer) installAPIServices(strategy InstallStrategy) error {
	for _, apiservice := range strategy.GetAPIServices() {
		if found, err := i.apireg.Get(apiservice.Name, v1.GetOptions{}); err == nil && found != nil {
			log.Printf("Updating ApiService %v\n", apiservice.Name)
			found.Spec = apiservice.Spec
			_, err := i.apireg.Update(found)
			return err
		}
		log.Printf("Creating ApiService %v\n", apiservice.Name)
		if _, err := i.apireg.Create(apiservice); err != nil {
			return err
		}
	}
	return nil
}
