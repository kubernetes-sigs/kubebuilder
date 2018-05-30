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

package ctrl

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/eventhandler"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/predicate"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/reconcile"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/source"
)

// Controllers are work queues that watch for changes to objects (i.e. Create / Update / Delete events) and
// then Reconcile an object (i.e. make changes to ensure the system state matches what is specified in the object).
type Controller struct {
	// Name is used to uniquely identify a Controller in tracing, logging and monitoring.  Name is required.
	Name string

	// Reconcile is a function that can be called at any time with the Name / Namespace of an object and
	// ensures that the state of the system matches the state specified in the object.
	// Defaults to the DefaultReconcileFunc.
	Reconcile reconcile.Reconcile

	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 1.
	MaxConcurrentReconciles int

	// Stop is used to shutdown the Reconcile.  Defaults to a new channel.
	Stop <-chan struct{}
}

// Watch takes events provided by a Source and uses the EventHandler to enqueue ReconcileRequests in
// response to the events.
//
// Watch may be provided one or more Predicates to filter events before they are given to the EventHandler.
// Events will be passed to the EventHandler iff all provided Predicates evaluate to true.
func (*Controller) Watch(source.Source, eventhandler.EventHandler, ...predicate.Predicate) {
	// TODO: Write this
}

// Start starts the Controller.  Start blocks until the Stop channel is closed.
func (*Controller) Start() error {
	// TODO: Write this
	return nil
}
