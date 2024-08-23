/*
Copyright 2024 The Kubernetes authors.

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
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cachev1alpha1 "example.com/memcached/api/v1alpha1"
)

var _ = Describe("Memcached Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		memcached := &cachev1alpha1.Memcached{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Memcached")
			err := k8sClient.Get(ctx, typeNamespacedName, memcached)
			if err != nil && errors.IsNotFound(err) {
				resource := &cachev1alpha1.Memcached{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: cachev1alpha1.MemcachedSpec{
						Size: 1,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &cachev1alpha1.Memcached{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Memcached")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &MemcachedReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.Deployment{}
				return k8sClient.Get(ctx, typeNamespacedName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking the latest Status Condition added to the Memcached instance")
			Eventually(func() error {
				if memcached.Status.Conditions != nil &&
					len(memcached.Status.Conditions) != 0 {
					latestStatusCondition := memcached.Status.Conditions[len(memcached.Status.Conditions)-1]
					expectedLatestStatusCondition := metav1.Condition{
						Type:   typeAvailableMemcached,
						Status: metav1.ConditionTrue,
						Reason: "Reconciling",
						Message: fmt.Sprintf(
							"Deployment for custom resource (%s) with %d replicas created successfully",
							memcached.Name,
							memcached.Spec.Size),
					}
					if latestStatusCondition != expectedLatestStatusCondition {
						return fmt.Errorf("The latest status condition added to the Memcached instance is not as expected")
					}
				}
				return nil
			}, time.Minute, time.Second).Should(Succeed())
		})
	})
})
