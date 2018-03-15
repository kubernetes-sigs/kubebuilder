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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
)

var _ = Describe("ListeningQueue", func() {
	var (
		instance               listeningQueue
		manager                *ControllerManager
		fakePodInformer        *test.FakeInformer
		fakeReplicaSetInformer *test.FakeInformer
	)

	BeforeEach(func() {
		// Create a new informers map with fake informers
		manager = &ControllerManager{}
		fakePodInformer = &test.FakeInformer{Synced: true}
		manager.AddInformerProvider(&corev1.Pod{}, fakePodInformer)

		fakeReplicaSetInformer = &test.FakeInformer{Synced: true}
		manager.AddInformerProvider(&appsv1.ReplicaSet{}, fakeReplicaSetInformer)

		// Create a new listeningQueue
		instance = listeningQueue{
			RateLimitingInterface: workqueue.NewNamedRateLimitingQueue(
				workqueue.DefaultControllerRateLimiter(), "test"),
			informerProvider: manager,
		}
	})

	Describe("Listening to a Pod SharedInformer", func() {
		Context("Where a Pod has been added", func() {
			It("should add the Pod namespace/name key to the queue", func() {
				// Listen for Pod changes
				Expect(instance.watchFor(&corev1.Pod{})).Should(Succeed())

				// Create a Pod event
				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				})

				Eventually(instance.Len, time.Second*2).Should(Equal(1))
				key, shutdown := instance.Get()
				Expect(shutdown).To(Equal(false))
				Expect(key).To(Equal("default/test-pod"))
				Expect(instance.Len()).To(Equal(0))
			})
		})

		Context("Where several Pods have been added", func() {
			It("should add all the Pod namespace/name keys to the queue", func() {
				// Listen for Pod changes
				Expect(instance.watchFor(&corev1.Pod{})).Should(Succeed())

				// Create a Pod event
				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "default-1",
					},
				})

				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-2",
						Namespace: "default-2",
					},
				})

				keys := []string{}
				Eventually(instance.Len, time.Second*2).Should(Equal(2))

				key, shutdown := instance.Get()
				Expect(shutdown).To(Equal(false))
				keys = append(keys, key.(string))

				key, shutdown = instance.Get()
				Expect(shutdown).To(Equal(false))
				keys = append(keys, key.(string))

				Expect(instance.Len()).To(Equal(0))
				Expect(keys).Should(ConsistOf("default-1/test-pod-1", "default-2/test-pod-2"))
			})
		})

		Context("Where the same Pod is added multiple times", func() {
			It("should add the Pod namespace/name to the queue exactly once", func() {
				// Listen for Pod changes
				Expect(instance.watchFor(&corev1.Pod{})).Should(Succeed())

				// Create a Pod event
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				}

				// Add the Pod a bunch of times
				for i := 0; i < 10; i++ {
					fakePodInformer.Add(pod)
				}

				Consistently(instance.Len, time.Second*2).Should(BeNumerically("<=", 1))

				Expect(instance.Len()).To(Equal(1))
				key, shutdown := instance.Get()
				Expect(shutdown).To(Equal(false))
				Expect(key).To(Equal("default/test-pod"))
				Expect(instance.Len()).To(Equal(0))
			})
		})
	})

	Describe("Listening to a ReplicaSet SharedInformer", func() {
		Context("Where a Pod has been added by the RS", func() {
			It("should add the parent ReplicaSet namespace/name to the queue", func() {
				// Listen for Pod changes
				Expect(instance.watchForAndMapToController(&corev1.Pod{})).Should(Succeed())

				// Create a Pod event
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

				Eventually(instance.Len, time.Second*1).Should(Equal(1))
				key, shutdown := instance.Get()
				Expect(shutdown).To(Equal(false))
				Expect(key).To(Equal("default/test-replicaset"))
				Expect(instance.Len()).To(Equal(0))
			})
		})

		Context("Where a Pod has been added but not by the RS", func() {
			It("should add the parent ReplicaSet namespace/name to the queue", func() {
				// Listen for Pod changes
				Expect(instance.watchForAndMapToController(&corev1.Pod{})).Should(Succeed())

				// Create a Pod event
				fakePodInformer.Add(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name: "test-replicaset",
								// Doesn't have controller set
							},
						},
					},
				})

				// Should not enqueue a message since controller isn't set
				Consistently(instance.Len, time.Second*1).Should(Equal(0))
			})
		})
	})

	Describe("Listening to a Channel", func() {
		Context("Where a message is sent", func() {
			It("should enqueue the message", func() {
				// Listen for Pod changes
				c := make(chan string)
				Expect(instance.watchChannel(c)).Should(Succeed())
				c <- "default/test-pod"

				Eventually(instance.Len, time.Second*1).Should(Equal(1))
				key, shutdown := instance.Get()
				Expect(shutdown).To(Equal(false))
				Expect(key).To(Equal("default/test-pod"))
				Expect(instance.Len()).To(Equal(0))
			})
		})
	})
})
