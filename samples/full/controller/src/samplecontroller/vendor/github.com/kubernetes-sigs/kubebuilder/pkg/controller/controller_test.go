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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/test"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

var _ = Describe("GenericController", func() {
	var (
		instance               *GenericController
		mgr                    *ControllerManager
		fakePodInformer        *test.FakeInformer
		fakeReplicaSetInformer *test.FakeInformer
		result                 chan string
		stop                   chan struct{}
		ch                     chan string
		t                      = true
	)

	BeforeEach(func() {
		mgr = &ControllerManager{}

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

	Describe("Watching a Pod from a controller", func() {
		BeforeEach(func() {
			// Create a new listeningQueue
			instance = &GenericController{
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
			It("should be able to lookup the controller", func() {
				Expect(mgr.GetController("TestInstance")).Should(Equal(instance))
			})

			It("should be able to lookup the informer provider", func() {
				Expect(mgr.GetInformerProvider(&corev1.Pod{})).Should(Equal(fakePodInformer))
			})

			It("should be able to lookup the informer provider", func() {
				Expect(mgr.GetInformer(&corev1.Pod{})).Should(Equal(fakePodInformer))
			})

			It("should reconcile the Pod namespace/name", func() {
				// Listen for Pod changes
				Expect(instance.Watch(&corev1.Pod{})).Should(Succeed())

				// Create a Pod event
				fakePodInformer.Add(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"}})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("default/test-pod"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})

			It("should reconcile the Controller namespace/name if the UID matches", func() {
				// Function to lookup the ReplicaSet based on the key
				fn := func(k types.ReconcileKey) (interface{}, error) {
					return &appsv1.ReplicaSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-replicaset",
							Namespace: "default",
							UID:       "uid5",
						},
					}, nil
				}
				// Listen for Pod changes
				Expect(instance.WatchControllerOf(&corev1.Pod{}, eventhandlers.Path{fn})).Should(Succeed())

				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid5",
							},
						},
					},
				})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("default/test-replicaset"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})

			It("should not reconcile the Controller namespace/name if the UID doesn't match", func() {
				// Function to lookup the ReplicaSet based on the key
				fn := func(k types.ReconcileKey) (interface{}, error) {
					return &appsv1.ReplicaSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-replicaset",
							Namespace: "default",
							UID:       "uid5",
						},
					}, nil
				}
				// Listen for Pod changes
				Expect(instance.WatchControllerOf(&corev1.Pod{}, eventhandlers.Path{fn})).Should(Succeed())

				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid3", // UID doesn't match
							},
						},
					},
				})

				val := ChannelResult{}
				Consistently(result).Should(Not(Receive(&val.result)))
			})

			It("should reconcile the Controller-Controller namespace/name", func() {
				// Function to lookup the ReplicaSet based on the key
				fn1 := func(k types.ReconcileKey) (interface{}, error) {
					return &appsv1.ReplicaSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-replicaset",
							Namespace: "default",
							UID:       "uid5",
							OwnerReferences: []metav1.OwnerReference{
								{
									Name:       "test-deployment",
									UID:        "uid7",
									Controller: &t,
								},
							},
						},
					}, nil
				}
				fn2 := func(k types.ReconcileKey) (interface{}, error) {
					return &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-deployment",
							Namespace: "default",
							UID:       "uid7",
						},
					}, nil
				}

				Expect(instance.WatchControllerOf(&corev1.Pod{}, eventhandlers.Path{fn1, fn2})).Should(Succeed())
				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid5",
							},
						},
					},
				})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("default/test-deployment"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})

			It("should use the map function to reconcile a different key", func() {
				// Listen for Pod changes
				Expect(instance.WatchTransformationOf(&corev1.Pod{}, func(obj interface{}) string {
					p := obj.(*corev1.Pod)
					return p.Namespace + "-namespace/" + p.Name + "-name"
				})).Should(Succeed())

				fakePodInformer.Add(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"}})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("default-namespace/test-pod-name"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})

			It("should call the event handling add function", func() {
				// Listen for Pod changes
				Expect(instance.WatchEvents(&corev1.Pod{},
					func(w workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
						return cache.ResourceEventHandlerFuncs{
							AddFunc:    func(obj interface{}) { w.AddRateLimited("key/value") },
							DeleteFunc: func(obj interface{}) { Fail("Delete function called") },
							UpdateFunc: func(obj, old interface{}) { Fail("Update function called") },
						}
					})).Should(Succeed())

				fakePodInformer.Add(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"}})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("key/value"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})

		Context("Where a Pod has been updated", func() {
			It("should call the event handling update function", func() {
				// Listen for Pod changes
				Expect(instance.WatchEvents(&corev1.Pod{},
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
				Expect(instance.WatchEvents(&corev1.Pod{},
					func(w workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
						return cache.ResourceEventHandlerFuncs{
							AddFunc:    func(obj interface{}) { Fail("Add function called") },
							DeleteFunc: func(obj interface{}) { w.AddRateLimited("key/value") },
							UpdateFunc: func(obj, old interface{}) { Fail("Update function called") },
						}
					})).Should(Succeed())

				fakePodInformer.Delete(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"}})

				val := ChannelResult{}
				Eventually(result).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("key/value"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})
	})

	Describe("Watching a channel", func() {
		BeforeEach(func() {
			ch = make(chan string)
			instance = &GenericController{
				Name:             "TestInstance",
				InformerRegistry: mgr,
				Reconcile: func(k types.ReconcileKey) error {
					// Write the result to a channel
					result <- k.Namespace + "/" + k.Name
					return nil
				},
			}
			mgr.AddController(instance)
			Expect(instance.WatchChannel(ch)).Should(Succeed())
			mgr.RunInformersAndControllers(run.RunArguments{Stop: stop})
		})

		Context("Where a key is added to the channel", func() {
			It("should reconcile the added namespace/name", func() {
				go func() { ch <- "hello/world" }()
				val := ChannelResult{}
				Eventually(result, time.Second*1).Should(Receive(&val.result))
				Expect(val.result).Should(Equal("hello/world"))
				Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
			})
		})

		Context("Where a key does not have a namespace/name", func() {
			It("should not reconcile the any namespace/name", func() {
				go func() { ch <- "hello/world/foo" }()
				val := ChannelResult{}
				Consistently(result, time.Second*1).Should(Not(Receive(&val.result)))
			})
		})
	})

	Describe("Creating an empty controller", func() {
		BeforeEach(func() {
			instance = &GenericController{
				AfterReconcile: func(k types.ReconcileKey, err error) {
					defer GinkgoRecover()
					Expect(err).Should(BeNil())
					result <- k.Namespace + "/" + k.Name
				},
			}
			defaultManager = ControllerManager{}
			AddInformerProvider(&corev1.Pod{}, fakePodInformer)
			Expect(GetInformerProvider(&corev1.Pod{})).Should(Equal(fakePodInformer))
			AddController(instance)
			RunInformersAndControllers(run.RunArguments{Stop: stop})
		})

		It("should create a name for the controller", func() {
			Expect(instance.Watch(&corev1.Pod{})).Should(Succeed())
			Expect(instance.Name).Should(Not(BeEmpty()))
		})

		It("should use the default informer registry", func() {
			Expect(instance.Watch(&corev1.Pod{})).Should(Succeed())

			// Create a Pod event
			fakePodInformer.Add(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"}})

			val := ChannelResult{}
			Eventually(result).Should(Receive(&val.result))
			Expect(val.result).Should(Equal("default/test-pod"))
			Expect(instance.GetMetrics().QueueLength).Should(Equal(0))
		})
	})

	Describe("Adding a non-string item to the queue", func() {
		BeforeEach(func() {
			instance = &GenericController{
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
		It("should not call reconcile", func() {
			instance.Watch(&corev1.Pod{})
			instance.queue.AddRateLimited(fakePodInformer)
			val := ChannelResult{}
			Consistently(result).Should(Not(Receive(&val.result)))
		})
	})

	Describe("Adding string where the namespace/name cannot be parsed", func() {
		BeforeEach(func() {
			instance = &GenericController{
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

		It("should not call reconcile", func() {
			instance.Watch(&corev1.Pod{})
			instance.queue.AddRateLimited("1/2/3")
			val := ChannelResult{}
			Consistently(result).Should(Not(Receive(&val.result)))
		})
	})

	AfterEach(func() {
		close(stop)
	})
})

type ChannelResult struct {
	result string
}
