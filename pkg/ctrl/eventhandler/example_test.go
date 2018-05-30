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

package eventhandler_test

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/event"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/eventhandler"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/reconcile"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/source"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
)

var c = &ctrl.Controller{Name: "pod-controller"}

// This example watches Pods and enqueues ReconcileRequests with the Name and Namespace of the Pod from
// the Event (i.e. change caused by a Create, Update, Delete).
func ExampleEnqueueHandler() {
	// c is a ctrl.Controller
	c.Watch(
		source.KindSource{Group: "core", Version: "v1", Kind: "Pod"},
		eventhandler.EnqueueHandler{},
	)
}

// This example watches ReplicaSets and enqueues a ReconcileRequest containing the Name and Namespace of the
// owning (direct) Deployment responsible for the creation of the ReplicaSet.
func ExampleEnqueueOwnerHandler_1() {
	// c is a ctrl.Controller
	c.Watch(
		source.KindSource{Group: "core", Version: "v1", Kind: "ReplicaSet"},
		eventhandler.EnqueueOwnerHandler{
			OwnerType:    v1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			IsController: true,
		},
	)
}

// This example watches Pods and enqueues a ReconcileRequest containing the Name and Namespace of the
// owning (transitive) Deployment responsible for the creation of the Pod.
func ExampleEnqueueOwnerHandler_2() {
	// c is a ctrl.Controller
	c.Watch(
		source.KindSource{Group: "core", Version: "v1", Kind: "Pod"},
		eventhandler.EnqueueOwnerHandler{
			OwnerType:        v1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			TransitiveOwners: true,
			IsController:     true,
		},
	)
}

// This example watches Deployments and enqueues a ReconcileRequest contain the Name and Namespace of different
// objects (of Kind: MyKind) using a mapping function defined by the user.
func ExampleEnqueueMappedHandler() {
	// c is a ctrl.Controller
	c.Watch(
		source.KindSource{Group: "apps", Version: "v1", Kind: "Deployment"},
		eventhandler.EnqueueMappedHandler{
			ToRequests: eventhandler.ToRequestsFunc(func(i interface{}) []reconcile.ReconcileRequest {
				d := i.(*appsv1.Deployment)
				return []reconcile.ReconcileRequest{
					{GroupVersionKind: v1.GroupVersionKind{Group: "mygroup", Version: "myversion", Kind: "MyKind"},
						Name:      d.Name + "-1",
						Namespace: d.Namespace,
					},
					{GroupVersionKind: v1.GroupVersionKind{Group: "mygroup", Version: "myversion", Kind: "MyKind"},
						Name:      d.Name + "-2",
						Namespace: d.Namespace,
					},
				}
			}),
		})
}

// This example implements eventhandler.EnqueueHandler.
func ExampleEventHandlerFunc() {
	// c is a ctrl.Controller
	c.Watch(
		source.KindSource{Group: "core", Version: "v1", Kind: "Pod"},
		eventhandler.EventHandlerFuncs{
			CreateFunc: func(q workqueue.RateLimitingInterface, e event.CreateEvent) {
				q.Add(reconcile.ReconcileRequest{
					Name:      e.Meta.Name,
					Namespace: e.Meta.Namespace,
				})
			},
			UpdateFunc: func(q workqueue.RateLimitingInterface, e event.UpdateEvent) {
				q.Add(reconcile.ReconcileRequest{
					Name:      e.MetaNew.Name,
					Namespace: e.MetaNew.Namespace,
				})
			},
			DeleteFunc: func(q workqueue.RateLimitingInterface, e event.DeleteEvent) {
				q.Add(reconcile.ReconcileRequest{
					Name:      e.Meta.Name,
					Namespace: e.Meta.Namespace,
				})
			},
			GenericFunc: func(q workqueue.RateLimitingInterface, e event.GenericEvent) {
				q.Add(reconcile.ReconcileRequest{
					Name:      e.Meta.Name,
					Namespace: e.Meta.Namespace,
				})
			},
		},
	)
}
