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

package test

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// FakeInformer provides fake Informer functionality for testing
type FakeInformer struct {
	// Synced is returned by the HasSynced functions to implement the Informer interface
	Synced bool

	// RunCount is incremented each time RunInformersAndControllers is called
	RunCount int

	handlers []cache.ResourceEventHandler
}

func (f *FakeInformer) AddIndexers(indexers cache.Indexers) error {
	return nil
}

func (f *FakeInformer) GetIndexer() cache.Indexer {
	return nil
}

func (f *FakeInformer) Informer() cache.SharedIndexInformer {
	return f
}

// HasSynced implements the Informer interface.  Returns f.Synced
func (f *FakeInformer) HasSynced() bool {
	return f.Synced
}

// AddEventHandler implements the Informer interface.
func (f *FakeInformer) AddEventHandler(handler cache.ResourceEventHandler) {
	f.handlers = append(f.handlers, handler)
}

// RunInformersAndControllers implements the Informer interface.  Increments f.RunCount
func (f *FakeInformer) Run(<-chan struct{}) {
	f.RunCount++
}

// Add fakes an Add event for obj
func (f *FakeInformer) Add(obj metav1.Object) {
	for _, h := range f.handlers {
		h.OnAdd(obj)
	}
}

// Add fakes an Update event for obj
func (f *FakeInformer) Update(oldObj, newObj metav1.Object) {
	for _, h := range f.handlers {
		h.OnUpdate(oldObj, newObj)
	}
}

// Add fakes an Delete event for obj
func (f *FakeInformer) Delete(obj metav1.Object) {
	for _, h := range f.handlers {
		h.OnDelete(obj)
	}
}

func (f *FakeInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) {

}

func (f *FakeInformer) GetStore() cache.Store {
	return nil
}

func (f *FakeInformer) GetController() cache.Controller {
	return nil
}

func (f *FakeInformer) LastSyncResourceVersion() string {
	return ""
}
