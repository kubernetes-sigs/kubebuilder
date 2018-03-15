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

package controller_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/test"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var _ = Describe("GenericController", func() {
	var (
		instance               *controller.GenericController
		mgr                    *controller.ControllerManager
		fakePodInformer        *test.FakeInformer
		fakeReplicaSetInformer *test.FakeInformer
		result                 chan string
		stop                   chan struct{}
	)

	BeforeEach(func() {
		mgr = &controller.ControllerManager{}

		// Create a new informers map with fake informers
		fakePodInformer = &test.FakeInformer{Synced: true}
		Expect(mgr.AddInformerProvider(&corev1.Pod{}, fakePodInformer)).To(Succeed())

		// Don't allow inserting the same informer 2x
		Expect(mgr.AddInformerProvider(&corev1.Pod{}, fakePodInformer)).To(Not(Succeed()))

		fakeReplicaSetInformer = &test.FakeInformer{Synced: true}
		Expect(mgr.AddInformerProvider(&appsv1.ReplicaSet{}, fakeReplicaSetInformer)).To(Succeed())

		result = make(chan string)
		stop = make(chan struct{})
	})

	Describe("Listening to a Pod SharedInformer", func() {
		BeforeEach(func() {
			// Create a new listeningQueue
			instance = &controller.GenericController{
				Name:             "TestInstance",
				InformerRegistry: mgr,
				Reconcile: func(k types.ReconcileKey) error {
					// Write the result to a channel
					result <- k.Namespace + "/" + k.Name
					return nil
				},
			}
			mgr.AddController(instance)
			mgr.RunInformersAndControllers(run.RunArguments{Stop: stop})
		})

		Context("Where a Pod has been added", func() {
			It("should reconcile the Pod namespace/name", func() {
				// Listen for Pod changes
				Expect(instance.Watch(&corev1.Pod{})).Should(Succeed())

				// Create a Pod event
				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("default/test-pod"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})

		Context("Where a Pod has been added", func() {
			It("should reconcile the Controller namespace/name", func() {
				// Listen for Pod changes
				Expect(instance.WatchAndMapToController(&corev1.Pod{})).Should(Succeed())

				c := true
				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &c,
							},
						},
					},
				})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("default/test-replicaset"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})

		Context("Where a Pod has been added", func() {
			It("should reconcile the mapped key", func() {
				// Listen for Pod changes
				Expect(instance.WatchAndMap(&corev1.Pod{}, func(obj interface{}) string {
					p := obj.(*corev1.Pod)
					return p.Namespace + "-namespace/" + p.Name + "-name"
				})).Should(Succeed())

				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("default-namespace/test-pod-name"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})

		Context("Where a Pod has been added", func() {
			It("should call the event handling add function", func() {
				// Listen for Pod changes
				Expect(instance.WatchAndHandleEvents(&corev1.Pod{},
					func(w workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
						return cache.ResourceEventHandlerFuncs{
							AddFunc:    func(obj interface{}) { w.AddRateLimited("key/value") },
							DeleteFunc: func(obj interface{}) { Fail("Delete function called") },
							UpdateFunc: func(obj, old interface{}) { Fail("Update function called") },
						}
					})).Should(Succeed())

				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("key/value"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})

		Context("Where a Pod has been updated", func() {
			It("should call the event handling update function", func() {
				// Listen for Pod changes
				Expect(instance.WatchAndHandleEvents(&corev1.Pod{},
					func(w workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
						return cache.ResourceEventHandlerFuncs{
							AddFunc:    func(obj interface{}) { Fail("Add function called") },
							DeleteFunc: func(obj interface{}) { Fail("Delete function called") },
							UpdateFunc: func(obj, old interface{}) { w.AddRateLimited("key/value") },
						}
					})).Should(Succeed())

				fakePodInformer.Update(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				}, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("key/value"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})

		Context("Where a Pod has been deleted", func() {
			It("should call the event handling delete function", func() {
				// Listen for Pod changes
				Expect(instance.WatchAndHandleEvents(&corev1.Pod{},
					func(w workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
						return cache.ResourceEventHandlerFuncs{
							AddFunc:    func(obj interface{}) { Fail("Add function called") },
							DeleteFunc: func(obj interface{}) { w.AddRateLimited("key/value") },
							UpdateFunc: func(obj, old interface{}) { Fail("Update function called") },
						}
					})).Should(Succeed())

				fakePodInformer.Delete(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("key/value"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})
	})

	Describe("Checking Metrics to a Pod SharedInformer", func() {
		BeforeEach(func() {
			// Create a new listeningQueue
			instance = &controller.GenericController{
				Name:             "TestInstance",
				InformerRegistry: mgr,
				Reconcile: func(k types.ReconcileKey) error {
					// Write the result to a channel
					result <- k.Namespace + "/" + k.Name
					return nil
				},
			}
			mgr.AddController(instance)
			mgr.RunInformersAndControllers(run.RunArguments{Stop: stop})
		})

		Context("Where a Pod has been added", func() {
			It("should reconcile the Pod namespace/name", func() {
				// Listen for Pod changes
				Expect(instance.Watch(&corev1.Pod{})).Should(Succeed())

				// Create a Pod event
				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("default/test-pod"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})
	})

	AfterEach(func() {
		close(stop)
	})
})

type ChannelResult struct {
	result string
}
