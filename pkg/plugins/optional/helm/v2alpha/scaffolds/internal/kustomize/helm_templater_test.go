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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// Test expectation constants for test-project.resourceName templates
	expectedIssuerName = `name: {{ include "test-project.resourceName" (dict "suffix" "selfsigned-issuer" "context" $) }}`
)

var _ = Describe("HelmTemplater", func() {
	var templater *HelmTemplater

	BeforeEach(func() {
		templater = &HelmTemplater{
			detectedPrefix:   "test-project",
			chartName:        "test-project",
			managerNamespace: "test-project-system",
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
			// Should replace app.kubernetes.io/name with chart name template
			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
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
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
        - --webhook-cert-path=/tmp/k8s-webhook-server/serving-certs/tls.crt
        - --metrics-cert-path=/tmp/k8s-metrics-server/metrics-certs/tls.crt
        - --leader-elect
        env:
        - name: BUSYBOX_IMAGE
          value: busybox:1.36.1
        image: controller:latest
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
        name: manager
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      serviceAccountName: test-project-controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			Expect(result).To(ContainSubstring("{{- if .Values.metrics.enable }}"))
			Expect(result).To(ContainSubstring("- --metrics-bind-address=:{{ .Values.metrics.port }}"))
			Expect(result).To(ContainSubstring("- --metrics-bind-address=0"))
			Expect(result).To(ContainSubstring("- --health-probe-bind-address=:8081"))
			Expect(result).To(ContainSubstring("{{- range .Values.manager.args }}"))
			Expect(result).NotTo(ContainSubstring("BUSYBOX_IMAGE"))
			Expect(result).NotTo(ContainSubstring("MEMCACHED_IMAGE"))
			Expect(result).To(ContainSubstring("image: " +
				"\"{{ .Values.manager.image.repository }}:{{ .Values.manager.image.tag }}\""))
			Expect(result).To(ContainSubstring("imagePullPolicy: {{ .Values.manager.image.pullPolicy }}"))
			Expect(result).NotTo(ContainSubstring("controller:latest"))
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

		It("should template imagePullSecrets", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      imagePullSecrets:
      - name: test-secret
      containers:
      - args:
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
        - --webhook-cert-path=/tmp/k8s-webhook-server/serving-certs/tls.crt
        - --metrics-cert-path=/tmp/k8s-metrics-server/metrics-certs/tls.crt
        - --leader-elect`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			Expect(result).To(ContainSubstring("imagePullSecrets:"))
			Expect(result).NotTo(ContainSubstring("test-secret"))
		})

		It("should template empty imagePullSecrets", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      imagePullSecrets: []
      containers:
      - args:
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
        - --webhook-cert-path=/tmp/k8s-webhook-server/serving-certs/tls.crt
        - --metrics-cert-path=/tmp/k8s-metrics-server/metrics-certs/tls.crt
        - --leader-elect`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			Expect(result).To(ContainSubstring("imagePullSecrets:"))
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
		})

		It("should add webhook conditional for webhook services", func() {
			serviceResource := &unstructured.Unstructured{}
			serviceResource.SetAPIVersion("v1")
			serviceResource.SetKind("Service")
			serviceResource.SetName("test-project-webhook-service")

			webhookContent := `apiVersion: v1
kind: Service
metadata:
  name: test-project-webhook-service`

			webhookResult := templater.ApplyHelmSubstitutions(webhookContent, serviceResource)

			// Should wrap webhook service with webhook.enable conditional
			Expect(webhookResult).To(ContainSubstring("{{- if .Values.webhook.enable }}"))
			Expect(webhookResult).To(ContainSubstring("{{- end }}"))
		})

		It("should add webhook conditional for webhook configurations", func() {
			mutatingWebhookResource := &unstructured.Unstructured{}
			mutatingWebhookResource.SetAPIVersion("admissionregistration.k8s.io/v1")
			mutatingWebhookResource.SetKind("MutatingWebhookConfiguration")
			mutatingWebhookResource.SetName("test-project-mutating-webhook-configuration")

			content := `apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: test-project-mutating-webhook-configuration`

			result := templater.ApplyHelmSubstitutions(content, mutatingWebhookResource)

			// Webhook configurations should be conditional on webhook.enable
			Expect(result).To(ContainSubstring("{{- if .Values.webhook.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should add webhook conditional for validating webhook configurations", func() {
			validatingWebhookResource := &unstructured.Unstructured{}
			validatingWebhookResource.SetAPIVersion("admissionregistration.k8s.io/v1")
			validatingWebhookResource.SetKind("ValidatingWebhookConfiguration")
			validatingWebhookResource.SetName("test-project-validating-webhook-configuration")

			content := `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  annotations:
    cert-manager.io/inject-ca-from: test-project-system/test-project-serving-cert
  name: test-project-validating-webhook-configuration`

			result := templater.ApplyHelmSubstitutions(content, validatingWebhookResource)

			// Webhook configurations should be wrapped with webhook.enable
			Expect(result).To(ContainSubstring("{{- if .Values.webhook.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
			// Cert-manager annotation should still be conditional on certManager.enable
			Expect(result).To(ContainSubstring("{{- if .Values.certManager.enable }}"))
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

	Context("chart.fullname templating", func() {
		It("should template resource names with test-project.resourceName for proper truncation", func() {
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

			// Should template with test-project.resourceName which handles 63-char truncation
			expected := `name: {{ include "test-project.resourceName" (dict "suffix" "controller-manager" "context" $) }}`
			Expect(result).To(ContainSubstring(expected))
			Expect(result).NotTo(ContainSubstring("name: test-project-controller-manager"))
		})
		It("should template ServiceMonitor name with test-project.resourceName for proper truncation", func() {
			serviceMonitorResource := &unstructured.Unstructured{}
			serviceMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			serviceMonitorResource.SetKind("ServiceMonitor")
			serviceMonitorResource.SetName("test-project-controller-manager-metrics-monitor")

			content := `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-project-controller-manager-metrics-monitor`

			result := templater.ApplyHelmSubstitutions(content, serviceMonitorResource)

			// Should template with test-project.resourceName which handles 63-char truncation
			expected := `name: {{ include "test-project.resourceName" ` +
				`(dict "suffix" "controller-manager-metrics-monitor" "context" $) }}`
			Expect(result).To(ContainSubstring(expected))
			Expect(result).NotTo(ContainSubstring("name: test-project-controller-manager-metrics-monitor"))
		})
	})

	Context("app.kubernetes.io/name label templating", func() {
		It("should template app.kubernetes.io/name for Deployment", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: test-project
    control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should template app.kubernetes.io/name for Service", func() {
			service := &unstructured.Unstructured{}
			service.SetAPIVersion("v1")
			service.SetKind("Service")
			service.SetName("test-project-webhook-service")

			content := `apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: test-project`

			result := templater.ApplyHelmSubstitutions(content, service)

			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should template app.kubernetes.io/name for ServiceAccount", func() {
			sa := &unstructured.Unstructured{}
			sa.SetAPIVersion("v1")
			sa.SetKind("ServiceAccount")
			sa.SetName("test-project-controller-manager")

			content := `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project`

			result := templater.ApplyHelmSubstitutions(content, sa)

			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should template app.kubernetes.io/name for ClusterRole", func() {
			clusterRole := &unstructured.Unstructured{}
			clusterRole.SetAPIVersion("rbac.authorization.k8s.io/v1")
			clusterRole.SetKind("ClusterRole")
			clusterRole.SetName("test-project-manager-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app.kubernetes.io/name: test-project`

			result := templater.ApplyHelmSubstitutions(content, clusterRole)

			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should template app.kubernetes.io/name for Role", func() {
			role := &unstructured.Unstructured{}
			role.SetAPIVersion("rbac.authorization.k8s.io/v1")
			role.SetKind("Role")
			role.SetName("test-project-leader-election-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  labels:
    app.kubernetes.io/name: test-project`

			result := templater.ApplyHelmSubstitutions(content, role)

			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should template app.kubernetes.io/name for RoleBinding", func() {
			rb := &unstructured.Unstructured{}
			rb.SetAPIVersion("rbac.authorization.k8s.io/v1")
			rb.SetKind("RoleBinding")
			rb.SetName("test-project-leader-election-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    app.kubernetes.io/name: test-project`

			result := templater.ApplyHelmSubstitutions(content, rb)

			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should template app.kubernetes.io/name for ClusterRoleBinding", func() {
			crb := &unstructured.Unstructured{}
			crb.SetAPIVersion("rbac.authorization.k8s.io/v1")
			crb.SetKind("ClusterRoleBinding")
			crb.SetName("test-project-manager-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    app.kubernetes.io/name: test-project`

			result := templater.ApplyHelmSubstitutions(content, crb)

			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should template app.kubernetes.io/name for Certificate", func() {
			cert := &unstructured.Unstructured{}
			cert.SetAPIVersion("cert-manager.io/v1")
			cert.SetKind("Certificate")
			cert.SetName("test-project-serving-cert")

			content := `apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  labels:
    app.kubernetes.io/name: test-project`

			result := templater.ApplyHelmSubstitutions(content, cert)

			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should template app.kubernetes.io/name for Issuer", func() {
			issuer := &unstructured.Unstructured{}
			issuer.SetAPIVersion("cert-manager.io/v1")
			issuer.SetKind("Issuer")
			issuer.SetName("test-project-selfsigned-issuer")

			content := `apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  labels:
    app.kubernetes.io/name: test-project`

			result := templater.ApplyHelmSubstitutions(content, issuer)

			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should handle label already templated without breaking", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: {{ include "test-project.name" . }}
    control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should keep the template as-is
			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).ToNot(ContainSubstring("app.kubernetes.io/name: test-project"))
		})

		It("should template multiple occurrences in same resource", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: test-project
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: test-project
  template:
    metadata:
      labels:
        app.kubernetes.io/name: test-project`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// All three should be templated
			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: test-project"))
			// Count occurrences - should be 3
			count := strings.Count(result, "app.kubernetes.io/name: {{ include \"test-project.name\" . }}")
			Expect(count).To(Equal(3))
		})
	})

	Context("existing Go template syntax escaping", func() {
		It("should escape existing Go template syntax in CRD samples", func() {
			crdResource := &unstructured.Unstructured{}
			crdResource.SetAPIVersion("apiextensions.k8s.io/v1")
			crdResource.SetKind("CustomResourceDefinition")
			crdResource.SetName("changetransferpolicies.promoter.argoproj.io")

			content := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: changetransferpolicies.promoter.argoproj.io
spec:
  names:
    kind: ChangeTransferPolicy
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              pullRequest:
                properties:
                  template:
                    properties:
                      description:
                        default: "Promoting {{ .ChangeTransferPolicy.Spec.ActiveBranch }}"
                        type: string
                      title:
                        default: "Promote {{ trunc 5 .ChangeTransferPolicy.Status.Proposed.Dry.Sha }}"
                        type: string`

			result := templater.ApplyHelmSubstitutions(content, crdResource)

			// Existing {{ }} should be escaped to {{ "{{ ... }}" }}
			Expect(result).To(ContainSubstring(`{{ "{{ .ChangeTransferPolicy.Spec.ActiveBranch }}" }}`),
				"existing template syntax should be escaped")
			Expect(result).To(ContainSubstring(`{{ "{{ trunc 5 .ChangeTransferPolicy.Status.Proposed.Dry.Sha }}" }}`),
				"function calls in templates should be escaped")

			// Should NOT have unescaped Go template syntax (which would break Helm)
			// We check that all ChangeTransferPolicy references are properly wrapped
			// Pattern checks for: default: "...<text>{{ .ChangeTransferPolicy" (not escaped)
			// The properly escaped version is: default: "...{{ "{{ .ChangeTransferPolicy..." }}"
			Expect(result).NotTo(MatchRegexp(`default:\s+"[^{]*\{\{\s*\.ChangeTransferPolicy`),
				"unescaped Go templates should not exist in default values")
		})

		It("should escape multiple template expressions on the same line", func() {
			crdResource := &unstructured.Unstructured{}
			crdResource.SetAPIVersion("apiextensions.k8s.io/v1")
			crdResource.SetKind("CustomResourceDefinition")
			crdResource.SetName("policies.example.com")

			content := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  versions:
  - schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              message:
                default: "From {{ .Source.Branch }} to {{ .Target.Branch }}"`

			result := templater.ApplyHelmSubstitutions(content, crdResource)

			// Both templates should be escaped (applies to all resources)
			Expect(result).To(ContainSubstring(`{{ "{{ .Source.Branch }}" }}`))
			Expect(result).To(ContainSubstring(`{{ "{{ .Target.Branch }}" }}`))
		})

		It("should escape templates with special characters", func() {
			crdResource := &unstructured.Unstructured{}
			crdResource.SetAPIVersion("apiextensions.k8s.io/v1")
			crdResource.SetKind("CustomResourceDefinition")
			crdResource.SetName("configs.example.com")

			content := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  versions:
  - schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              value:
                default: "Value: {{ .Config.Key-With-Dashes }}"`

			result := templater.ApplyHelmSubstitutions(content, crdResource)

			Expect(result).To(ContainSubstring(`{{ "{{ .Config.Key-With-Dashes }}" }}`))
		})

		It("should handle template syntax with quotes correctly", func() {
			crdResource := &unstructured.Unstructured{}
			crdResource.SetAPIVersion("apiextensions.k8s.io/v1")
			crdResource.SetKind("CustomResourceDefinition")
			crdResource.SetName("messages.example.com")

			content := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  versions:
  - schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              template:
                default: '{{ .Config.Message "default" }}'`

			result := templater.ApplyHelmSubstitutions(content, crdResource)

			// Quotes inside templates should be escaped
			Expect(result).To(ContainSubstring(`{{ "{{ .Config.Message \"default\" }}" }}`))
		})

		It("should escape templates in ConfigMaps and other non-CRD resources", func() {
			configMapResource := &unstructured.Unstructured{}
			configMapResource.SetAPIVersion("v1")
			configMapResource.SetKind("ConfigMap")
			configMapResource.SetName("template-config")
			configMapResource.SetNamespace("test-project-system")

			// ANY resource can have Go template syntax that needs escaping
			// Examples: ConfigMaps with notification templates, Secrets with webhook URLs,
			// Deployment annotations with CI/CD metadata, etc.
			content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: template-config
  namespace: test-project-system
  labels:
    app.kubernetes.io/name: test-project
data:
  notification: "Deployed from {{ .Source.Branch }} to {{ .Target.Branch }}"`

			result := templater.ApplyHelmSubstitutions(content, configMapResource)

			// Existing templates should be escaped (applies to ALL resources, not just CRDs)
			Expect(result).To(ContainSubstring(`{{ "{{ .Source.Branch }}" }}`))
			Expect(result).To(ContainSubstring(`{{ "{{ .Target.Branch }}" }}`))

			// Helm templates should still be added normally
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
			Expect(result).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`))
		})

		It("should handle content without any templates", func() {
			configMapResource := &unstructured.Unstructured{}
			configMapResource.SetAPIVersion("v1")
			configMapResource.SetKind("ConfigMap")
			configMapResource.SetName("no-template")

			content := `apiVersion: v1
kind: ConfigMap
data:
  message: "No templates here"`

			result := templater.ApplyHelmSubstitutions(content, configMapResource)

			// Should not add any escaping
			Expect(result).To(ContainSubstring(`message: "No templates here"`))
			Expect(result).NotTo(ContainSubstring(`{{ "{{`))
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

	Context("namespace-scoped RBAC resources", func() {
		It("should preserve explicit namespace in Role for cross-namespace permissions", func() {
			roleResource := &unstructured.Unstructured{}
			roleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			roleResource.SetKind("Role")
			roleResource.SetName("test-project-manager-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-project-manager-role
  namespace: infrastructure
  labels:
    app.kubernetes.io/name: test-project
rules:
- apiGroups:
  - apps
  resources:
  - deployments
  verbs:
  - get
  - list
  - patch
  - update
  - watch`

			result := templater.ApplyHelmSubstitutions(content, roleResource)

			// Namespace should be preserved (not templated) for cross-namespace permissions
			Expect(result).To(ContainSubstring("namespace: infrastructure"),
				"explicit namespace should be preserved for cross-namespace Role")
			Expect(result).NotTo(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"explicit namespace should NOT be templated to Release.Namespace")

			// Labels should still be templated
			Expect(result).To(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.name" . }}`))

			// Name should be templated
			Expect(result).To(ContainSubstring(`name: {{ include "test-project.resourceName"`))

			// Rules should be preserved
			Expect(result).To(ContainSubstring("- apps"))
			Expect(result).To(ContainSubstring("- deployments"))
		})

		It("should preserve explicit namespace in Role for leader election", func() {
			roleResource := &unstructured.Unstructured{}
			roleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			roleResource.SetKind("Role")
			roleResource.SetName("test-project-leader-election-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-project-leader-election-role
  namespace: production
  labels:
    app.kubernetes.io/name: test-project
rules:
- apiGroups:
  - coordination.k8s.io
  resources:
  - leases
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - patch
  - update`

			result := templater.ApplyHelmSubstitutions(content, roleResource)

			// Namespace should be preserved for cross-namespace leader election
			Expect(result).To(ContainSubstring("namespace: production"),
				"explicit namespace should be preserved for cross-namespace leader election Role")

			// Verify leader election permissions
			Expect(result).To(ContainSubstring("- coordination.k8s.io"))
			Expect(result).To(ContainSubstring("- leases"))
			Expect(result).To(ContainSubstring("- events"))
		})

		It("should preserve explicit namespace in RoleBinding metadata", func() {
			roleBindingResource := &unstructured.Unstructured{}
			roleBindingResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			roleBindingResource.SetKind("RoleBinding")
			roleBindingResource.SetName("test-project-manager-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-manager-rolebinding
  namespace: infrastructure
  labels:
    app.kubernetes.io/name: test-project
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: test-project-manager-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system`

			result := templater.ApplyHelmSubstitutions(content, roleBindingResource)

			// RoleBinding metadata namespace should be preserved
			Expect(result).To(ContainSubstring("metadata:\n  name:"))
			Expect(result).To(ContainSubstring("namespace: infrastructure"),
				"RoleBinding metadata namespace should be preserved")

			// Subject namespace should be templated (references the controller namespace)
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"subject namespace should be templated to Release.Namespace")

			// Name references should be templated
			Expect(result).To(ContainSubstring(`name: {{ include "test-project.resourceName"`))
		})

		It("should template Role namespace when it matches project namespace", func() {
			roleResource := &unstructured.Unstructured{}
			roleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			roleResource.SetKind("Role")
			roleResource.SetName("test-project-leader-election-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-project-leader-election-role
  namespace: test-project-system
  labels:
    app.kubernetes.io/name: test-project
rules:
- apiGroups:
  - coordination.k8s.io
  resources:
  - leases
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete`

			result := templater.ApplyHelmSubstitutions(content, roleResource)

			// When namespace matches project namespace, it should be templated
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"Role namespace should be templated when it matches project namespace")
			Expect(result).NotTo(ContainSubstring("namespace: test-project-system"),
				"project namespace should be templated, not preserved")
		})

		It("should handle RoleBinding with multiple subjects correctly", func() {
			roleBindingResource := &unstructured.Unstructured{}
			roleBindingResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			roleBindingResource.SetKind("RoleBinding")
			roleBindingResource.SetName("test-project-manager-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-manager-rolebinding
  namespace: infrastructure
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: test-project-manager-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system
- kind: ServiceAccount
  name: test-project-webhook
  namespace: test-project-system`

			result := templater.ApplyHelmSubstitutions(content, roleBindingResource)

			// Both subject namespaces should be templated
			subjectNamespaceCount := strings.Count(result, "namespace: {{ .Release.Namespace }}")
			Expect(subjectNamespaceCount).To(BeNumerically(">=", 2),
				"both subject namespaces should be templated")

			// RoleBinding metadata namespace should be preserved
			Expect(result).To(ContainSubstring("namespace: infrastructure"))
		})

		It("should preserve resource names when namespace appears as substring", func() {
			// Critical: namespace "user" must NOT break resource name "users"
			// This validates field-aware replacement prevents substring corruption
			roleResource := &unstructured.Unstructured{}
			roleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			roleResource.SetKind("ClusterRole")
			roleResource.SetName("manager-role")

			// Scenario: manager namespace is "user", CRD resource is "users"
			customTemplater := &HelmTemplater{
				detectedPrefix:   "test-project",
				chartName:        "test-project",
				managerNamespace: "user", // Short namespace that appears as substring
			}

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-manager-role
rules:
- apiGroups:
  - identity.example.com
  resources:
  - users
  - users/finalizers
  - users/status
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete`

			result := customTemplater.ApplyHelmSubstitutions(content, roleResource)

			// Critical: resource name "users" must NOT be replaced with "{{ .Release.Namespace }}s"
			Expect(result).To(ContainSubstring("- users"),
				"resource name 'users' should remain unchanged")
			Expect(result).To(ContainSubstring("- users/finalizers"),
				"resource name 'users/finalizers' should remain unchanged")
			Expect(result).To(ContainSubstring("- users/status"),
				"resource name 'users/status' should remain unchanged")

			// Ensure we didn't create templated resource names
			Expect(result).NotTo(ContainSubstring("- {{ .Release.Namespace }}s"),
				"must NOT replace 'user' substring in resource names")
			Expect(result).NotTo(MatchRegexp(`resources:\s*-\s*\{\{.*\}\}`),
				"resource names must never be templated")
		})

		It("should handle edge case where namespace is substring of multiple fields", func() {
			// Test more edge cases: namespace "app" appears in "applications", "apps", etc.
			customTemplater := &HelmTemplater{
				detectedPrefix:   "test-project",
				chartName:        "test-project",
				managerNamespace: "app",
			}

			roleBindingResource := &unstructured.Unstructured{}
			roleBindingResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			roleBindingResource.SetKind("RoleBinding")
			roleBindingResource.SetName("manager-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-manager-rolebinding
  namespace: app
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: test-project-manager-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: app`

			result := customTemplater.ApplyHelmSubstitutions(content, roleBindingResource)

			// Namespace fields should be templated
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"),
				"namespace fields should be templated")

			// Verify we have exactly 2 namespace template substitutions (metadata + subject)
			namespaceTemplateCount := strings.Count(result, "namespace: {{ .Release.Namespace }}")
			Expect(namespaceTemplateCount).To(Equal(2),
				"should have exactly 2 namespace field replacements")

			// Verify apiGroup field is NOT affected (contains "app" in "rbac.authorization.k8s.io")
			Expect(result).To(ContainSubstring("apiGroup: rbac.authorization.k8s.io"),
				"apiGroup should not be affected by namespace replacement")
		})

		It("should handle ALL Kubernetes DNS patterns generically", func() {
			// This test validates DNS replacement works for ANY K8s DNS pattern
			configMapResource := &unstructured.Unstructured{}
			configMapResource.SetAPIVersion("v1")
			configMapResource.SetKind("ConfigMap")
			configMapResource.SetName("dns-config")

			content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: dns-config
  namespace: test-project-system
data:
  # Standard service DNS
  service-short: api.test-project-system.svc
  service-full: api.test-project-system.svc.cluster.local
  service-port: api.test-project-system.svc:8080
  service-path: https://api.test-project-system.svc.cluster.local:443/v1
  
  # Pod DNS
  pod-dns: my-pod.test-project-system.pod.cluster.local
  
  # Endpoints DNS  
  endpoints-dns: my-service.test-project-system.endpoints.cluster.local
  
  # Headless service (StatefulSet)
  stateful-0: app-0.app-headless.test-project-system.svc.cluster.local
  stateful-1: app-1.app-headless.test-project-system.svc.cluster.local
  
  # External namespace should be preserved
  external-svc: monitoring.monitoring-system.svc.cluster.local`

			result := templater.ApplyHelmSubstitutions(content, configMapResource)

			// Verify ALL manager namespace DNS patterns are templated
			Expect(result).To(ContainSubstring("api.{{ .Release.Namespace }}.svc"))
			Expect(result).To(ContainSubstring("api.{{ .Release.Namespace }}.svc.cluster.local"))
			Expect(result).To(ContainSubstring("api.{{ .Release.Namespace }}.svc:8080"))
			Expect(result).To(ContainSubstring("api.{{ .Release.Namespace }}.svc.cluster.local:443"))
			Expect(result).To(ContainSubstring("my-pod.{{ .Release.Namespace }}.pod.cluster.local"))
			Expect(result).To(ContainSubstring("my-service.{{ .Release.Namespace }}.endpoints.cluster.local"))
			Expect(result).To(ContainSubstring("app-0.app-headless.{{ .Release.Namespace }}.svc.cluster.local"))
			Expect(result).To(ContainSubstring("app-1.app-headless.{{ .Release.Namespace }}.svc.cluster.local"))

			// Verify NO hardcoded manager namespace remains
			Expect(result).NotTo(ContainSubstring(".test-project-system.svc"))
			Expect(result).NotTo(ContainSubstring(".test-project-system.pod"))
			Expect(result).NotTo(ContainSubstring(".test-project-system.endpoints"))

			// Verify external namespace is preserved
			Expect(result).To(ContainSubstring("monitoring.monitoring-system.svc.cluster.local"))
		})

		It("should NOT replace namespace-like strings in non-DNS contexts", func() {
			// Edge case: ensure we don't break strings that happen to contain the namespace
			configMapResource := &unstructured.Unstructured{}
			configMapResource.SetAPIVersion("v1")
			configMapResource.SetKind("ConfigMap")
			configMapResource.SetName("edge-cases")

			// Using "app" as namespace to test substring issues
			customTemplater := &HelmTemplater{
				detectedPrefix:   "test",
				chartName:        "test",
				managerNamespace: "app",
			}

			content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: edge-cases
  namespace: app
data:
  # DNS patterns - should be templated
  service-url: http://api.app.svc:8080
  
  # NOT DNS patterns - should be preserved
  app-name: "my-application"
  app-version: "v1.2.3"
  mapping: "application-mapping"
  labels: "app=frontend,app.kubernetes.io/name=myapp"
  
  # Tricky: "app" in various contexts
  erapplication: "some-value"
  wrapperapp: "another-value"`

			result := customTemplater.ApplyHelmSubstitutions(content, configMapResource)

			// DNS pattern should be templated
			Expect(result).To(ContainSubstring("api.{{ .Release.Namespace }}.svc:8080"))

			// Non-DNS occurrences should be preserved
			Expect(result).To(ContainSubstring(`app-name: "my-application"`))
			Expect(result).To(ContainSubstring("app-version"))
			Expect(result).To(ContainSubstring("mapping: \"application-mapping\""))
			Expect(result).To(ContainSubstring("app=frontend"))
			Expect(result).To(ContainSubstring("app.kubernetes.io/name=myapp"))
			Expect(result).To(ContainSubstring("erapplication"))
			Expect(result).To(ContainSubstring("wrapperapp"))

			// Namespace field should be templated
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
		})

		It("should handle ANY resource type with namespace references (generic test)", func() {
			// This test validates that namespace replacement is GENERIC and works
			// for any resource type, including custom resources in extras/ directory

			// Test with a custom ConfigMap (common in extras/)
			configMapResource := &unstructured.Unstructured{}
			configMapResource.SetAPIVersion("v1")
			configMapResource.SetKind("ConfigMap")
			configMapResource.SetName("custom-config")

			content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-config
  namespace: test-project-system
data:
  service-url: http://api-service.test-project-system.svc.cluster.local:8080
  webhook-endpoint: https://webhook.test-project-system.svc:9443/validate
  annotation-ref: "test-project-system/my-resource"`

			result := templater.ApplyHelmSubstitutions(content, configMapResource)

			// 1. Namespace field should be templated
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
			Expect(result).NotTo(ContainSubstring("namespace: test-project-system"))

			// 2. DNS names in data values should be templated
			Expect(result).To(ContainSubstring("http://api-service.{{ .Release.Namespace }}.svc.cluster.local:8080"))
			Expect(result).NotTo(ContainSubstring(".test-project-system.svc.cluster.local"))

			// 3. DNS names with ports should be templated
			Expect(result).To(ContainSubstring("https://webhook.{{ .Release.Namespace }}.svc:9443"))
			Expect(result).NotTo(ContainSubstring(".test-project-system.svc:9443"))

			// 4. Annotation-style references should be templated
			Expect(result).To(ContainSubstring("{{ .Release.Namespace }}/my-resource"))
			Expect(result).NotTo(ContainSubstring("test-project-system/my-resource"))
		})

		It("should handle Secret with namespace references", func() {
			secretResource := &unstructured.Unstructured{}
			secretResource.SetAPIVersion("v1")
			secretResource.SetKind("Secret")
			secretResource.SetName("app-secret")

			content := `apiVersion: v1
kind: Secret
metadata:
  name: app-secret
  namespace: test-project-system
  annotations:
    source: test-project-system/config
stringData:
  database-url: postgresql://db.test-project-system.svc:5432/mydb
  redis-url: redis://cache.test-project-system.svc.cluster.local:6379`

			result := templater.ApplyHelmSubstitutions(content, secretResource)

			// Namespace field templated
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))

			// Annotation value templated
			Expect(result).To(ContainSubstring("source: {{ .Release.Namespace }}/config"))

			// DNS names in data templated
			Expect(result).To(ContainSubstring("postgresql://db.{{ .Release.Namespace }}.svc:5432"))
			Expect(result).To(ContainSubstring("redis://cache.{{ .Release.Namespace }}.svc.cluster.local:6379"))

			// No hardcoded namespace remains
			Expect(result).NotTo(ContainSubstring(".test-project-system.svc"))
		})

		It("should handle Ingress with namespace references", func() {
			ingressResource := &unstructured.Unstructured{}
			ingressResource.SetAPIVersion("networking.k8s.io/v1")
			ingressResource.SetKind("Ingress")
			ingressResource.SetName("app-ingress")

			content := `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
  namespace: test-project-system
  annotations:
    nginx.ingress.kubernetes.io/auth-url: http://auth.test-project-system.svc.cluster.local/verify
    cert-manager.io/issuer: test-project-system/letsencrypt
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app-service
            port:
              number: 80`

			result := templater.ApplyHelmSubstitutions(content, ingressResource)

			// Namespace field templated
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))

			// Annotation DNS templated
			Expect(result).To(ContainSubstring("http://auth.{{ .Release.Namespace }}.svc.cluster.local/verify"))

			// Annotation reference templated
			Expect(result).To(ContainSubstring("cert-manager.io/issuer: {{ .Release.Namespace }}/letsencrypt"))

			// No hardcoded namespace
			Expect(result).NotTo(ContainSubstring("test-project-system/"))
			Expect(result).NotTo(ContainSubstring(".test-project-system.svc"))
		})

		It("should handle PodMonitor with namespace references", func() {
			podMonitorResource := &unstructured.Unstructured{}
			podMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			podMonitorResource.SetKind("PodMonitor")
			podMonitorResource.SetName("app-monitor")

			content := `apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: app-monitor
  namespace: test-project-system
spec:
  selector:
    matchLabels:
      app: myapp
  podMetricsEndpoints:
  - port: metrics
    scheme: https
    tlsConfig:
      serverName: metrics.test-project-system.svc
      ca:
        configMap:
          name: prometheus-ca
          namespace: test-project-system`

			result := templater.ApplyHelmSubstitutions(content, podMonitorResource)

			// Namespace field templated
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))

			// ServerName DNS templated
			Expect(result).To(ContainSubstring("serverName: metrics.{{ .Release.Namespace }}.svc"))

			// ConfigMap namespace reference templated
			namespaceCount := strings.Count(result, "namespace: {{ .Release.Namespace }}")
			Expect(namespaceCount).To(Equal(2), "both metadata and configMap namespace should be templated")

			// No hardcoded namespace in DNS
			Expect(result).NotTo(ContainSubstring(".test-project-system.svc"))
		})

		It("should handle custom CRD with multiple namespace contexts", func() {
			customResource := &unstructured.Unstructured{}
			customResource.SetAPIVersion("example.com/v1")
			customResource.SetKind("Application")
			customResource.SetName("my-app")

			content := `apiVersion: example.com/v1
kind: Application
metadata:
  name: my-app
  namespace: test-project-system
  annotations:
    backup.velero.io/backup-volumes: test-project-system/pvc
spec:
  database:
    host: postgres.test-project-system.svc.cluster.local
    port: 5432
  messaging:
    brokerURL: amqp://rabbitmq.test-project-system.svc:5672
  externalServices:
    - name: external-api
      url: https://api.external-namespace.svc/v1
  references:
    configMapRef: test-project-system/app-config
    secretRef: test-project-system/app-secret`

			result := templater.ApplyHelmSubstitutions(content, customResource)

			// Namespace field templated
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))

			// Annotation reference templated
			Expect(result).To(ContainSubstring("{{ .Release.Namespace }}/pvc"))

			// DNS names templated
			Expect(result).To(ContainSubstring("postgres.{{ .Release.Namespace }}.svc.cluster.local"))
			Expect(result).To(ContainSubstring("rabbitmq.{{ .Release.Namespace }}.svc:5672"))

			// External namespace preserved (not manager namespace)
			Expect(result).To(ContainSubstring("https://api.external-namespace.svc/v1"))

			// ConfigMap/Secret refs templated
			Expect(result).To(ContainSubstring("configMapRef: {{ .Release.Namespace }}/app-config"))
			Expect(result).To(ContainSubstring("secretRef: {{ .Release.Namespace }}/app-secret"))

			// No manager namespace remains
			Expect(result).NotTo(ContainSubstring("test-project-system/"))
			Expect(result).NotTo(ContainSubstring(".test-project-system.svc"))
		})

		It("should NOT replace namespace in non-manager context", func() {
			// Critical: cross-namespace references must be preserved
			customResource := &unstructured.Unstructured{}
			customResource.SetAPIVersion("v1")
			customResource.SetKind("ConfigMap")
			customResource.SetName("federation-config")

			content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: federation-config
  namespace: test-project-system
data:
  clusters: |
    - name: cluster-a
      apiserver: https://api.cluster-a-system.svc:6443
    - name: cluster-b
      apiserver: https://api.cluster-b-system.svc:6443
  external-service: https://monitoring.monitoring-system.svc.cluster.local:9090
  internal-service: https://internal.test-project-system.svc:8080`

			result := templater.ApplyHelmSubstitutions(content, customResource)

			// Manager namespace field templated
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
			Expect(result).NotTo(ContainSubstring("namespace: test-project-system"))

			// Manager namespace in DNS templated (appears once in internal-service)
			Expect(result).To(ContainSubstring("internal.{{ .Release.Namespace }}.svc:8080"))
			Expect(result).NotTo(ContainSubstring(".test-project-system.svc"))

			// External namespaces preserved (these don't match manager namespace)
			Expect(result).To(ContainSubstring("cluster-a-system.svc"))
			Expect(result).To(ContainSubstring("cluster-b-system.svc"))
			Expect(result).To(ContainSubstring("monitoring-system.svc"))
		})
	})

	Context("templatePorts", func() {
		It("should template webhook service ports", func() {
			webhookService := &unstructured.Unstructured{}
			webhookService.SetAPIVersion("v1")
			webhookService.SetKind("Service")
			webhookService.SetName("test-project-webhook-service")

			content := `apiVersion: v1
kind: Service
metadata:
  name: test-project-webhook-service
  namespace: test-project-system
spec:
  ports:
  - port: 443
    targetPort: 9443
    protocol: TCP
  selector:
    control-plane: controller-manager`

			result := templater.templatePorts(content, webhookService)

			// Should template webhook port
			Expect(result).To(ContainSubstring("targetPort: {{ .Values.webhook.port }}"))
			Expect(result).NotTo(ContainSubstring("targetPort: 9443"))
		})

		It("should template metrics service ports", func() {
			metricsService := &unstructured.Unstructured{}
			metricsService.SetAPIVersion("v1")
			metricsService.SetKind("Service")
			metricsService.SetName("test-project-controller-manager-metrics-service")

			content := `apiVersion: v1
kind: Service
metadata:
  name: test-project-controller-manager-metrics-service
  namespace: test-project-system
spec:
  ports:
  - port: 8443
    targetPort: 8443
    protocol: TCP
    name: https
  selector:
    control-plane: controller-manager`

			result := templater.templatePorts(content, metricsService)

			// Should template metrics port
			Expect(result).To(ContainSubstring("port: {{ .Values.metrics.port }}"))
			Expect(result).To(ContainSubstring("targetPort: {{ .Values.metrics.port }}"))
			Expect(result).NotTo(ContainSubstring("port: 8443"))
			Expect(result).NotTo(ContainSubstring("targetPort: 8443"))
		})

		It("should template webhook container ports in Deployment", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        ports:
        - containerPort: 9443
          name: webhook-server
          protocol: TCP`

			result := templater.templatePorts(content, deployment)

			// Should template webhook containerPort
			Expect(result).To(ContainSubstring("containerPort: {{ .Values.webhook.port }}"))
			Expect(result).NotTo(ContainSubstring("containerPort: 9443"))
		})

		It("should template health probe ports in Deployment", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081`

			result := templater.templatePorts(content, deployment)

			Expect(result).To(ContainSubstring("port: 8081"))
			Expect(result).NotTo(ContainSubstring("{{ .Values"))
		})

		It("should template port-related args in Deployment", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --metrics-bind-address=:8443
        - --health-probe-bind-address=:8081
        - --leader-elect`

			result := templater.templatePorts(content, deployment)

			Expect(result).To(ContainSubstring("--metrics-bind-address=:{{ .Values.metrics.port }}"))
			Expect(result).NotTo(ContainSubstring("--metrics-bind-address=:8443"))
			Expect(result).To(ContainSubstring("--health-probe-bind-address=:8081"))
			Expect(result).To(ContainSubstring("--leader-elect"))
		})

		It("should template custom port values", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --metrics-bind-address=:9090
        - --health-probe-bind-address=:9091
        - --webhook-port=9444
        ports:
        - containerPort: 9444
          name: webhook-server
        livenessProbe:
          httpGet:
            port: 9091`

			result := templater.templatePorts(content, deployment)

			Expect(result).To(ContainSubstring("--metrics-bind-address=:{{ .Values.metrics.port }}"))
			Expect(result).To(ContainSubstring("--webhook-port={{ .Values.webhook.port }}"))
			Expect(result).To(ContainSubstring("containerPort: {{ .Values.webhook.port }}"))
			Expect(result).To(ContainSubstring("--health-probe-bind-address=:9091"))
			Expect(result).To(ContainSubstring("port: 9091"))
		})

		It("should not template non-webhook/metrics resources", func() {
			regularService := &unstructured.Unstructured{}
			regularService.SetAPIVersion("v1")
			regularService.SetKind("Service")
			regularService.SetName("test-project-some-other-service")

			content := `apiVersion: v1
kind: Service
metadata:
  name: test-project-some-other-service
spec:
  ports:
  - port: 8080
    targetPort: 8080`

			result := templater.templatePorts(content, regularService)

			// Should not template regular service ports
			Expect(result).To(ContainSubstring("port: 8080"))
			Expect(result).To(ContainSubstring("targetPort: 8080"))
			Expect(result).NotTo(ContainSubstring("{{ .Values"))
		})
	})

	Context("cert-manager resource name templating", func() {
		It("should template Certificate resource name with chart.fullname", func() {
			cert := &unstructured.Unstructured{}
			cert.SetAPIVersion("cert-manager.io/v1")
			cert.SetKind("Certificate")
			cert.SetName("test-project-serving-cert")

			content := `apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-project-serving-cert
  namespace: test-project-system`

			result := templater.ApplyHelmSubstitutions(content, cert)

			expectedCert := `name: {{ include "test-project.resourceName" (dict "suffix" "serving-cert" "context" $) }}`
			Expect(result).To(ContainSubstring(expectedCert))
			Expect(result).NotTo(ContainSubstring("name: test-project-serving-cert"))
		})

		It("should template Issuer resource name with chart.fullname", func() {
			issuer := &unstructured.Unstructured{}
			issuer.SetAPIVersion("cert-manager.io/v1")
			issuer.SetKind("Issuer")
			issuer.SetName("test-project-selfsigned-issuer")

			content := `apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: test-project-selfsigned-issuer
  namespace: test-project-system
spec:
  selfSigned: {}`

			result := templater.ApplyHelmSubstitutions(content, issuer)

			Expect(result).To(ContainSubstring(expectedIssuerName))
			Expect(result).NotTo(ContainSubstring("name: test-project-selfsigned-issuer"))
		})

		It("should template issuer reference in certificates with chart.fullname", func() {
			cert := &unstructured.Unstructured{}
			cert.SetAPIVersion("cert-manager.io/v1")
			cert.SetKind("Certificate")
			cert.SetName("test-project-serving-cert")

			content := `apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-project-serving-cert
spec:
  issuerRef:
    kind: Issuer
    name: test-project-selfsigned-issuer`

			result := templater.ApplyHelmSubstitutions(content, cert)

			Expect(result).To(ContainSubstring(expectedIssuerName))
			Expect(result).NotTo(ContainSubstring("name: test-project-selfsigned-issuer"))
		})

		It("should template all resource types generically", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  template:
    spec:
      serviceAccountName: test-project-controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// All name fields should use test-project.resourceName
			expectedName := `name: {{ include "test-project.resourceName" (dict "suffix" "controller-manager" "context" $) }}`
			expectedSA := `serviceAccountName: {{ include "test-project.resourceName" ` +
				`(dict "suffix" "controller-manager" "context" $) }}`
			Expect(result).To(ContainSubstring(expectedName))
			Expect(result).To(ContainSubstring(expectedSA))
			Expect(result).NotTo(ContainSubstring("name: test-project-controller-manager"))
		})

		It("should handle custom kustomize prefix", func() {
			customPrefixTemplater := &HelmTemplater{
				detectedPrefix:   "ln",           // Custom short prefix from kustomize
				chartName:        "test-project", // Chart/project name
				managerNamespace: "ln-system",    // Manager namespace
			}

			issuer := &unstructured.Unstructured{}
			issuer.SetAPIVersion("cert-manager.io/v1")
			issuer.SetKind("Issuer")
			issuer.SetName("ln-selfsigned-issuer")

			content := `apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: ln-selfsigned-issuer
  labels:
    app.kubernetes.io/name: ln`

			result := customPrefixTemplater.ApplyHelmSubstitutions(content, issuer)

			// Resource name uses test-project.resourceName
			Expect(result).To(ContainSubstring(expectedIssuerName))
			Expect(result).NotTo(ContainSubstring("name: ln-selfsigned-issuer"))
			// Label uses test-project.name
			Expect(result).To(ContainSubstring("app.kubernetes.io/name: {{ include \"test-project.name\" . }}"))
			Expect(result).NotTo(ContainSubstring("app.kubernetes.io/name: ln"))
		})

		It("should template RoleBinding roleRef and subjects", func() {
			rb := &unstructured.Unstructured{}
			rb.SetAPIVersion("rbac.authorization.k8s.io/v1")
			rb.SetKind("RoleBinding")
			rb.SetName("test-project-leader-election-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-leader-election-rolebinding
roleRef:
  name: test-project-leader-election-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager`

			result := templater.ApplyHelmSubstitutions(content, rb)

			// All references should use test-project.resourceName
			expectedRB := `name: {{ include "test-project.resourceName" ` +
				`(dict "suffix" "leader-election-rolebinding" "context" $) }}`
			expectedRole := `name: {{ include "test-project.resourceName" ` +
				`(dict "suffix" "leader-election-role" "context" $) }}`
			expectedSA := `name: {{ include "test-project.resourceName" ` +
				`(dict "suffix" "controller-manager" "context" $) }}`
			Expect(result).To(ContainSubstring(expectedRB))
			Expect(result).To(ContainSubstring(expectedRole))
			Expect(result).To(ContainSubstring(expectedSA))
		})
	})

	Context("custom container name support", func() {
		It("should template deployment fields when container name is not 'manager'", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Deployment with custom container name "osiris-manager" using default-container annotation
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: osiris-manager
    spec:
      containers:
      - name: osiris-manager
        image: docker.io/server/osiris:1.0.5
        imagePullPolicy: Always
        env:
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        args:
        - --leader-elect
        - --health-probe-bind-address=:8081
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
        volumeMounts: []
      serviceAccountName: controller-manager
      volumes: []`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should template image reference (not hardcoded)
			Expect(result).To(ContainSubstring(
				`image: "{{ .Values.manager.image.repository }}:{{ .Values.manager.image.tag }}"`))
			Expect(result).NotTo(ContainSubstring("image: docker.io/server/osiris:1.0.5"))

			// Should template imagePullPolicy
			Expect(result).To(ContainSubstring("imagePullPolicy: {{ .Values.manager.image.pullPolicy }}"))
			Expect(result).NotTo(ContainSubstring("imagePullPolicy: Always"))

			// Should template resources
			Expect(result).To(ContainSubstring("{{- if .Values.manager.resources }}"))
			Expect(result).To(ContainSubstring("{{- toYaml .Values.manager.resources | nindent"))

			// Should template environment variables
			Expect(result).To(ContainSubstring("{{- if .Values.manager.env }}"))
			Expect(result).To(ContainSubstring("{{- toYaml .Values.manager.env | nindent"))

			// Should template args
			Expect(result).To(ContainSubstring("{{- range .Values.manager.args }}"))

			// Container name should remain "osiris-manager"
			Expect(result).To(ContainSubstring("name: osiris-manager"))
		})

		It("should fall back to 'manager' when default-container annotation is missing", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Deployment without default-container annotation (backward compatibility test)
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
        resources:
          limits:
            cpu: 500m
            memory: 128Mi`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should still template fields for "manager" container
			Expect(result).To(ContainSubstring(
				`image: "{{ .Values.manager.image.repository }}:{{ .Values.manager.image.tag }}"`))
			Expect(result).To(ContainSubstring("{{- if .Values.manager.resources }}"))
		})

		It("should not template when container name doesn't match annotation", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Deployment with mismatched annotation and container name
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: main-container
    spec:
      containers:
      - name: sidecar
        image: sidecar:latest
        resources:
          limits:
            cpu: 100m`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should NOT template sidecar container (doesn't match annotation)
			Expect(result).To(ContainSubstring("image: sidecar:latest"))
			Expect(result).NotTo(ContainSubstring("{{ .Values.manager.image.repository }}"))
		})
	})
})
