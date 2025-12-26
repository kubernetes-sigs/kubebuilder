/*
Copyright 2025 The Kubernetes Authors.

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

package kustomize

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("ResourceOrganizer extras handling", func() {
	It("moves unhandled resources into the extras group", func() {
		// Known resource (handled)
		deployment := &unstructured.Unstructured{}
		deployment.SetAPIVersion("apps/v1")
		deployment.SetKind("Deployment")
		deployment.SetName("test-controller-manager")

		// Unknown / unscaffolded Service
		service := &unstructured.Unstructured{}
		service.SetAPIVersion("v1")
		service.SetKind("Service")
		service.SetName("custom-user-service")

		// Unknown / unscaffolded ConfigMap
		configMap := &unstructured.Unstructured{}
		configMap.SetAPIVersion("v1")
		configMap.SetKind("ConfigMap")
		configMap.SetName("custom-config")

		// Unknown / unscaffolded Secret
		secret := &unstructured.Unstructured{}
		secret.SetAPIVersion("v1")
		secret.SetKind("Secret")
		secret.SetName("custom-secret")

		resources := &ParsedResources{
			AllResources: []*unstructured.Unstructured{
				deployment,
				service,
				configMap,
				secret,
			},
			Deployment: deployment,
			Services: []*unstructured.Unstructured{
				service,
			},
		}

		organizer := NewResourceOrganizer(resources)
		groups := organizer.OrganizeByFunction()

		// Deployment should be classified correctly
		Expect(groups).To(HaveKey("manager"))
		Expect(groups["manager"]).To(ContainElement(deployment))

		// Unhandled resources should NOT be dropped
		Expect(groups).To(HaveKey("extras"))
		Expect(groups["extras"]).To(ContainElement(service))
		Expect(groups["extras"]).To(ContainElement(configMap))
		Expect(groups["extras"]).To(ContainElement(secret))

		// Unhandled resources must NOT be misclassified
		Expect(groups).NotTo(HaveKey("metrics"))
		Expect(groups).NotTo(HaveKey("webhook"))
		Expect(groups).NotTo(HaveKey("cert-manager"))
	})
})
