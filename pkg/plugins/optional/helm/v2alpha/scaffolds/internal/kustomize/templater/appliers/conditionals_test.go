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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func networkPolicy(name string) *unstructured.Unstructured {
	np := &unstructured.Unstructured{}
	np.SetAPIVersion("networking.k8s.io/v1")
	np.SetKind("NetworkPolicy")
	np.SetName(name)
	return np
}

func certificate(name string) *unstructured.Unstructured {
	cert := &unstructured.Unstructured{}
	cert.SetAPIVersion("cert-manager.io/v1")
	cert.SetKind("Certificate")
	cert.SetName(name)
	return cert
}

func conversionCRD() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		apiVersionKey: "apiextensions.k8s.io/v1",
		kindKey:       "CustomResourceDefinition",
		"metadata":    map[string]any{"name": "widgets.example.io"},
		"spec": map[string]any{
			"conversion": map[string]any{"strategy": "Webhook"},
		},
	}}
}

var _ = Describe("AddConditionalWrappers", func() {
	It("rejects disabling a webhook required by CRD conversion", func() {
		result := AddConditionalWrappers("kind: CustomResourceDefinition\n", conversionCRD(), "test-project")

		Expect(result).To(ContainSubstring(`{{- if and .Values.crd.enabled (not .Values.webhook.enabled) }}`))
		Expect(result).To(ContainSubstring(
			`fail "webhook.enabled must be true for CRD conversion"`))
		Expect(result).To(ContainSubstring("{{- if .Values.crd.enabled }}"))
	})

	It("does not require a webhook when conversion CRDs are disabled", func() {
		result := AddConditionalWrappers("kind: CustomResourceDefinition\n", conversionCRD(), "test-project")

		Expect(result).To(ContainSubstring(".Values.crd.enabled"))
		Expect(result).To(ContainSubstring(".Values.webhook.enabled"))
		Expect(result).NotTo(ContainSubstring("{{- if not .Values.webhook.enabled }}"))
	})

	It("keeps custom serving certificates under the cert-manager conditional", func() {
		result := AddConditionalWrappers(
			"kind: Certificate\n", certificate("test-project-custom-serving-cert"), "test-project")

		Expect(result).To(ContainSubstring("{{- if .Values.certManager.enabled }}"))
		Expect(result).NotTo(ContainSubstring(".Values.webhook.enabled"))
	})

	It("gates the metrics policy on both networkPolicy.enabled and metrics.enabled", func() {
		result := AddConditionalWrappers(
			"kind: NetworkPolicy\n", networkPolicy("test-project-allow-metrics-traffic"), "test-project")

		Expect(result).To(ContainSubstring(
			"{{- if and .Values.networkPolicy.enabled .Values.metrics.enabled }}"))
		Expect(result).To(ContainSubstring("{{- end }}"))
	})

	It("gates the webhook policy on both networkPolicy.enabled and webhook.enabled", func() {
		result := AddConditionalWrappers(
			"kind: NetworkPolicy\n", networkPolicy("test-project-allow-webhook-traffic"), "test-project")

		Expect(result).To(ContainSubstring(
			"{{- if and .Values.networkPolicy.enabled .Values.webhook.enabled }}"))
		Expect(result).To(ContainSubstring("{{- end }}"))
	})

	It("gates a custom policy on networkPolicy.enabled only", func() {
		result := AddConditionalWrappers(
			"kind: NetworkPolicy\n", networkPolicy("test-project-allow-dns-traffic"), "test-project")

		Expect(result).To(ContainSubstring("{{- if .Values.networkPolicy.enabled }}"))
		Expect(result).NotTo(ContainSubstring(".Values.metrics.enabled"))
		Expect(result).NotTo(ContainSubstring(".Values.webhook.enabled"))
	})

	It("does not classify a custom policy with an unrelated prefix as scaffolded", func() {
		result := AddConditionalWrappers(
			"kind: NetworkPolicy\n", networkPolicy("team-allow-metrics-traffic"), "test-project")

		Expect(result).To(ContainSubstring("{{- if .Values.networkPolicy.enabled }}"))
		Expect(result).NotTo(ContainSubstring(".Values.metrics.enabled"))
	})

	It("gates a policy whose name only resembles the scaffolded ones on networkPolicy.enabled only", func() {
		for _, name := range []string{"test-project-disallow-metrics-traffic", "test-project-disallow-webhook-traffic"} {
			result := AddConditionalWrappers("kind: NetworkPolicy\n", networkPolicy(name), "test-project")

			Expect(result).To(ContainSubstring("{{- if .Values.networkPolicy.enabled }}"))
			Expect(result).NotTo(ContainSubstring(".Values.metrics.enabled"))
			Expect(result).NotTo(ContainSubstring(".Values.webhook.enabled"))
		}
	})

	It("does not wrap a NetworkPolicy served by a non-standard apiVersion", func() {
		np := networkPolicy("custom-policy")
		np.SetAPIVersion("acme.io/v1")

		result := AddConditionalWrappers("kind: NetworkPolicy\n", np, "test-project")

		Expect(result).NotTo(ContainSubstring(".Values.networkPolicy.enabled"))
	})
})
