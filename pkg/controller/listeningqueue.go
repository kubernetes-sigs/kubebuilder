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

package controller

import (
	"fmt"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// listeningQueue registers event providers and maps the observed events into strings that it then enqueues.
type listeningQueue struct {
	// RateLimitingInterface is the workqueue backing the listeningQueue
	workqueue.RateLimitingInterface

	// Name is the Name of the queue
	Name string

	// informerProvider contains a InformerGetter that is able to lookup informers for objects from their type
	informerProvider informers.InformerGetter

	// synced is a slice of functions that return whether or not all informers have been synced
	synced []cache.InformerSynced
}

// watchChannel enqueues message from a channel
func (q *listeningQueue) watchChannel(source <-chan string) error {
	go func() {
		for msg := range source {
			q.AddRateLimited(msg)
		}
	}()
	return nil
}

// addEventHandler uses the provider functions to add an event handler for events to objects matching obj's type
func (q *listeningQueue) addEventHandler(obj metav1.Object, eh eventhandlers.EventHandler) error {

	i, err := q.lookupInformer(obj)
	if err != nil {
		return err
	}
	fns := eh.Get(q.RateLimitingInterface)
	q.synced = append(q.synced, i.HasSynced)
	i.AddEventHandler(fns)
	return nil
}

// lookupInformer returns the SharedInformer for the type if found, otherwise exists
func (q *listeningQueue) lookupInformer(obj metav1.Object) (cache.SharedInformer, error) {
	i := q.informerProvider.GetInformer(obj)
	if i == nil {
		return i, fmt.Errorf("Could not find SharedInformer for %T in %s", obj, q.informerProvider)
	}
	return i, nil
}
