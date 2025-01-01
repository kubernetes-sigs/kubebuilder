/*
Copyright 2025 The Kubernetes authors.

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

package examplecom

import (
	"context"
	"os"
	"time"

	//nolint:golint
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	examplecomv1alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v1alpha1"
)

var _ = Describe("Memcached controller", func() {
	Context("Memcached controller test", func() {

		const MemcachedName = "test-memcached"

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      MemcachedName,
				Namespace: MemcachedName,
			},
		}

		typeNamespacedName := types.NamespacedName{
			Name:      MemcachedName,
			Namespace: MemcachedName,
		}
		memcached := &examplecomv1alpha1.Memcached{}

		SetDefaultEventuallyTimeout(2 * time.Minute)
		SetDefaultEventuallyPollingInterval(time.Second)

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).NotTo(HaveOccurred())

			By("Setting the Image ENV VAR which stores the Operand image")
			err = os.Setenv("MEMCACHED_IMAGE", "example.com/image:test")
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind Memcached")
			err = k8sClient.Get(ctx, typeNamespacedName, memcached)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				memcached := &examplecomv1alpha1.Memcached{
					ObjectMeta: metav1.ObjectMeta{
						Name:      MemcachedName,
						Namespace: namespace.Name,
					},
					Spec: examplecomv1alpha1.MemcachedSpec{
						Size:          1,
						ContainerPort: 11211,
					},
				}

				err = k8sClient.Create(ctx, memcached)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			By("removing the custom resource for the Kind Memcached")
			found := &examplecomv1alpha1.Memcached{}
			err := k8sClient.Get(ctx, typeNamespacedName, found)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Delete(context.TODO(), found)).To(Succeed())
			}).Should(Succeed())

			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations.
			// More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)

			By("Removing the Image ENV VAR which stores the Operand image")
			_ = os.Unsetenv("MEMCACHED_IMAGE")
		})

		It("should successfully reconcile a custom resource for Memcached", func() {
			By("Checking if the custom resource was successfully created")
			Eventually(func(g Gomega) {
				found := &examplecomv1alpha1.Memcached{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, found)).To(Succeed())
			}).Should(Succeed())

			By("Reconciling the custom resource created")
			memcachedReconciler := &MemcachedReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := memcachedReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func(g Gomega) {
				found := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, found)).To(Succeed())
			}).Should(Succeed())

			By("Reconciling the custom resource again")
			_, err = memcachedReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the latest Status Condition added to the Memcached instance")
			Expect(k8sClient.Get(ctx, typeNamespacedName, memcached)).To(Succeed())
			conditions := []metav1.Condition{}
			Expect(memcached.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal(typeAvailableMemcached)), &conditions))
			Expect(conditions).To(HaveLen(1), "Multiple conditions of type %s", typeAvailableMemcached)
			Expect(conditions[0].Status).To(Equal(metav1.ConditionTrue), "condition %s", typeAvailableMemcached)
			Expect(conditions[0].Reason).To(Equal("Reconciling"), "condition %s", typeAvailableMemcached)
		})
	})
})
