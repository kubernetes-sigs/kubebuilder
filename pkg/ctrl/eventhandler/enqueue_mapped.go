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

package eventhandler

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/event"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/reconcile"
	"k8s.io/client-go/util/workqueue"
)

var _ EventHandler = EnqueueMappedHandler{}

// EnqueueMappedHandler enqueues ReconcileRequests resulting from running a user provided transformation
// function on the Event.
type EnqueueMappedHandler struct {
	// ToRequests transforms the argument into a slice of keys to be reconciled
	ToRequests ToRequests
}

// Create implements EventHandler
func (e EnqueueMappedHandler) Create(q workqueue.RateLimitingInterface, event event.CreateEvent) {}

// Update implements EventHandler
func (e EnqueueMappedHandler) Update(q workqueue.RateLimitingInterface, event event.UpdateEvent) {}

// Delete implements EventHandler
func (e EnqueueMappedHandler) Delete(q workqueue.RateLimitingInterface, event event.DeleteEvent) {}

// Generic implements EventHandler
func (e EnqueueMappedHandler) Generic(workqueue.RateLimitingInterface, event.GenericEvent) {}

// ToRequests maps an object to a collection of keys to be enqueued
type ToRequests interface {
	Map(interface{}) []reconcile.ReconcileRequest
}

var _ ToRequests = ToRequestsFunc(func(interface{}) []reconcile.ReconcileRequest { return nil })

// ToRequestsFunc implements ToRequests using a function.
type ToRequestsFunc func(interface{}) []reconcile.ReconcileRequest

func (m ToRequestsFunc) Map(i interface{}) []reconcile.ReconcileRequest {
	return m(i)
}
