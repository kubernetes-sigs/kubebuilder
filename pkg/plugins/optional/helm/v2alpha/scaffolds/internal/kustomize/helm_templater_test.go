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
			detectedPrefix: "test-project",
			chartName:      "test-project",
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

	Context("edge cases", func() {
		It("should handle empty content", func() {
			testResource := &unstructured.Unstructured{}
			testResource.SetAPIVersion("v1")
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
			testResource.SetAPIVersion("v1")
			testResource.SetKind("ConfigMap")

			malformedContent := "not: valid: yaml: content:"
			result := templater.ApplyHelmSubstitutions(malformedContent, testResource)

			// Should return content as-is for malformed YAML
			Expect(result).To(Equal(malformedContent))
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
				detectedPrefix: "ln",           // Custom short prefix from kustomize
				chartName:      "test-project", // Chart/project name
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
})
