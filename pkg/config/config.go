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

package config

import (
	"flag"
	"log"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig, masterURL string
)

func init() {
	// TODO: Fix this to allow double vendoring this library but still register flags on behalf of users
	flag.StringVar(&kubeconfig, "kubeconfig", "",
		"Path to a kubeconfig. Only required if out-of-cluster.")

	flag.StringVar(&masterURL, "master", "",
		"The address of the Kubernetes API server. Overrides any value in kubeconfig. "+
			"Only required if out-of-cluster.")
}

// GetConfig creates a *rest.Config for talking to a Kubernetes apiserver.
// If --kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
// in cluster and use the cluster provided kubeconfig.
//
// Will log.Fatal if KubernetesInformers cannot be created
func GetConfig() (*rest.Config, error) {
	if len(kubeconfig) > 0 {
		return clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	} else {
		return rest.InClusterConfig()
	}
}

// GetConfig creates a *rest.Config for talking to a Kubernetes apiserver.
// If --kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
// in cluster and use the cluster provided kubeconfig.
func GetConfigOrDie() *rest.Config {
	config, err := GetConfig()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return config
}

// GetKubernetesClientSet creates a *kubernetes.ClientSet for talking to a Kubernetes apiserver.
// If --kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
// in cluster and use the cluster provided kubeconfig.
func GetKubernetesClientSet() (*kubernetes.Clientset, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// GetKubernetesClientSetOrDie creates a *kubernetes.ClientSet for talking to a Kubernetes apiserver.
// If --kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
// in cluster and use the cluster provided kubeconfig.
//
// Will log.Fatal if KubernetesInformers cannot be created
func GetKubernetesClientSetOrDie() (*kubernetes.Clientset, error) {
	cs, err := GetKubernetesClientSet()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return cs, nil
}

// GetKubernetesInformers creates a informers.SharedInformerFactory for talking to a Kubernetes apiserver.
// If --kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
// in cluster and use the cluster provided kubeconfig.
func GetKubernetesInformers() (informers.SharedInformerFactory, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}
	i, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return informers.NewSharedInformerFactory(i, time.Minute*5), nil
}

// GetKubernetesInformers creates a informers.SharedInformerFactory for talking to a Kubernetes apiserver.
// If --kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
// in cluster and use the cluster provided kubeconfig.
//
// Will log.Fatal if KubernetesInformers cannot be created
func GetKubernetesInformersOrDie() informers.SharedInformerFactory {
	i, err := GetKubernetesInformers()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return i
}
