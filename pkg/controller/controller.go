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
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/metrics"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/predicates"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var (
	// DefaultReconcileFn is used by GenericController if Reconcile is not set
	DefaultReconcileFn = func(k types.ReconcileKey) error {
		log.Printf("No ReconcileFn defined - skipping %+v", k)
		return nil
	}

	counter uint64
)

// Code originally copied from kubernetes/sample-controller at
// https://github.com/kubernetes/sample-controller/blob/994cb3621c790e286ab11fb74b3719b20bb55ca7/controller.go

// GenericController watches event sources and invokes a Reconcile function
type GenericController struct {
	// Name is the name of the controller
	Name string

	// Reconcile implements the controller business logic.
	Reconcile types.ReconcileFn

	// informerProvider contains the registry of shared informers to use.
	InformerRegistry informers.InformerGetter

	BeforeReconcile func(key types.ReconcileKey)
	AfterReconcile  func(key types.ReconcileKey, err error)

	// listeningQueue is an listeningQueue that listens for events from informers and adds object keys to
	// the queue for processing
	queue listeningQueue

	// syncTs contains the start times of each currently running reconcile loop
	syncTs sets.Int64

	// once ensures unspecified fields get default values
	once sync.Once
}

// GetMetrics returns metrics about the queue processing
func (gc *GenericController) GetMetrics() metrics.Metrics {
	// Get the current timestamps and sort them
	ts := gc.syncTs.List()
	sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
	return metrics.Metrics{
		UncompletedReconcileTs: ts,
		QueueLength:            gc.queue.Len(),
	}
}

// Watch watches objects matching obj's type and enqueues their keys to be reconcild.
func (gc *GenericController) Watch(obj metav1.Object, p ...predicates.Predicate) error {
	gc.once.Do(gc.init)
	return gc.queue.addEventHandler(obj,
		eventhandlers.MapAndEnqueue{Map: eventhandlers.MapToSelf, Predicates: p})
}

// WatchControllerOf reconciles the controller of the object type being watched.  e.g. If the
// controller created a Pod, watch the Pod for events and invoke the controller reconcile function.
// Uses path to lookup the ancestors.  Will lookup each ancestor in the path until it gets to the
// root and then reconcile this key.
//
// Example: Deployment controller creates a ReplicaSet.  ReplicaSet controller creates a Pod.  Deployment
// controller wants to have its reconcile method called for Pod events for any Pods it created (transitively).
// - Pod event occurs - find owners references
// - Lookup the Pod parent ReplicaSet by using the first path element (compare UID to ref)
// - Lookup the ReplicaSet parent Deployment by using the second path element (compare UID to ref)
// - Enqueue reconcile for Deployment namespace/name
//
// This could be implemented as:
// WatchControllerOf(&corev1.Pod, eventhandlers.Path{FnToLookupReplicaSetByNamespaceName, FnToLookupDeploymentByNamespaceName })
func (gc *GenericController) WatchControllerOf(obj metav1.Object, path eventhandlers.Path,
	p ...predicates.Predicate) error {
	gc.once.Do(gc.init)
	return gc.queue.addEventHandler(obj,
		eventhandlers.MapAndEnqueue{Map: eventhandlers.MapToController{Path: path}.Map, Predicates: p})
}

// WatchTransformationOf watches objects matching obj's type and enqueues the key returned by mapFn.
func (gc *GenericController) WatchTransformationOf(obj metav1.Object, mapFn eventhandlers.ObjToKey,
	p ...predicates.Predicate) error {
	gc.once.Do(gc.init)
	return gc.queue.addEventHandler(obj,
		eventhandlers.MapAndEnqueue{Map: mapFn, Predicates: p})
}

// WatchTransformationsOf watches objects matching obj's type and enqueues the keys returned by mapFn.
func (gc *GenericController) WatchTransformationsOf(obj metav1.Object, mapFn eventhandlers.ObjToKeys,
	p ...predicates.Predicate) error {
	gc.once.Do(gc.init)
	return gc.queue.addEventHandler(obj,
		eventhandlers.MapAndEnqueue{MultiMap: func(i interface{}) []types.ReconcileKey {
			result := []types.ReconcileKey{}
			for _, k := range mapFn(i) {
				if namespace, name, err := cache.SplitMetaNamespaceKey(k); err == nil {
					result = append(result, types.ReconcileKey{namespace, name})
				}
			}
			return result
		}, Predicates: p})
}

// WatchTransformationKeyOf watches objects matching obj's type and enqueues the key returned by mapFn.
func (gc *GenericController) WatchTransformationKeyOf(obj metav1.Object, mapFn eventhandlers.ObjToReconcileKey,
	p ...predicates.Predicate) error {
	gc.once.Do(gc.init)
	return gc.queue.addEventHandler(obj,
		eventhandlers.MapAndEnqueue{MultiMap: func(i interface{}) []types.ReconcileKey {
			if k := mapFn(i); len(k.Name) > 0 {
				return []types.ReconcileKey{k}
			} else {
				return []types.ReconcileKey{}
			}
		}, Predicates: p})
}

// WatchTransformationKeysOf watches objects matching obj's type and enqueues the keys returned by mapFn.
func (gc *GenericController) WatchTransformationKeysOf(obj metav1.Object, mapFn eventhandlers.ObjToReconcileKeys,
	p ...predicates.Predicate) error {
	gc.once.Do(gc.init)
	return gc.queue.addEventHandler(obj,
		eventhandlers.MapAndEnqueue{MultiMap: mapFn, Predicates: p})
}

// WatchEvents watches objects matching obj's type and uses the functions from provider to handle events.
func (gc *GenericController) WatchEvents(obj metav1.Object, provider types.HandleFnProvider) error {
	gc.once.Do(gc.init)
	return gc.queue.addEventHandler(obj, fnToInterfaceAdapter{provider})
}

// WatchChannel enqueues object keys read from the channel.
func (gc *GenericController) WatchChannel(source <-chan string) error {
	gc.once.Do(gc.init)
	return gc.queue.watchChannel(source)
}

// fnToInterfaceAdapter adapts a function to an interface
type fnToInterfaceAdapter struct {
	val func(workqueue.RateLimitingInterface) cache.ResourceEventHandler
}

func (f fnToInterfaceAdapter) Get(q workqueue.RateLimitingInterface) cache.ResourceEventHandler {
	return f.val(q)
}

// RunInformersAndControllers will set up the event handlers for types we are interested in, as well
// as syncing SharedIndexInformer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (gc *GenericController) run(options run.RunArguments) error {
	gc.once.Do(gc.init)
	defer runtime.HandleCrash()
	defer gc.queue.ShutDown()

	// Start the SharedIndexInformer factories to begin populating the SharedIndexInformer caches
	glog.Infof("Starting %s controller", gc.Name)

	// Wait for the caches to be synced before starting workers
	glog.Infof("Waiting for %s SharedIndexInformer caches to sync", gc.Name)
	if ok := cache.WaitForCacheSync(options.Stop, gc.queue.synced...); !ok {
		return fmt.Errorf("failed to wait for %s caches to sync", gc.Name)
	}

	glog.Infof("Starting %s workers", gc.Name)
	// Launch two workers to process resources
	for i := 0; i < options.ControllerParallelism; i++ {
		go wait.Until(gc.runWorker, time.Second, options.Stop)
	}

	glog.Infof("Started %s workers", gc.Name)
	<-options.Stop
	glog.Infof("Shutting %s down workers", gc.Name)

	return nil
}

func defaultWorkQueueProvider(name string) workqueue.RateLimitingInterface {
	return workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name)
}

// init defaults field values on c
func (gc *GenericController) init() {
	if gc.syncTs == nil {
		gc.syncTs = sets.Int64{}
	}

	if gc.InformerRegistry == nil {
		gc.InformerRegistry = DefaultManager
	}

	// Set the default reconcile fn to just print messages
	if gc.Reconcile == nil {
		gc.Reconcile = DefaultReconcileFn
	}

	if len(gc.Name) == 0 {
		gc.Name = fmt.Sprintf("controller-%d", atomic.AddUint64(&counter, 1))
	}

	// Default the queue name to match the controller name
	if len(gc.queue.Name) == 0 {
		gc.queue.Name = gc.Name
	}

	// Default the RateLimitingInterface to a NamedRateLimitingQueue
	if gc.queue.RateLimitingInterface == nil {
		gc.queue.RateLimitingInterface = defaultWorkQueueProvider(gc.Name)
	}

	// Set the InformerRegistry on the queue
	gc.queue.informerProvider = gc.InformerRegistry
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (gc *GenericController) runWorker() {
	for gc.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (gc *GenericController) processNextWorkItem() bool {
	obj, shutdown := gc.queue.Get()

	start := time.Now().Unix()
	gc.syncTs.Insert(start)
	defer gc.syncTs.Delete(start)

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workque   ue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer gc.queue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/Name. We do this as the delayed nature of the
		// workqueue means the items in the SharedIndexInformer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			gc.queue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in %s workqueue but got %#v", gc.Name, obj))
			return nil
		}
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			runtime.HandleError(fmt.Errorf("invalid resource key in %s queue: %s", gc.Name, key))
			return nil
		}

		rk := types.ReconcileKey{
			Name:      name,
			Namespace: namespace,
		}
		if gc.BeforeReconcile != nil {
			gc.BeforeReconcile(rk)
		}
		// RunInformersAndControllers the syncHandler, passing it the namespace/Name string of the
		// resource to be synced.
		if err = gc.Reconcile(rk); err != nil {
			if gc.AfterReconcile != nil {
				gc.AfterReconcile(rk, err)
			}
			return fmt.Errorf("error syncing %s queue '%s': %s", gc.Name, key, err.Error())
		}
		if gc.AfterReconcile != nil {
			gc.AfterReconcile(rk, err)
		}

		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		gc.queue.Forget(obj)
		glog.Infof("Successfully synced %s queue '%s'", gc.Name, key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}
