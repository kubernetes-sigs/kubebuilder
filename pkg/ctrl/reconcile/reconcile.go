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

package reconcile

import (
	"log"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReconcileResult contains the result of a Reconcile.
type ReconcileResult struct {
	// Requeue tells the Controller to requeue the reconcile key
	Requeue bool
}

// ReconcileRequest contains the information necessary to Reconcile a Kubernetes object.  This includes the
// information to uniquely identify the object - its Name and Namespace.  It does NOT contain information about
// any specific Event or the object contents itself.
type ReconcileRequest struct {
	// Name is the name of the object.
	Name string

	// Namespace is the namespace of the object.  Empty for non-namespaced objects.
	Namespace string

	// GroupVersionKind is the GroupVersionKind type of the object to Reconcile.
	GroupVersionKind v1.GroupVersionKind
}

/*
Reconcile implements a Kubernetes API for a specific Resource by Creating, Updating or Deleting Kubernetes
objects, or by making changes to systems external to the cluster (e.g. cloudproviders, github, etc).

Reconcile implementations compare the state specified in an object by a user against the actual cluster state,
and then perform operations to make the actual cluster state reflect the state specified by the user.

Typically, Reconcile is triggered by a Controller in response to cluster Events (e.g. Creating, Updating,
Deleting Kubernetes objects) or external Events (GitHub Webhooks, polling external sources, etc).

Example Reconcile Logic:

	* Read an object and all the Pods it owns.
	* Observe that the object spec specifies 5 replicas but actual cluster contains only 1 Pod replica.
	* Create 4 Pods and set their OwnerReferences to the object.

Reconcile may be implemented as either a type:

	type Reconcile struct {}

	func (Reconcile) Reconcile(ctrl.ReconcileRequest) (ctrl.ReconcileResult, error) {
		// Implement business logic of reading and writing objects here
		return ctrl.ReconcileResult{}, nil
	}

	controller := &ctrl.Controller{Name: "pod-controller", Reconcile: Reconcile{}}


Or as a function:

	controller := &ctrl.Controller{
	  Name: "pod-controller",
	  Reconcile: ctrl.ReconcileFunc(func(o ctrl.ReconcileRequest) (ctrl.ReconcileResult, error) {
		// Implement business logic of reading and writing objects here
		return ctrl.ReconcileResult{}, nil
	  })
	}

Reconciliation is level-based, meaning action isn't driven off changes in individual Events, but instead is
driven by actual cluster state read from the apiserver or a local cache.
For example if responding to a Pod Delete Event, the ReconcileRequest won't contain that a Pod was deleted,
instead the Reconcile function observes this when reading the cluster state and seeing the Pod as missing.
*/
type Reconcile interface {
	// Reconcile performs a full reconciliation for the object referred to by the ReconcileRequest.
	Reconcile(ReconcileRequest) (ReconcileResult, error)
}

// ReconcileFunc is a function that implements the Reconcile interface.
type ReconcileFunc func(ReconcileRequest) (ReconcileResult, error)

var _ Reconcile = ReconcileFunc(func(ReconcileRequest) (ReconcileResult, error) { return ReconcileResult{}, nil })

func (r ReconcileFunc) Reconcile(o ReconcileRequest) (ReconcileResult, error) { return r(o) }

// DefaultReconcileFunc provides a no-op Reconcile implementation for development.
var DefaultReconcileFunc = ReconcileFunc(func(o ReconcileRequest) (ReconcileResult, error) {
	log.Printf("Reconcile called for %v", o)
	return ReconcileResult{}, nil
})
