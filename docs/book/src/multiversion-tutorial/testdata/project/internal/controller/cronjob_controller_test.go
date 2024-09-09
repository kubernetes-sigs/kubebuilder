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
	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

var _ = Describe("CronJob Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		cronjob := &batchv1.CronJob{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind CronJob")
			err := k8sClient.Get(ctx, typeNamespacedName, cronjob)
			if err != nil && errors.IsNotFound(err) {
				resource := &batchv1.CronJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "*/1 * * * *", // Example: runs every minute
						JobTemplate: kbatch.JobTemplateSpec{
							Spec: kbatch.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "example-container",
												Image:   "example-image",
												Command: []string{"echo", "Hello World"},
											},
										},
										RestartPolicy: corev1.RestartPolicyOnFailure,
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &batchv1.CronJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CronJob")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		// TODO: Fix me. We need to implement the tests and ensure
		// that the controller implementation of multi-version tutorial is accurate
		//It("should successfully reconcile the resource", func() {
		//	By("Reconciling the created resource")
		//	controllerReconciler := &CronJobReconciler{
		//		Client: k8sClient,
		//		Scheme: k8sClient.Scheme(),
		//	}
		//
		//	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
		//		NamespacedName: typeNamespacedName,
		//	})
		//	Expect(err).NotTo(HaveOccurred())
		//	// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
		//	// Example: If you expect a certain status condition after reconciliation, verify it here.
		//})
	})
})
