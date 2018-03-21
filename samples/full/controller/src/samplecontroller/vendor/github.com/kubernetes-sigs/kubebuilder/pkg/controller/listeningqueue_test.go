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
