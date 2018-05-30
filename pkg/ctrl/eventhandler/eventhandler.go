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
	"k8s.io/client-go/util/workqueue"
)

// EventHandler enqueues ReconcileRequests in response to events (e.g. Pod Create).  EventHandlers map an Event
// for one object to trigger Reconciles for either the same object or different objects - e.g. if there is an
// Event for object with type Foo (using source.KindSource) then Reconcile one or more object(s) with type Bar.
//
// Identical ReconcileRequests will be batched together through the queuing mechanism before Reconcile is called.
//
// * Use EnqueueHandler to Reconcile the object the event is for
// - do this for events for the type the Controller Reconciles. (e.g. Deployment for a Deployment Controller)
//
// * Use EnqueueOwnerHandler to Reconcile the owner of the object the event is for
// - do this for events for the types the Controller creates.  (e.g. ReplicaSets created by a Deployment Controller)
//
// * Use EnqueueMappendHandler to transform an event for an object to a Reconcile of an object
// of a different type - do this for events for types the Controller may be interested in, but doesn't create.
// (e.g. If Foo responds to cluster size events, map Node events to Foo objects.)
//
// Unless you are implementing your own EventHandler, you can ignore the functions on the EventHandler interface.
// Most users shouldn't need to implement their own EventHandler.
type EventHandler interface {
	// Create is called in response to an create event - e.g. Pod Creation.
	Create(workqueue.RateLimitingInterface, event.CreateEvent)

	// Update is called in response to an update event -  e.g. Pod Updated.
	Update(workqueue.RateLimitingInterface, event.UpdateEvent)

	// Delete is called in response to a delete event - e.g. Pod Deleted.
	Delete(workqueue.RateLimitingInterface, event.DeleteEvent)

	// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
	// external trigger request - e.g. Reconcile Autoscaling, or a Webhook.
	Generic(workqueue.RateLimitingInterface, event.GenericEvent)
}

var _ EventHandler = EmptyEventHandler{}

// EmptyEventHandler implements EventHandler with no-op implementations.  EmptyEventHandler may be embedded
// to only implement handling a subset of Event types.
type EmptyEventHandler struct{}

func (EmptyEventHandler) Create(workqueue.RateLimitingInterface, event.CreateEvent)   {}
func (EmptyEventHandler) Update(workqueue.RateLimitingInterface, event.UpdateEvent)   {}
func (EmptyEventHandler) Delete(workqueue.RateLimitingInterface, event.DeleteEvent)   {}
func (EmptyEventHandler) Generic(workqueue.RateLimitingInterface, event.GenericEvent) {}

// EventHandlerFuncs allows specifying a subset of EventHandler functions are fields.
type EventHandlerFuncs struct {
	// Create is called in response to an add event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.ReconcileRequests.
	CreateFunc func(workqueue.RateLimitingInterface, event.CreateEvent)

	// Update is called in response to an update event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.ReconcileRequests.
	UpdateFunc func(workqueue.RateLimitingInterface, event.UpdateEvent)

	// Delete is called in response to a delete event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.ReconcileRequests.
	DeleteFunc func(workqueue.RateLimitingInterface, event.DeleteEvent)

	// GenericFunc is called in response to a generic event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.ReconcileRequests.
	GenericFunc func(workqueue.RateLimitingInterface, event.GenericEvent)
}

func (h EventHandlerFuncs) Create(q workqueue.RateLimitingInterface, e event.CreateEvent) {
	if h.CreateFunc != nil {
		h.CreateFunc(q, e)
	}
}

func (h EventHandlerFuncs) Delete(q workqueue.RateLimitingInterface, e event.DeleteEvent) {
	if h.DeleteFunc != nil {
		h.DeleteFunc(q, e)
	}
}

func (h EventHandlerFuncs) Update(q workqueue.RateLimitingInterface, e event.UpdateEvent) {
	if h.UpdateFunc != nil {
		h.UpdateFunc(q, e)
	}
}

func (h EventHandlerFuncs) Generic(q workqueue.RateLimitingInterface, e event.GenericEvent) {
	if h.GenericFunc != nil {
		h.GenericFunc(q, e)
	}
}
