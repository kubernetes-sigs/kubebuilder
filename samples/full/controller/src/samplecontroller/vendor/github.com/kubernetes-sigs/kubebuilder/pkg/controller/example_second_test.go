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

package controller_test

import (
	"flag"
	"log"

	"github.com/kubernetes-sigs/kubebuilder/pkg/config"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	corev1 "k8s.io/api/core/v1"
	corev1informer "k8s.io/client-go/informers/core/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
)

func Example_second() {
	// Step 1: Register informers to Watch for Pod events
	flag.Parse()
	informerFactory := config.GetKubernetesInformersOrDie()
	if err := controller.AddInformerProvider(&corev1.Pod{}, informerFactory.Core().V1().Pods()); err != nil {
		log.Fatalf("Could not set informer %v", err)
	}

	// Step 2: Create a new Pod controller to reconcile Pods changes using the default
	// Reconcile function to print messages on events
	podController := &Controller{
		podlister: controller.GetInformerProvider(&corev1.Pod{}).(corev1informer.PodInformer).Lister(),
	}
	genericController := &controller.GenericController{
		Name:      "PodController",
		Reconcile: podController.Reconcile,
	}
	if err := genericController.Watch(&corev1.Pod{}); err != nil {
		log.Fatalf("%v", err)
	}
	controller.AddController(genericController)

	// Step 3: RunInformersAndControllers all informers and controllers
	controller.RunInformersAndControllers(run.CreateRunArguments())
}

type Controller struct {
	podlister corev1lister.PodLister
}

func (c *Controller) Reconcile(key types.ReconcileKey) error {
	pod, err := c.podlister.Pods(key.Namespace).Get(key.Name)
	if err != nil {
		log.Printf("Failed to reconcile Pod %+v\n", err)
		return err
	}
	log.Printf("Reconcile Pod %+v\n", pod)
	return nil
}
