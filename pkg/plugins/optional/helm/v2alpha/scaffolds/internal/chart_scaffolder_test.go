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

package internal

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("ChartScaffolder", func() {
	Describe("PrepareTemplates", func() {
		It("should add the generic metrics NetworkPolicy when it is missing", func() {
			manifestsPath := filepath.Join(GinkgoT().TempDir(), "install.yaml")
			Expect(os.WriteFile(manifestsPath, []byte(manifestsWithoutNetworkPolicy), 0o600)).To(Succeed())

			fs := executeChartScaffolder(manifestsPath)

			content, err := afero.ReadFile(fs, "dist/chart/templates/network-policy/allow-metrics-traffic.yaml")
			Expect(err).NotTo(HaveOccurred())

			rendered := string(content)
			Expect(rendered).To(ContainSubstring("{{- if .Values.networkPolicy.enabled }}"))
			Expect(rendered).To(ContainSubstring("kind: NetworkPolicy"))
			Expect(rendered).To(ContainSubstring(
				`name: {{ include "test-project.resourceName" (dict "suffix" "allow-metrics-traffic" "context" $) }}`))
			Expect(rendered).To(ContainSubstring("metrics: enabled"))
			Expect(rendered).To(ContainSubstring("port: {{ .Values.metrics.port }}"))

			_, err = afero.ReadFile(fs, "dist/chart/templates/network-policy/allow-webhook-traffic.yaml")
			Expect(err).To(HaveOccurred())

			values, err := afero.ReadFile(fs, "dist/chart/values.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(values)).To(ContainSubstring("networkPolicy:\n  enabled: false"))
		})

		It("should add generic metrics and webhook NetworkPolicies when no policy exists", func() {
			manifestsPath := filepath.Join(GinkgoT().TempDir(), "install.yaml")
			Expect(os.WriteFile(
				manifestsPath,
				[]byte(manifestsWithWebhooksWithoutNetworkPolicy),
				0o600,
			)).To(Succeed())

			fs := executeChartScaffolder(manifestsPath)

			metricsPolicy, err := afero.ReadFile(
				fs,
				"dist/chart/templates/network-policy/allow-metrics-traffic.yaml",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(metricsPolicy)).To(ContainSubstring("metrics: enabled"))
			Expect(string(metricsPolicy)).To(ContainSubstring("port: {{ .Values.metrics.port }}"))

			webhookPolicy, err := afero.ReadFile(
				fs,
				"dist/chart/templates/network-policy/allow-webhook-traffic.yaml",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(webhookPolicy)).To(ContainSubstring("webhook: enabled"))
			Expect(string(webhookPolicy)).To(ContainSubstring("port: {{ .Values.webhook.port }}"))
		})

		It("should not add fallback policies when kustomize output provides NetworkPolicies", func() {
			manifestsPath := filepath.Join(GinkgoT().TempDir(), "install.yaml")
			Expect(os.WriteFile(
				manifestsPath,
				[]byte(manifestsWithMetricsNetworkPolicyAndWebhooks),
				0o600,
			)).To(Succeed())

			fs := executeChartScaffolder(manifestsPath)

			content, err := afero.ReadFile(fs, "dist/chart/templates/network-policy/allow-metrics-traffic.yaml")
			Expect(err).NotTo(HaveOccurred())
			rendered := string(content)
			Expect(rendered).To(ContainSubstring("{{- if .Values.networkPolicy.enabled }}"))
			Expect(rendered).To(ContainSubstring("metrics: enabled"))
			Expect(rendered).To(ContainSubstring("port: {{ .Values.metrics.port }}"))

			_, err = afero.ReadFile(fs, "dist/chart/templates/network-policy/allow-webhook-traffic.yaml")
			Expect(err).To(HaveOccurred())

			values, err := afero.ReadFile(fs, "dist/chart/values.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(values)).To(ContainSubstring("networkPolicy:\n  enabled: true"))
		})

		It("should place all NetworkPolicies from kustomize output in the network-policy directory", func() {
			manifestsPath := filepath.Join(GinkgoT().TempDir(), "install.yaml")
			Expect(os.WriteFile(
				manifestsPath,
				[]byte(manifestsWithMultipleNetworkPolicies),
				0o600,
			)).To(Succeed())

			fs := executeChartScaffolder(manifestsPath)

			metricsPolicy, err := afero.ReadFile(
				fs,
				"dist/chart/templates/network-policy/allow-metrics-traffic.yaml",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(metricsPolicy)).To(ContainSubstring("{{- if .Values.networkPolicy.enabled }}"))
			Expect(string(metricsPolicy)).To(ContainSubstring("metrics: enabled"))
			Expect(string(metricsPolicy)).To(ContainSubstring("port: {{ .Values.metrics.port }}"))

			dnsPolicy, err := afero.ReadFile(
				fs,
				"dist/chart/templates/network-policy/allow-dns-traffic.yaml",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(dnsPolicy)).To(ContainSubstring("{{- if .Values.networkPolicy.enabled }}"))
			Expect(string(dnsPolicy)).To(ContainSubstring("dns: enabled"))
			Expect(string(dnsPolicy)).To(ContainSubstring("port: 5353"))

			webhookPolicy, err := afero.ReadFile(
				fs,
				"dist/chart/templates/network-policy/allow-webhook-traffic.yaml",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(webhookPolicy)).To(ContainSubstring(
				"{{- if and .Values.networkPolicy.enabled .Values.webhook.enabled }}"))
			Expect(string(webhookPolicy)).To(ContainSubstring("webhook: enabled"))
			Expect(string(webhookPolicy)).To(ContainSubstring("port: {{ .Values.webhook.port }}"))

			values, err := afero.ReadFile(fs, "dist/chart/values.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(values)).To(ContainSubstring("networkPolicy:\n  enabled: true"))
		})

		It("should add new kustomize NetworkPolicies after fallback policies were scaffolded", func() {
			tmpDir := GinkgoT().TempDir()
			firstManifestsPath := filepath.Join(tmpDir, "install-without-network-policy.yaml")
			Expect(os.WriteFile(firstManifestsPath, []byte(manifestsWithoutNetworkPolicy), 0o600)).To(Succeed())

			fs := executeChartScaffolder(firstManifestsPath)
			_, err := afero.ReadFile(fs, "dist/chart/templates/network-policy/allow-metrics-traffic.yaml")
			Expect(err).NotTo(HaveOccurred())

			secondManifestsPath := filepath.Join(tmpDir, "install-with-custom-network-policies.yaml")
			Expect(os.WriteFile(
				secondManifestsPath,
				[]byte(manifestsWithMultipleNetworkPolicies),
				0o600,
			)).To(Succeed())

			executeChartScaffolderWithFS(secondManifestsPath, fs)

			metricsPolicy, err := afero.ReadFile(
				fs,
				"dist/chart/templates/network-policy/allow-metrics-traffic.yaml",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(metricsPolicy)).To(ContainSubstring("metrics: enabled"))
			Expect(string(metricsPolicy)).To(ContainSubstring("port: {{ .Values.metrics.port }}"))

			dnsPolicy, err := afero.ReadFile(
				fs,
				"dist/chart/templates/network-policy/allow-dns-traffic.yaml",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(dnsPolicy)).To(ContainSubstring("dns: enabled"))
			Expect(string(dnsPolicy)).To(ContainSubstring("port: 5353"))

			webhookPolicy, err := afero.ReadFile(
				fs,
				"dist/chart/templates/network-policy/allow-webhook-traffic.yaml",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(webhookPolicy)).To(ContainSubstring(
				"{{- if and .Values.networkPolicy.enabled .Values.webhook.enabled }}"))
			Expect(string(webhookPolicy)).To(ContainSubstring("webhook: enabled"))
			Expect(string(webhookPolicy)).To(ContainSubstring("port: {{ .Values.webhook.port }}"))

			values, err := afero.ReadFile(fs, "dist/chart/values.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(values)).To(ContainSubstring("networkPolicy:\n  enabled: false"))
		})
	})
})

func executeChartScaffolder(manifestsPath string) afero.Fs {
	return executeChartScaffolderWithFS(manifestsPath, afero.NewMemMapFs())
}

func executeChartScaffolderWithFS(manifestsPath string, fs afero.Fs) afero.Fs {
	scaffolder := NewChartScaffolder(ChartScaffolderConfig{
		ProjectName:   "test-project",
		ManifestsFile: manifestsPath,
		OutputDir:     "dist",
	})
	builders, err := scaffolder.PrepareTemplates(machinery.Filesystem{})
	Expect(err).NotTo(HaveOccurred())

	cfg := cfgv3.New()
	Expect(cfg.SetProjectName("test-project")).To(Succeed())

	scaffold := machinery.NewScaffold(machinery.Filesystem{FS: fs}, machinery.WithConfig(cfg))
	Expect(scaffold.Execute(builders...)).To(Succeed())

	return fs
}

const manifestsWithoutNetworkPolicy = `apiVersion: v1
kind: Namespace
metadata:
  name: test-system
---
apiVersion: v1
kind: Service
metadata:
  name: test-project-controller-manager-metrics-service
  namespace: test-system
spec:
  ports:
    - name: https
      port: 8443
      targetPort: 8443
  selector:
    control-plane: controller-manager
    app.kubernetes.io/name: test-project
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: test-project
  template:
    metadata:
      labels:
        control-plane: controller-manager
        app.kubernetes.io/name: test-project
    spec:
      containers:
        - name: manager
          image: controller:latest
`

const manifestsWithWebhooksWithoutNetworkPolicy = manifestsWithoutNetworkPolicy + `---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: test-project-validating-webhook-configuration
webhooks: []
`

const manifestsWithMetricsNetworkPolicyAndWebhooks = manifestsWithoutNetworkPolicy + `---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-project-allow-metrics-traffic
  namespace: test-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: test-project
  policyTypes:
    - Ingress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            metrics: enabled
      ports:
        - port: 8443
          protocol: TCP
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: test-project-validating-webhook-configuration
webhooks: []
`

const manifestsWithMultipleNetworkPolicies = manifestsWithoutNetworkPolicy + `---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-project-allow-metrics-traffic
  namespace: test-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: test-project
  policyTypes:
    - Ingress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            metrics: enabled
      ports:
        - port: 8443
          protocol: TCP
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-project-allow-dns-traffic
  namespace: test-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: test-project
  policyTypes:
    - Ingress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            dns: enabled
      ports:
        - port: 5353
          protocol: UDP
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-project-allow-webhook-traffic
  namespace: test-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: test-project
  policyTypes:
    - Ingress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            webhook: enabled
      ports:
        - port: 443
          protocol: TCP
`
