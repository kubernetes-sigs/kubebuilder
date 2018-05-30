/*
Copyright 2018 The Kubernetes Authors.

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

package predicate_test

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/event"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/eventhandler"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/predicate"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/source"
	appsv1 "k8s.io/api/apps/v1"
)

// This example filters Deployment events using a Predicate that drops Update Events where the Deployment
// Generation has not changed.
func ExamplePredicateFunc() {
	controller := &ctrl.Controller{Name: "deployment-controller"}

	controller.Watch(
		source.KindSource{Group: "apps", Version: "v1", Kind: "Deployment"},
		eventhandler.EnqueueHandler{},
		predicate.PredicateFuncs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oDep := &appsv1.Deployment{}
				e.UnmarshalObjOld(oDep)

				nDep := &appsv1.Deployment{}
				e.UnmarshalObjNew(nDep)

				return oDep.Generation != nDep.Generation
			},
		},
	)
}
