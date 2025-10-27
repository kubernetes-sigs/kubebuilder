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

var _ = Describe("HelmTemplater", func() {
	var templater *HelmTemplater

	BeforeEach(func() {
		templater = &HelmTemplater{
			projectName: "test-project",
		}
	})

	// No global labels injection is performed by v2-alpha

	Context("basic template processing", func() {
		It("should replace kustomize managed-by labels with Helm equivalents", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  template:
    metadata:
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			// Should replace kustomize managed-by with Helm template
			Expect(result).To(ContainSubstring("app.kubernetes.io/managed-by: {{ .Release.Service }}"))
			Expect(result).To(ContainSubstring("app.kubernetes.io/name: test-project"))
			Expect(result).To(ContainSubstring("control-plane: controller-manager"))

			// Should substitute namespace
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))

			// Should NOT add extra Helm metadata injection
			Expect(result).NotTo(ContainSubstring(`{{- include "chart.labels"`))
			Expect(result).NotTo(ContainSubstring(`{{- include "chart.annotations"`))
		})

		It("should handle cert-manager annotations with proper indentation", func() {
			resource := &unstructured.Unstructured{}
			resource.SetAPIVersion("admissionregistration.k8s.io/v1")
			resource.SetKind("ValidatingWebhookConfiguration")
			resource.SetName("test-project-validating-webhook-configuration")

			content := `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  annotations:
    cert-manager.io/inject-ca-from: test-project-system/test-project-serving-cert
  name: test-project-validating-webhook-configuration
webhooks:
- admissionReviewVersions:
  - v1`

			result := templater.ApplyHelmSubstitutions(content, resource)

			// Should have proper conditional formatting without extra spaces
			Expect(result).To(ContainSubstring("{{- if .Values.certManager.enable }}"))
			Expect(result).To(ContainSubstring("cert-manager.io/inject-ca-from:"))
			Expect(result).To(ContainSubstring("{{- end }}"))

			// Should NOT have extra blank lines or improper indentation
			Expect(result).NotTo(ContainSubstring("{{- if .Values.certManager.enable }}\n\n"))
			Expect(result).NotTo(ContainSubstring("cert-manager.io/inject-ca-from:\n\n"))
		})

		It("should handle container args with proper indentation", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - args:
        - --webhook-cert-path=/tmp/k8s-webhook-server/serving-certs/tls.crt
        - --metrics-cert-path=/tmp/k8s-metrics-server/metrics-certs/tls.crt
        - --leader-elect
        name: manager`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			// Should have proper conditional formatting for webhook cert path
			Expect(result).To(ContainSubstring("{{- if .Values.certManager.enable }}"))
			Expect(result).To(ContainSubstring("- --webhook-cert-path=/tmp/k8s-webhook-server/serving-certs/tls.crt"))

			// Should have proper conditional formatting for metrics cert path
			Expect(result).To(ContainSubstring("{{- if and .Values.certManager.enable .Values.metrics.enable }}"))
			Expect(result).To(ContainSubstring("- --metrics-cert-path=/tmp/k8s-metrics-server/metrics-certs/tls.crt"))

			// Should NOT have extra blank lines
			Expect(result).NotTo(ContainSubstring("{{- if .Values.certManager.enable }}\n\n"))
		})

		It("should handle volume mounts with proper indentation", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - volumeMounts:
        - mountPath: /tmp/k8s-webhook-server/serving-certs
          name: webhook-certs
          readOnly: true
        - mountPath: /tmp/k8s-metrics-server/metrics-certs
          name: metrics-certs
          readOnly: true`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			// Should have conditional blocks for webhook certs
			Expect(result).To(ContainSubstring("{{- if .Values.certManager.enable }}"))
			Expect(result).To(ContainSubstring("mountPath: /tmp/k8s-webhook-server/serving-certs"))

			// Should have conditional blocks for metrics certs
			Expect(result).To(ContainSubstring("{{- if and .Values.certManager.enable .Values.metrics.enable }}"))
			Expect(result).To(ContainSubstring("mountPath: /tmp/k8s-metrics-server/metrics-certs"))
		})

		It("should handle namespace substitution correctly", func() {
			serviceResource := &unstructured.Unstructured{}
			serviceResource.SetAPIVersion("v1")
			serviceResource.SetKind("Service")
			serviceResource.SetName("test-project-webhook-service")

			content := `apiVersion: v1
kind: Service
metadata:
  name: test-project-webhook-service
  namespace: test-project-system
spec:
  type: ClusterIP`

			result := templater.ApplyHelmSubstitutions(content, serviceResource)

			// Should substitute namespace with Helm template
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
			Expect(result).NotTo(ContainSubstring("namespace: test-project-system"))
		})

		It("should preserve annotations without modification", func() {
			webhookResource := &unstructured.Unstructured{}
			webhookResource.SetAPIVersion("admissionregistration.k8s.io/v1")
			webhookResource.SetKind("ValidatingWebhookConfiguration")
			webhookResource.SetName("test-project-validating-webhook-configuration")

			content := `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  annotations:
    cert-manager.io/inject-ca-from: test-system/test-serving-cert
  name: test-project-validating-webhook-configuration`

			result := templater.ApplyHelmSubstitutions(content, webhookResource)

			// Should preserve existing kustomize annotations as-is
			Expect(result).To(ContainSubstring("cert-manager.io/inject-ca-from"))

			// Should NOT add extra Helm metadata injection
			Expect(result).NotTo(ContainSubstring(`{{- include "chart.labels"`))
			Expect(result).NotTo(ContainSubstring(`{{- include "chart.annotations"`))
		})
	})

	Context("conditional wrapping", func() {
		It("should add metrics conditional for ServiceMonitor resources", func() {
			serviceMonitorResource := &unstructured.Unstructured{}
			serviceMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			serviceMonitorResource.SetKind("ServiceMonitor")
			serviceMonitorResource.SetName("test-project-controller-manager-metrics-monitor")

			content := `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-project-controller-manager-metrics-monitor`

			result := templater.ApplyHelmSubstitutions(content, serviceMonitorResource)

			// Should be wrapped with prometheus enable conditional
			Expect(result).To(ContainSubstring("{{- if .Values.prometheus.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should add metrics conditional for metrics services", func() {
			serviceResource := &unstructured.Unstructured{}
			serviceResource.SetAPIVersion("v1")
			serviceResource.SetKind("Service")
			serviceResource.SetName("test-project-controller-manager-metrics-service")

			content := `apiVersion: v1
kind: Service
metadata:
  name: test-project-controller-manager-metrics-service`

			result := templater.ApplyHelmSubstitutions(content, serviceResource)

			// Should be wrapped with metrics enable conditional
			Expect(result).To(ContainSubstring("{{- if .Values.metrics.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should add cert-manager conditional for Certificate resources", func() {
			certResource := &unstructured.Unstructured{}
			certResource.SetAPIVersion("cert-manager.io/v1")
			certResource.SetKind("Certificate")
			certResource.SetName("test-project-serving-cert")

			content := `apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-project-serving-cert`

			result := templater.ApplyHelmSubstitutions(content, certResource)

			// Should be wrapped with certManager enable conditional
			Expect(result).To(ContainSubstring("{{- if .Values.certManager.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should add combined conditionals for metrics certificates", func() {
			certResource := &unstructured.Unstructured{}
			certResource.SetAPIVersion("cert-manager.io/v1")
			certResource.SetKind("Certificate")
			certResource.SetName("test-project-metrics-certs")

			content := `apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-project-metrics-certs`

			result := templater.ApplyHelmSubstitutions(content, certResource)

			// Should be wrapped with both metrics and certManager conditionals
			Expect(result).To(ContainSubstring("{{- if and .Values.certManager.enable .Values.metrics.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should NOT add conditionals to essential resources", func() {
			// Test essential RBAC
			clusterRoleResource := &unstructured.Unstructured{}
			clusterRoleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			clusterRoleResource.SetKind("ClusterRole")
			clusterRoleResource.SetName("test-project-manager-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-manager-role`

			result := templater.ApplyHelmSubstitutions(content, clusterRoleResource)

			// Should NOT wrap essential RBAC with conditionals
			Expect(result).NotTo(ContainSubstring("{{- if .Values"))

			// Test webhook service (also essential)
			serviceResource := &unstructured.Unstructured{}
			serviceResource.SetAPIVersion("v1")
			serviceResource.SetKind("Service")
			serviceResource.SetName("test-project-webhook-service")

			webhookContent := `apiVersion: v1
kind: Service
metadata:
  name: test-project-webhook-service`

			webhookResult := templater.ApplyHelmSubstitutions(webhookContent, serviceResource)

			// Should NOT wrap webhook service with conditionals (it's essential)
			Expect(webhookResult).NotTo(ContainSubstring("{{- if .Values"))
		})

		It("should NOT add cert-manager conditionals to webhook configurations", func() {
			mutatingWebhookResource := &unstructured.Unstructured{}
			mutatingWebhookResource.SetAPIVersion("admissionregistration.k8s.io/v1")
			mutatingWebhookResource.SetKind("MutatingWebhookConfiguration")
			mutatingWebhookResource.SetName("test-project-mutating-webhook-configuration")

			content := `apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: test-project-mutating-webhook-configuration`

			result := templater.ApplyHelmSubstitutions(content, mutatingWebhookResource)

			// Webhook configurations should NOT be conditional on cert-manager
			// (they're essential and cert-manager is optional)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.certManager.enable }}"))
		})
	})

	Context("helper RBAC wrapping", func() {
		It("should add rbacHelpers conditional for helper RBAC roles", func() {
			clusterRoleResource := &unstructured.Unstructured{}
			clusterRoleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			clusterRoleResource.SetKind("ClusterRole")
			clusterRoleResource.SetName("test-project-memcached-editor-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-memcached-editor-role`

			result := templater.ApplyHelmSubstitutions(content, clusterRoleResource)

			// Should be wrapped with rbacHelpers conditional
			Expect(result).To(ContainSubstring("{{- if .Values.rbacHelpers.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should add rbacHelpers conditional for helper ClusterRoleBindings", func() {
			bindingResource := &unstructured.Unstructured{}
			bindingResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			bindingResource.SetKind("ClusterRoleBinding")
			bindingResource.SetName("test-project-memcached-viewer-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: test-project-memcached-viewer-rolebinding`

			result := templater.ApplyHelmSubstitutions(content, bindingResource)

			// Should be wrapped with rbacHelpers conditional
			Expect(result).To(ContainSubstring("{{- if .Values.rbacHelpers.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})
	})

	Context("project name handling", func() {
		It("should preserve project names as-is (no templating)", func() {
			serviceAccountResource := &unstructured.Unstructured{}
			serviceAccountResource.SetAPIVersion("v1")
			serviceAccountResource.SetKind("ServiceAccount")
			serviceAccountResource.SetName("test-project-controller-manager")

			content := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-project-controller-manager
  namespace: test-project-system`

			result := templater.ApplyHelmSubstitutions(content, serviceAccountResource)

			// Should keep project name as-is for resource names
			Expect(result).To(ContainSubstring("name: test-project-controller-manager"))
			Expect(result).NotTo(ContainSubstring("{{ include"))
		})
	})

	Context("edge cases", func() {
		It("should handle empty content", func() {
			testResource := &unstructured.Unstructured{}
			testResource.SetKind("ConfigMap")

			result := templater.ApplyHelmSubstitutions("", testResource)
			Expect(result).To(BeEmpty())
		})

		It("should handle resources without namespace", func() {
			clusterRoleResource := &unstructured.Unstructured{}
			clusterRoleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			clusterRoleResource.SetKind("ClusterRole")
			clusterRoleResource.SetName("test-project-manager-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-manager-role`

			result := templater.ApplyHelmSubstitutions(content, clusterRoleResource)

			// Should not add namespace substitution for cluster-scoped resources
			Expect(result).NotTo(ContainSubstring("namespace:"))
		})

		It("should handle malformed YAML gracefully", func() {
			testResource := &unstructured.Unstructured{}
			testResource.SetKind("ConfigMap")

			malformedContent := "not: valid: yaml: content:"
			result := templater.ApplyHelmSubstitutions(malformedContent, testResource)

			// Should return content as-is for malformed YAML
			Expect(result).To(Equal(malformedContent))
		})
	})

	Context("image and resources templating", func() {
		It("should template manager image, pullPolicy and resources with correct indentation", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: gcr.io/project/controller:v1.2.3
        imagePullPolicy: Always
        resources:
          limits:
            cpu: 200m
            memory: 256Mi
          requests:
            cpu: 100m
            memory: 128Mi`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			// Should template image with digest/tag conditional
			Expect(result).To(ContainSubstring(`{{- if .Values.controllerManager.image.digest }}`))
			Expect(result).To(ContainSubstring(`image: "{{ .Values.controllerManager.image.repository }}@{{ .Values.controllerManager.image.digest }}"`))
			Expect(result).To(ContainSubstring(`{{- else }}`))
			Expect(result).To(ContainSubstring(`image: "{{ .Values.controllerManager.image.repository }}:{{ .Values.controllerManager.image.tag }}"`))
			Expect(result).To(ContainSubstring(`{{- end }}`))

			// Should template pullPolicy with quotes
			Expect(result).To(ContainSubstring(`imagePullPolicy: "{{ .Values.controllerManager.image.pullPolicy }}"`))

			// Should template resources with correct indentation (8 + 2 = 10 spaces for nindent)
			Expect(result).To(ContainSubstring("{{- with .Values.controllerManager.resources }}"))
			Expect(result).To(ContainSubstring("{{- toYaml . | nindent 10 }}"))

			// Should NOT have hardcoded values
			Expect(result).NotTo(ContainSubstring("gcr.io/project/controller:v1.2.3"))
			Expect(result).NotTo(ContainSubstring("imagePullPolicy: Always"))
			Expect(result).NotTo(ContainSubstring("cpu: 200m"))

			// Should NOT have double indentation (no literal spaces before toYaml)
			Expect(result).NotTo(ContainSubstring("  {{- toYaml"))
		})

		It("should preserve env and securityContext unchanged", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        env:
        - name: CLUSTER_DOMAIN
          value: cluster.local
        securityContext:
          runAsNonRoot: true`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			// env and securityContext should be preserved exactly
			Expect(result).To(ContainSubstring("CLUSTER_DOMAIN"))
			Expect(result).To(ContainSubstring("cluster.local"))
			Expect(result).To(ContainSubstring("runAsNonRoot: true"))
		})

		It("should template image with digest OR tag conditional logic", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: gcr.io/project/controller@sha256:abc123def456
        imagePullPolicy: Always`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			// Should template image with digest/tag conditional (same as tag test since we use conditional logic)
			Expect(result).To(ContainSubstring(`{{- if .Values.controllerManager.image.digest }}`))
			Expect(result).To(ContainSubstring(`image: "{{ .Values.controllerManager.image.repository }}@{{ .Values.controllerManager.image.digest }}"`))

			// Should template pullPolicy
			Expect(result).To(ContainSubstring(`imagePullPolicy: "{{ .Values.controllerManager.image.pullPolicy }}"`))

			// Should NOT have hardcoded values
			Expect(result).NotTo(ContainSubstring("gcr.io/project/controller@sha256:abc123def456"))
			Expect(result).NotTo(ContainSubstring("imagePullPolicy: Always"))
		})
	})
})
