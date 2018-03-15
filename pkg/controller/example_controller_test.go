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
	"fmt"
	"log"

	"github.com/kubernetes-sigs/kubebuilder/pkg/config"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func ExampleGenericController() {
	// Step 1: Register informers with the ControllerManager to Watch for Pod and ReplicaSet events
	flag.Parse()
	informerFactory := config.GetKubernetesInformersOrDie()
	if err := controller.AddInformerProvider(&corev1.Pod{}, informerFactory.Core().V1().Pods()); err != nil {
		log.Fatalf("Could not set informer %v", err)
	}
	if err := controller.AddInformerProvider(&appsv1.ReplicaSet{}, informerFactory.Apps().V1().ReplicaSets()); err != nil {
		log.Fatalf("Could not set informer %v", err)
	}

	// Step 3.1: Create a new Pod controller to reconcile Pods changes
	podController := &controller.GenericController{
		Reconcile: func(key types.ReconcileKey) error {
			fmt.Printf("Reconciling Pod %v\n", key)
			return nil
		},
	}
	if err := podController.Watch(&corev1.Pod{}); err != nil {
		log.Fatalf("%v", err)
	}
	controller.AddController(podController)

	// Step 3.2: Create a new ReplicaSet controller to reconcile ReplicaSet changes
	rsController := &controller.GenericController{
		Reconcile: func(key types.ReconcileKey) error {
			fmt.Printf("Reconciling ReplicaSet %v\n", key)
			return nil
		},
	}
	if err := rsController.WatchAndMapToController(&corev1.Pod{}); err != nil {
		log.Fatalf("%v", err)
	}
	if err := rsController.Watch(&appsv1.ReplicaSet{}); err != nil {
		log.Fatalf("%v", err)
	}
	controller.AddController(rsController)

	// Step 4: RunInformersAndControllers all informers and controllers
	controller.RunInformersAndControllers(run.CreateRunArguments())
}
