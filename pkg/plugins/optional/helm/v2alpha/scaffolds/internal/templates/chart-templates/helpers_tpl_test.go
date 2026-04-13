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

package charttemplates

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("HelmHelpers", func() {
	Context("ServiceAccount helper template generation", func() {
		It("generates serviceAccountName helper that delegates to resourceName for truncation", func() {
			helpers := &HelmHelpers{
				ProjectNameMixin: machinery.ProjectNameMixin{ProjectName: "test-project"},
			}

			err := helpers.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())

			templateBody := helpers.TemplateBody

			Expect(templateBody).To(ContainSubstring(`{{- define "test-project.serviceAccountName" -}}`))
			Expect(templateBody).To(ContainSubstring(
				`{{- if and (not (.Values.serviceAccount.enable | default true)) .Values.serviceAccount.name }}`))
			Expect(templateBody).To(ContainSubstring(`{{- .Values.serviceAccount.name }}`))
			Expect(templateBody).To(ContainSubstring(
				`{{- include "test-project.resourceName" (dict "suffix" "controller-manager" "context" .) }}`))
		})

		It("generates resourceName helper with 63-character limit truncation logic", func() {
			helpers := &HelmHelpers{
				ProjectNameMixin: machinery.ProjectNameMixin{ProjectName: "my-operator"},
			}

			err := helpers.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())

			templateBody := helpers.TemplateBody

			Expect(templateBody).To(ContainSubstring(`{{- define "my-operator.resourceName" -}}`))
			Expect(templateBody).To(ContainSubstring(`{{- $maxLen := sub 62 (len $suffix) | int }}`))
			Expect(templateBody).To(ContainSubstring(`{{- if gt (len $fullname) $maxLen }}`))
			Expect(templateBody).To(ContainSubstring(`| trunc 63 | trimSuffix "-"`))
		})

		It("allows external ServiceAccount name to bypass nameOverride/fullnameOverride", func() {
			helpers := &HelmHelpers{
				ProjectNameMixin: machinery.ProjectNameMixin{ProjectName: "test-project"},
			}

			err := helpers.SetTemplateDefaults()
			Expect(err).NotTo(HaveOccurred())

			templateBody := helpers.TemplateBody

			Expect(templateBody).To(ContainSubstring(
				`{{- if and (not (.Values.serviceAccount.enable | default true)) .Values.serviceAccount.name }}`))
			Expect(templateBody).To(ContainSubstring(
				`{{- .Values.serviceAccount.name }}`))
		})
	})
})
