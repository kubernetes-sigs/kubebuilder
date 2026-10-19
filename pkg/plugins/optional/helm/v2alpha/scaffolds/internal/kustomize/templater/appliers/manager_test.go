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

var _ = Describe("Manager Pod Annotations", func() {
	Context("when pod template has no annotations", func() {
		It("injects pod annotations using toYaml without nested .Values re-evaluation", func() {
			input := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    control-plane: controller-manager
  name: controller-manager
spec:
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest`

			rendered := AddCustomLabelsAndAnnotations(input)
			pod := podTemplateSlice(rendered)

			Expect(pod).To(ContainSubstring("{{- with .Values.manager.pod }}"))
			Expect(pod).To(ContainSubstring("{{- with .annotations }}"))
			Expect(pod).To(ContainSubstring("annotations:"))
			Expect(pod).To(ContainSubstring("{{- toYaml . | nindent 8 }}"))
			// Must not re-reference .Values.manager.pod inside .annotations with-block
			disallowedSubstr := "{{- with .annotations }}\n      annotations:\n        {{- with .Values.manager.pod }}"
			Expect(pod).NotTo(ContainSubstring(disallowedSubstr))
			// Ensure only 1 .annotations block in pod template
			Expect(strings.Count(pod, "{{- with .annotations }}")).To(Equal(1))
		})
	})

	Context("when pod template has existing annotations", func() {
		It("merges into existing pod annotations using omit", func() {
			input := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    control-plane: controller-manager
  name: controller-manager
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest`

			rendered := AddCustomLabelsAndAnnotations(input)
			pod := podTemplateSlice(rendered)

			Expect(pod).To(ContainSubstring("kubectl.kubernetes.io/default-container: manager"))
			Expect(pod).To(ContainSubstring("{{- with .Values.manager.pod }}"))
			Expect(pod).To(ContainSubstring("{{- with .annotations }}"))
			Expect(pod).To(ContainSubstring(`{{- with omit . "kubectl.kubernetes.io/default-container" }}`))
			Expect(pod).To(ContainSubstring("{{- toYaml . | nindent 8 }}"))
			Expect(strings.Count(pod, "annotations:")).To(Equal(1))
		})
	})
})
