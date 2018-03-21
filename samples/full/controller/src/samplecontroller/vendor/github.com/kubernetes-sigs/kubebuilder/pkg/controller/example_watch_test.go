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
	"k8s.io/api/core/v1"
)

func ExampleGenericController_Watch() {
	// One time setup for program
	flag.Parse()
	informerFactory := config.GetKubernetesInformersOrDie()
	if err := controller.AddInformerProvider(&v1.Pod{}, informerFactory.Core().V1().Pods()); err != nil {
		log.Fatalf("Could not set informer %v", err)
	}

	// Per-controller setup
	c := &controller.GenericController{
		Name: "Foo",
		Reconcile: func(key types.ReconcileKey) error {
			fmt.Printf("Reconciling Pod %s\n", key)
			return nil
		},
	}
	if err := c.Watch(&v1.Pod{}); err != nil {
		log.Fatalf("%v", err)
	}
	controller.AddController(c)

	// One time for program
	controller.RunInformersAndControllers(run.CreateRunArguments())
}
