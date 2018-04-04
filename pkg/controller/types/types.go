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

package types

import (
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// ReconcileFn takes the key of an object and reconciles its desired and observed state.
type ReconcileFn func(ReconcileKey) error

// HandleFnProvider returns cache.ResourceEventHandler that may enqueue messages
type HandleFnProvider func(workqueue.RateLimitingInterface) cache.ResourceEventHandler

// ReconcileKey provides a lookup key for a Kubernetes object.
type ReconcileKey struct {
	// Namespace is the namespace of the object.  Empty for non-namespaced objects.
	Namespace string

	// Name is the name of the object.
	Name string
}

func (r ReconcileKey) String() string {
	if r.Namespace == "" {
		return r.Name
	}
	return r.Namespace + "/" + r.Name
}

// ParseReconcileKey returns the ReconcileKey that has been encoded into a string.
func ParseReconcileKey(key string) (ReconcileKey, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return ReconcileKey{}, err
	}
	return ReconcileKey{Name: name, Namespace: namespace}, nil
}
