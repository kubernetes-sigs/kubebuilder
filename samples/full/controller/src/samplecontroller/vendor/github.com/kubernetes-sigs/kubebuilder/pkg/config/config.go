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
	// Check if flag is already set so that the library may be double vendored without crashing the program
	if f := flag.Lookup("kubeconfig"); f == nil {
		flag.StringVar(&kubeconfig, "kubeconfig", "",
			"Path to a kubeconfig. Only required if out-of-cluster.")
	}

	// Check if flag is already set so that the library may be double vendored without crashing the program
	if f := flag.Lookup("master"); f == nil {
		flag.StringVar(&masterURL, "master", "",
			"The address of the Kubernetes API server. Overrides any value in kubeconfig. "+
				"Only required if out-of-cluster.")
	}
}

// GetConfig uses the kubeconfig file at kubeconfig to create a rest.Config for talking to a Kubernetes
// apiserver.  If kubeconfig is empty it will look for kubeconfig in the default locations.
func GetConfig() (*rest.Config, error) {
	if len(kubeconfig) > 0 {
		return clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	} else {
		return rest.InClusterConfig()
	}
}

// GetConfigOrDie uses the kubeconfig file at kubeconfig to create a rest.Config for talking to a Kubernetes
// apiserver.  If kubeconfig is empty it will look for kubeconfig in the default locations.
func GetConfigOrDie() *rest.Config {
	config, err := GetConfig()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return config
}

// GetKubernetesInformersOrDie uses the kubeconfig file at kubeconfig to create a informers.SharedInformerFactory
// for talking to a Kubernetes apiserver.  If kubeconfig is empty it will look for kubeconfig in the
// default locations.
func GetKubernetesInformersOrDie() informers.SharedInformerFactory {
	return informers.NewSharedInformerFactory(kubernetes.NewForConfigOrDie(GetConfigOrDie()), time.Minute*5)
}
