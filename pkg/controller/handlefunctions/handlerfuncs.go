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

package handlefunctions

import (
	"fmt"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// MappingEnqueuingFnProvider provides Fns to map objects to name/namespace keys and enqueue them as messages
type MappingEnqueuingFnProvider struct {
	// Map maps an object to a key that can be enqueued
	Map func(interface{}) string
}

// HandlingFnsForQueue accepts a workqueue and returns ResourceEventHandlerFuncs that enqueue messages to it
// for add / update / delete events
type HandlingFnsForQueue interface {
	Get(r workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs
}

// Get returns ResourceEventHandlerFuncs that Map an object to a Key and enqueue the key if it is non-empty
func (mp MappingEnqueuingFnProvider) Get(r workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
	// Enqueue the mapped key for updates to the object
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { mp.addRateLimited(r, obj) },
		UpdateFunc: func(old, obj interface{}) { mp.addRateLimited(r, obj) },
		DeleteFunc: func(obj interface{}) { mp.addRateLimited(r, obj) },
	}
}

// addRateLimited maps the obj to a string.  If the string is non-empty, it is enqueued.
func (mp MappingEnqueuingFnProvider) addRateLimited(r workqueue.RateLimitingInterface, obj interface{}) {
	k := mp.Map(obj)
	if len(k) > 0 {
		r.AddRateLimited(k)
	}
}

type MapToController struct {
	GVK []metav1.GroupVersionKind
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
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If GVK is empty don't filter
		found := len(m.GVK) == 0

		// Only notify the resource if its gvk matches
		for _, gvk := range m.GVK {
			if ownerRef.Kind == gvk.Kind && ownerRef.APIVersion == gvk.Group+"/"+gvk.Version {
				found = true
			}
		}

		if !found {
			return ""
		}
		return object.GetNamespace() + "/" + ownerRef.Name
	}
	return ""
}

// MapToSelf returns the namespace/name key of obj
func MapToSelf(obj interface{}) string {
	if key, err := cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return ""
	} else {
		return key
	}
}

// ObjToKey returns a string namespace/name key for an object
type ObjToKey func(interface{}) string
