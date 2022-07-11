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

	examplecomv1alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v3-with-deploy-image/api/v1alpha1"
)

var _ = Describe("Memcached controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		MemcachedName      = "test-memcached"
		MemcachedNamespace = "default"
	)

	Context("Memcached controller test", func() {
		It("should create successfully the custom resource for the Memcached", func() {
			ctx := context.Background()

			By("Creating the custom resource for the Kind Memcached")
			memcached := &examplecomv1alpha1.Memcached{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: MemcachedName, Namespace: MemcachedNamespace}, memcached)
			if err != nil && errors.IsNotFound(err) {
				// Define a new custom resource
				memcached := &examplecomv1alpha1.Memcached{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "example.com.testproject.org/v1alpha1",
						Kind:       "Memcached",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      MemcachedName,
						Namespace: MemcachedNamespace,
					},
					Spec: examplecomv1alpha1.MemcachedSpec{
						Size:          1,
						ContainerPort: 11211,
					},
				}
				fmt.Fprintf(GinkgoWriter, fmt.Sprintf("Creating a new custom resource in the namespace: %s with the name %s\n", memcached.Namespace, memcached.Name))
				err = k8sClient.Create(ctx, memcached)
				if err != nil {
					Expect(err).To(Not(HaveOccurred()))
				}
			}

			By("Checking with Memcached Kind exist")
			Eventually(func() error {
				found := &examplecomv1alpha1.Memcached{}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: MemcachedName, Namespace: MemcachedNamespace}, found)
				if err != nil {
					return err
				}
				return nil
			}, time.Minute, time.Second).Should(Succeed())
		})
	})

})
