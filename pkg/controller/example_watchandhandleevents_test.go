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
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/handlefunctions"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func ExampleGenericController_WatchAndHandleEvents() {
	// One time setup for program
	flag.Parse()
	informerFactory := config.GetKubernetesInformersOrDie()
	if err := controller.AddInformerProvider(&corev1.Pod{}, informerFactory.Core().V1().Pods()); err != nil {
		log.Fatalf("Could not set informer %v", err)

	}

	// Per-controller setup
	c := &controller.GenericController{
		Reconcile: func(key types.ReconcileKey) error {
			fmt.Printf("Reconciling Pod %s\n", key)
			return nil
		},
	}
	err := c.WatchAndHandleEvents(&corev1.Pod{},
		// This function returns the callbacks that will be invoked for events
		func(q workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
			// This function implements the same functionality as GenericController.Watch
			return cache.ResourceEventHandlerFuncs{
				AddFunc:    func(obj interface{}) { q.AddRateLimited(handlefunctions.MapToSelf(obj)) },
				UpdateFunc: func(old, obj interface{}) { q.AddRateLimited(handlefunctions.MapToSelf(obj)) },
				DeleteFunc: func(obj interface{}) { q.AddRateLimited(handlefunctions.MapToSelf(obj)) },
			}
		})
	if err != nil {
		log.Fatalf("%v", err)
	}
	controller.AddController(c)

	// One time for program
	controller.RunInformersAndControllers(run.CreateRunArguments())
}
