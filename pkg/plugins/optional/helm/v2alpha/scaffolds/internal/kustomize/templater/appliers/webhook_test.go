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

var _ = Describe("TemplateWebhookConfiguration", func() {
	It("templates settings for each webhook independently", func() {
		content := `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
webhooks:
- name: configured.example.com
  namespaceSelector:
    matchLabels:
      webhook: enabled
  rules:
  - apiGroups:
    - example.com
- name: unconfigured.example.com
  rules:
  - apiGroups:
    - example.com
`

		result := TemplateWebhookConfiguration(content)

		Expect(strings.Count(result, ".Values.webhook.objectSelector")).To(Equal(2))
		Expect(strings.Count(result, ".Values.webhook.matchConditions")).To(Equal(2))
		Expect(strings.Count(result, ".Values.webhook.timeoutSeconds")).To(Equal(2))
		Expect(strings.Count(result, "  namespaceSelector:")).To(Equal(2))
	})

	It("does not duplicate fields supplied by a patch marker", func() {
		content := `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
webhooks:
- name: validation.example.com
  namespaceSelector:
    matchLabels:
      webhook: enabled
  rules:
  - apiGroups:
    - example.com
`

		result := TemplateWebhookConfiguration(content)

		Expect(strings.Count(result, "  namespaceSelector:")).To(Equal(1))
		Expect(result).To(ContainSubstring(".Values.webhook.objectSelector"))
	})

	It("omits the timeout when the value is unset", func() {
		content := `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
webhooks:
- name: validation.example.com
  rules:
  - apiGroups:
    - example.com
`

		result := TemplateWebhookConfiguration(content)

		Expect(result).To(ContainSubstring("{{ with .Values.webhook.timeoutSeconds }}"))
		Expect(result).To(ContainSubstring("timeoutSeconds: {{ . }}"))
	})
})
