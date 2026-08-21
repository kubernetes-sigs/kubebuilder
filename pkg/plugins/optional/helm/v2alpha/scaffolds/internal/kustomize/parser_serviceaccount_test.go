/*
Copyright 2026 The Kubernetes Authors.

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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser ServiceAccount classification", func() {
	parseYAML := func(yamlContent string) *ParsedResources {
		parser := NewParser("")
		resources, err := parser.ParseFromReader(strings.NewReader(yamlContent))
		Expect(err).NotTo(HaveOccurred())
		return resources
	}

	It("keeps a single manager ServiceAccount in ServiceAccount", func() {
		resources := parseYAML(`---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-project-controller-manager
  namespace: test-system
`)
		Expect(resources.ServiceAccount).NotTo(BeNil())
		Expect(resources.ServiceAccount.GetName()).To(Equal("test-project-controller-manager"))
		Expect(resources.ExtraServiceAccounts).To(BeEmpty())
	})

	It("classifies manager first then external ServiceAccounts", func() {
		resources := parseYAML(`---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-project-controller-manager
  namespace: test-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-sa
  namespace: external-namespace
`)
		Expect(resources.ServiceAccount).NotTo(BeNil())
		Expect(resources.ServiceAccount.GetName()).To(Equal("test-project-controller-manager"))
		Expect(resources.ExtraServiceAccounts).To(HaveLen(1))
		Expect(resources.ExtraServiceAccounts[0].GetName()).To(Equal("external-sa"))
	})

	It("classifies external first then manager ServiceAccounts", func() {
		resources := parseYAML(`---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-sa
  namespace: external-namespace
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-project-controller-manager
  namespace: test-system
`)
		Expect(resources.ServiceAccount).NotTo(BeNil())
		Expect(resources.ServiceAccount.GetName()).To(Equal("test-project-controller-manager"))
		Expect(resources.ExtraServiceAccounts).To(HaveLen(1))
		Expect(resources.ExtraServiceAccounts[0].GetName()).To(Equal("external-sa"))
	})

	It("places only external ServiceAccounts in ExtraServiceAccounts when no manager SA is present", func() {
		resources := parseYAML(`---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-sa
  namespace: external-namespace
`)
		Expect(resources.ServiceAccount).To(BeNil())
		Expect(resources.ExtraServiceAccounts).To(HaveLen(1))
		Expect(resources.ExtraServiceAccounts[0].GetName()).To(Equal("external-sa"))
	})

	It("uses the last controller-manager suffix ServiceAccount when no Deployment is present", func() {
		resources := parseYAML(`---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-project-controller-manager
  namespace: test-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager
  namespace: test-system
`)
		Expect(resources.ServiceAccount).NotTo(BeNil())
		Expect(resources.ServiceAccount.GetName()).To(Equal("controller-manager"))
		Expect(resources.ExtraServiceAccounts).To(HaveLen(1))
		Expect(resources.ExtraServiceAccounts[0].GetName()).To(Equal("test-project-controller-manager"))
	})

	It("resolves the manager ServiceAccount from the Deployment serviceAccountName reference", func() {
		resources := parseYAML(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-system
spec:
  template:
    spec:
      serviceAccountName: custom-operator-sa
      containers:
      - name: manager
        image: controller:latest
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: custom-operator-sa
  namespace: test-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-sa
  namespace: external-namespace
`)
		Expect(resources.ServiceAccount).NotTo(BeNil())
		Expect(resources.ServiceAccount.GetName()).To(Equal("custom-operator-sa"))
		Expect(resources.ServiceAccount.GetNamespace()).To(Equal("test-system"))
		Expect(resources.ExtraServiceAccounts).To(HaveLen(1))
		Expect(resources.ExtraServiceAccounts[0].GetName()).To(Equal("external-sa"))
	})

	It("matches ServiceAccounts by namespace when the same name appears in multiple namespaces", func() {
		resources := parseYAML(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-system
spec:
  template:
    spec:
      serviceAccountName: shared-sa
      containers:
      - name: manager
        image: controller:latest
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: shared-sa
  namespace: other-namespace
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: shared-sa
  namespace: test-system
`)
		Expect(resources.ServiceAccount).NotTo(BeNil())
		Expect(resources.ServiceAccount.GetNamespace()).To(Equal("test-system"))
		Expect(resources.ExtraServiceAccounts).To(HaveLen(1))
		Expect(resources.ExtraServiceAccounts[0].GetNamespace()).To(Equal("other-namespace"))
	})
})
