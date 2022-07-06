/*
Copyright 2022 The Kubernetes authors.

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

package controllers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	examplecomv1alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-with-deploy-image/api/v1alpha1"
)

var _ = Describe("Busybox controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		BusyboxName      = "test-busybox"
		BusyboxNamespace = "default"
	)

	Context("Busybox controller test", func() {
		It("should create successfully the custom resource for the Busybox", func() {
			ctx := context.Background()

			By("Creating the custom resource for the Kind Busybox")
			busybox := &examplecomv1alpha1.Busybox{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: BusyboxName, Namespace: BusyboxNamespace}, busybox)
			if err != nil && errors.IsNotFound(err) {
				// Define a new custom resource
				busybox := &examplecomv1alpha1.Busybox{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "example.com.testproject.org/v1alpha1",
						Kind:       "Busybox",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      BusyboxName,
						Namespace: BusyboxNamespace,
					},
					Spec: examplecomv1alpha1.BusyboxSpec{
						Size: 1,
					},
				}
				fmt.Fprintf(GinkgoWriter, fmt.Sprintf("Creating a new custom resource in the namespace: %s with the name %s\n", busybox.Namespace, busybox.Name))
				err = k8sClient.Create(ctx, busybox)
				if err != nil {
					Expect(err).To(Not(HaveOccurred()))
				}
			}

			By("Checking with Busybox Kind exist")
			Eventually(func() error {
				found := &examplecomv1alpha1.Busybox{}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: BusyboxName, Namespace: BusyboxNamespace}, found)
				if err != nil {
					return err
				}
				return nil
			}, time.Minute, time.Second).Should(Succeed())
		})
	})

})
