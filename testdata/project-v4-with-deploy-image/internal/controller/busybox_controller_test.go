/*
Copyright 2023 The Kubernetes authors.

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
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	examplecomv1alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-with-deploy-image/api/v1alpha1"
)

var _ = Describe("Busybox controller", func() {
	Context("Busybox controller test", func() {

		const BusyboxName = "test-busybox"

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BusyboxName,
				Namespace: BusyboxName,
			},
		}

		typeNamespaceName := types.NamespacedName{Name: BusyboxName, Namespace: BusyboxName}
		busybox := &examplecomv1alpha1.Busybox{}

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))

			By("Setting the Image ENV VAR which stores the Operand image")
			err = os.Setenv("BUSYBOX_IMAGE", "example.com/image:test")
			Expect(err).To(Not(HaveOccurred()))

			By("creating the custom resource for the Kind Busybox")
			err = k8sClient.Get(ctx, typeNamespaceName, busybox)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				busybox := &examplecomv1alpha1.Busybox{
					ObjectMeta: metav1.ObjectMeta{
						Name:      BusyboxName,
						Namespace: namespace.Name,
					},
					Spec: examplecomv1alpha1.BusyboxSpec{
						Size: 1,
					},
				}

				err = k8sClient.Create(ctx, busybox)
				Expect(err).To(Not(HaveOccurred()))
			}
		})

		AfterEach(func() {
			By("removing the custom resource for the Kind Busybox")
			found := &examplecomv1alpha1.Busybox{}
			err := k8sClient.Get(ctx, typeNamespaceName, found)
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() error {
				return k8sClient.Delete(context.TODO(), found)
			}, 2*time.Minute, time.Second).Should(Succeed())

			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations.
			// More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)

			By("Removing the Image ENV VAR which stores the Operand image")
			_ = os.Unsetenv("BUSYBOX_IMAGE")
		})

		It("should successfully reconcile a custom resource for Busybox", func() {
			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &examplecomv1alpha1.Busybox{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			busyboxReconciler := &BusyboxReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := busyboxReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.Deployment{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking the latest Status Condition added to the Busybox instance")
			Eventually(func() error {
				if busybox.Status.Conditions != nil && len(busybox.Status.Conditions) != 0 {
					latestStatusCondition := busybox.Status.Conditions[len(busybox.Status.Conditions)-1]
					expectedLatestStatusCondition := metav1.Condition{Type: typeAvailableBusybox,
						Status: metav1.ConditionTrue, Reason: "Reconciling",
						Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", busybox.Name, busybox.Spec.Size)}
					if latestStatusCondition != expectedLatestStatusCondition {
						return fmt.Errorf("The latest status condition added to the busybox instance is not as expected")
					}
				}
				return nil
			}, time.Minute, time.Second).Should(Succeed())
		})
	})
})
