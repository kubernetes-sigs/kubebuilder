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

/*
Package ctrl provides libraries for building Controllers.  Controllers implement Kubernetes APIs
and are central to building Operators, Workload APIs, Configuration APIs, Autoscalers, and more.

Controllers

Controllers are work queues that enqueue work in response to source.Source events (e.g. Pod Create, Update, Delete)
and trigger reconcile.Reconcile functions when the work is dequeued.

Unlike http handlers, Controllers DO NOT perform work directly in response to events, but instead enqueue
ReconcileRequests so the work is performed eventually.

* Controllers run reconcile.Reconcile functions against objects (provided as Name / Namespace).

* Controllers enqueue reconcile.ReconcileRequests in response events provided by source.Sources.

Reconcile

reconcile.Reconcile is a function that may be called at anytime with the Name / Namespace of an
object.  When called, it will ensure that the state of the system matches what is specified in the object at the
time Reconcile is called.

Example: Reconcile is run against a ReplicationController object.  The ReplicationController specifies 5 replicas.
3 Pods exist in the system.  Reconcile creates 2 more Pods and sets their OwnerReference to point at the
ReplicationController.

* Reconcile works on a single object type. - e.g. it will only Reconcile ReplicaSets.

* Reconcile is triggered by a ReconcileRequest containing the Name / Namespace of an object to Reconcile.

* Reconcile does not care about the event contents or event type triggering the ReconcileRequest.
- e.g. it doesn't matter whether a ReplicaSet was created or updated, Reconcile will check that the correct
Pods exist either way.

* Users MUST implement Reconcile themselves.

Source

resource.Source provides a stream of events.  Events may be internal events from watching Kubernetes
APIs (e.g. Pod Create, Update, Delete), or may be synthetic Generic events triggered by cron or WebHooks
(e.g. through a Slackbot or GitHub callback).

Example 1: source.KindSource uses the Kubernetes API Watch endpoint for a GroupVersionKind to provide
Create, Update, Delete events.

Example 2: source.ChannelSource reads Generic events from a channel fed by a WebHook called from a Slackbot.

* Source provides a stream of events for EventHandlers to handle.

* Source may provide either events from Watches (e.g. object Create, Update, Delete) or Generic triggered
from another source (e.g. WebHook callback).

* Users SHOULD use the provided Source implementations instead of implementing their own for nearly all cases.

EventHandler

eventhandler.EventHandler transforms and enqueues events from a source.Source into reconcile.ReconcileRequests.

Example: a Pod Create event from a Source is provided to the eventhandler.EnqueueHandler, which enqueues a
ReconcileRequest containing the Name / Namespace of the Pod.

* EventHandler takes an event.Event and enqueues ReconcileRequests

* EventHandlers MAY map an event for an object of one type to a ReconcileRequest for an object of another type.

* EventHandlers MAY map an event for an object to multiple ReconcileRequests for different objects.

* Users SHOULD use the provided EventHandler implementations instead of implementing their own for almost all cases.

Predicate

predicate.Predicate allows events to be filtered before they are given to EventHandlers.  This allows common
filters to be reused and composed together with EventHandlers.

* Predicate takes and event.Event and returns a bool (true to enqueue)

* Predicates are optional

* Users SHOULD use the provided Predicate implementations, but MAY implement their own Predicates as needed.

PodController Diagram

Source provides event:

* source.KindSource{"core", "v1", "Pod"} -> (Pod foo/bar Create Event)

EventHandler enqueues ReconcileRequest:

* eventhandler.Enqueue{} -> (ReconcileRequest{"foo", "bar"})

Reconcile is called with the ReconcileRequest:

* Reconcile(ReconcileRequest{"foo", "bar"})


ControllerManager

ControllerManager registers and starts Controllers.  It initializes shared dependencies - such as clients, caches,
stop channels, etc and provides these to the Controllers that it manages.  ControllerManager should be used
anytime multiple Controllers exist within the same program.

Usage

Controllers should live in separate packages from the main program.  A single program may contain multiple
Controllers that share local caches and clients.

Step 1: Create a main that uses the ControllerManager to start the registered Controllers.

	pkg main

	import (
	  "flag"
	  "log"

	  "github.com/kubernetes-sigs/kubebuilder/pkg/ctrl"

	  _ "pkg/controller/mycontroller"
	)

	func main() {
	  flag.Parse()
	  log.Fatal(ctrl.Start())
	}

Step 2: Create a Controller in the package init function and register it with the ControllerManager.

	pkg mycontroller

	func init() {
	  controller := &ctrl.Controller{Name: "myresource-controller", Reconcile: Reconcile{})}
	  ctrl.Register(controller)

	  // Watch for changes to MyKind objects, and enqueues a ReconcileRequest with the Name and Namespace of the object.
	  controller.Watch(
	    source.KindSource{Group: "mygroup", Version: "myversion", Kind: "MyKind"},
	    eventhandler.Enqueue{},
	  )
	}

	// MyResourceReconciler implements the MyResource API
	type MyResourceReconciler struct{}

	// Reconcile handles ReconcileRequests to read MyResource objects and then makes changes in the cluster by
	// creating, updating and deleting other objects.
	func (MyResourceReconciler) Reconcile(r reconcile.ReconcileRequest) (reconcile.ReconcileResult, error) {
	  // Your business logic goes here.
	  return reconcile.ReconcileResult{}, nil
	}

Controller Example - Deployment

1. Watch Deployment, ReplicaSet, Pod Sources

1.1 Deployments -> eventhandler.EnqueueHandler - enqueue the Deployment object key.

1.2 ReplicaSets (created by Deployments) -> eventhandler.EnqueueOwnerHandler - enqueue the Owning Deployment key.

1.3 Pods (created by ReplicaSets) -> eventhandler.EnqueueOwnerHandler -> enqueue owning Deployment
key (transitive through ReplicaSet).

2. Reconcile Deployment

2.1 Deployment object created -> Read Deployment, try to read ReplicaSet, see if is missing create ReplicaSet.

2.2 Reconcile triggered by creation of ReplicaSet and Pods -> Read Deployment and ReplicaSet, do nothing.

Watching and EventHandling

Controllers may Watch multiple Kinds of objects (e.g. Pods, ReplicaSets and Deployments), but they should
enqueue keys for only a single Kind.  When one Kind of object must be be updated in response to changes
in another Kind of object, an EnqueueMappedHandler may be used to Reconcile the Kind that is being
updated and watch the other Kind for Events.  e.g. Respond to a cluster resize
Event (add / delete Node) by re-reconciling all instances of another Kind that cares about the cluster size.

For example, a Deployment Controller might use an EnqueueHandler and EnqueueOwnerHandler to:

* Watch for Deployment Events - enqueue the key of the Deployment.

* Watch for ReplicaSet Events - enqueue the key of the Deployment that created the ReplicaSet (owns directly)

* Watch for Pod Events - enqueue the key of the Deployment that created the Pod (owns transitively through a ReplicaSet).

Note: ReconcileRequests are deduplicated when they are enqueued.  Many Pod Events for the same Deployment
may trigger only 1 Reconcile invocation as each Event results in the Handler trying to enqueue
the same ReconcileRequest for the Deployment.

Controller Writing Tips

Reconcile Runtime Complexity:

* It is better to write Controllers to perform an O(1) Reconcile N times (e.g. on N different objects) instead of
performing an O(N) Reconcile 1 time (e.g. on a single object which manages N other objects).

* Example: If you need to update all Services in response to a Node being added - Reconcile Services but Watch
Node events (transformed to Service object Name / Namespaces) instead of Reconciling the Node and updating all
Services from a single Reconcile.

Event Multiplexing:

* ReconcileRequests for the same Name / Namespace are deduplicated when they are enqueued.  This allows
for Controllers to gracefully handle event storms for a single object.  Multiplexing multiple event Sources to
a single object type takes advantage of this.

* Example: Pod events for a ReplicaSet are transformed to a ReplicaSet Name / Namespace, so the ReplicaSet
will be Reconciled only 1 time for multiple Pods.
*/
package ctrl
