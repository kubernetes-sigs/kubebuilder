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

package eventhandlers

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/predicates"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// EventHandler accepts a workqueue and returns ResourceEventHandlerFuncs that enqueue messages to it
// for add / update / delete events
type EventHandler interface {
	Get(r workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs
}

// MapAndEnqueue provides Fns to map objects to name/namespace keys and enqueue them as messages
type MapAndEnqueue struct {
	Predicates []predicates.Predicate
	// Map maps an object to a key that can be enqueued
	Map func(interface{}) string
}

// Get returns ResourceEventHandlerFuncs that Map an object to a Key and enqueue the key if it is non-empty
func (mp MapAndEnqueue) Get(r workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
	// Enqueue the mapped key for updates to the object
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			for _, p := range mp.Predicates {
				if !p.HandleCreate(obj) {
					return
				}
			}
			mp.addRateLimited(r, obj)
		},
		UpdateFunc: func(old, obj interface{}) {
			for _, p := range mp.Predicates {
				if !p.HandleUpdate(old, obj) {
					return
				}
			}
			mp.addRateLimited(r, obj)
		},
		DeleteFunc: func(obj interface{}) {
			for _, p := range mp.Predicates {
				if !p.HandleDelete(obj) {
					return
				}
			}
			mp.addRateLimited(r, obj)
		},
	}
}

// addRateLimited maps the obj to a string.  If the string is non-empty, it is enqueued.
func (mp MapAndEnqueue) addRateLimited(r workqueue.RateLimitingInterface, obj interface{}) {
	k := mp.Map(obj)
	if len(k) > 0 {
		r.AddRateLimited(k)
	}
}

type ControllerLookup func(types.ReconcileKey) (interface{}, error)

type Path []ControllerLookup

type MapToController struct {
	Path Path
}

// MapToController returns the namespace/name key of the controller for obj
func (m MapToController) Map(obj interface{}) string {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return ""
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return ""
		}
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	glog.V(4).Infof("Processing object: %s", object.GetName())
	// Walk the controller path to the root
	o := object
	for len(m.Path) > 0 {
		// Get the owner reference
		if ownerRef := metav1.GetControllerOf(o); ownerRef != nil {
			// Resolve the owner object and check if the UID of the looked up object matches the reference.
			owner, err := m.Path[0](types.ReconcileKey{Name: ownerRef.Name, Namespace: o.GetNamespace()})
			if err != nil || owner == nil {
				glog.V(2).Infof("Could not lookup owner %v %v", owner, err)
				return ""
			}
			var ownerObject metav1.Object
			if ownerObject, ok = owner.(metav1.Object); !ok {
				glog.V(2).Infof("No ObjectMeta for owner %v %v", owner, err)
				return ""
			}
			if ownerObject.GetUID() != ownerRef.UID {
				return ""
			}

			// Pop the path element or return the value
			if len(m.Path) > 1 {
				o = ownerObject
				m.Path = m.Path[1:]
			} else {
				return object.GetNamespace() + "/" + ownerRef.Name
			}
		}
	}
	return ""
}

// ObjToKey returns a string namespace/name key for an object
type ObjToKey func(interface{}) string

// MapToSelf returns the namespace/name key of obj
func MapToSelf(obj interface{}) string {
	if key, err := cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return ""
	} else {
		return key
	}
}
