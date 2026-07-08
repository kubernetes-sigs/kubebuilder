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

package appliers

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// countMetadataHeader counts how many times key (for example "labels:") appears as a standalone
// metadata header line, ignoring Helm guards and value references that also mention the word.
func countMetadataHeader(rendered, key string) int {
	count := 0
	for line := range strings.SplitSeq(rendered, "\n") {
		if strings.TrimSpace(line) == key {
			count++
		}
	}
	return count
}

// The ServiceAccount metadata reaching the applier depends on what config/kustomize produced:
// the base always ships block-style labels, and commonAnnotations or edits to service_account.yaml
// can add annotations, drop labels, add empty-value keys, or emit inline empty maps.
const (
	saLabelsOnly = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager
  namespace: system`

	saAnnotationsThenLabels = `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    example.com/existing-annotation: keep
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager`

	saLabelsThenAnnotations = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
  annotations:
    example.com/existing-annotation: keep
  name: controller-manager`

	saAnnotationsOnly = `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    example.com/existing-annotation: keep
  name: controller-manager`

	saNoLabelsNoAnnotations = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager
  namespace: system`

	saEmptyValueKey = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
    example.com/empty:
  name: controller-manager`

	saInlineAnnotations = `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations: {}
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager`

	saInlineLabels = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels: {}
  name: controller-manager`

	// A labels: header that is not a direct child of metadata: (here nested under another key).
	// Only the metadata block should be templated; the deeper labels: must be left as written.
	saNestedNonMetadataLabels = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager
imagePullSecrets:
  - name: my-registry
    labels:
      example.com/team: platform`
)

var _ = Describe("AddServiceAccountLabelsAndAnnotations", func() {
	// Every metadata shape must render exactly one labels block and one annotations block, both
	// wired to the user-value maps, so the manifest stays valid YAML and honors the
	// serviceAccount.labels and serviceAccount.annotations chart options.
	DescribeTable("keeps one labels and one annotations block wired to the chart values",
		func(input string, check func(rendered string)) {
			rendered := AddServiceAccountLabelsAndAnnotations(input)

			Expect(countMetadataHeader(rendered, "labels:")).To(Equal(1),
				"want exactly one labels: header in:\n%s", rendered)
			Expect(countMetadataHeader(rendered, "annotations:")).To(Equal(1),
				"want exactly one annotations: header in:\n%s", rendered)
			Expect(rendered).To(ContainSubstring(valuesServiceAccountLabels))
			Expect(rendered).To(ContainSubstring(valuesServiceAccountAnnotations))

			check(rendered)
		},
		Entry("labels only (scaffold default): the annotations block is injected",
			saLabelsOnly, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" }}`))
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.annotations }}`))
			}),
		Entry("annotations then labels (Kustomize alphabetical order): both merge, none duplicated",
			saAnnotationsThenLabels, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`example.com/existing-annotation: keep`))
				Expect(rendered).To(ContainSubstring(`{{- with omit . "example.com/existing-annotation" }}`))
				Expect(rendered).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" }}`))
			}),
		Entry("labels then annotations (hand-ordered): both merge, none duplicated",
			saLabelsThenAnnotations, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`{{- with omit . "example.com/existing-annotation" }}`))
				Expect(rendered).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" }}`))
			}),
		Entry("annotations only (labels removed from base): the labels block is injected",
			saAnnotationsOnly, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.labels }}`))
				Expect(rendered).To(ContainSubstring(`{{- with omit . "example.com/existing-annotation" }}`))
			}),
		Entry("neither block present: both are injected",
			saNoLabelsNoAnnotations, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.labels }}`))
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.annotations }}`))
			}),
		Entry("empty-value label key is still omitted so a user override cannot duplicate it",
			saEmptyValueKey, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`"example.com/empty"`))
			}),
		Entry("inline empty annotations map does not duplicate the annotations key",
			saInlineAnnotations, func(rendered string) {
				Expect(rendered).NotTo(ContainSubstring("{}"))
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.annotations }}`))
			}),
		Entry("inline empty labels map does not duplicate the labels key",
			saInlineLabels, func(rendered string) {
				Expect(rendered).NotTo(ContainSubstring("{}"))
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.labels }}`))
			}),
	)

	It("only templates labels/annotations that are direct children of metadata", func() {
		rendered := AddServiceAccountLabelsAndAnnotations(saNestedNonMetadataLabels)

		// The metadata labels block is merged with the chart values once.
		Expect(strings.Count(rendered, valuesServiceAccountLabels)).To(Equal(1))
		Expect(rendered).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" }}`))

		// The labels: nested under imagePullSecrets is left exactly as written.
		Expect(rendered).To(ContainSubstring("    labels:\n      example.com/team: platform"))
	})
})
