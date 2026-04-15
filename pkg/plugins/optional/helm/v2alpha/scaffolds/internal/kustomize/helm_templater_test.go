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
	"regexp"
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
			roleNamespaces:   nil,
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

		It("should template deployment spec.replicas from .Values.manager.replicas", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
`
			result := templater.ApplyHelmSubstitutions(content, deploymentResource)
			Expect(result).To(ContainSubstring("replicas: {{ .Values.manager.replicas }}"))
			Expect(result).NotTo(ContainSubstring("replicas: 1"))
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
			Expect(result).To(ContainSubstring("{{- if not .Values.metrics.secure }}"))
			Expect(result).To(ContainSubstring("- --metrics-secure=false"))
			Expect(result).To(ContainSubstring("- --metrics-bind-address=0"))
			Expect(result).To(ContainSubstring("- --health-probe-bind-address=:8081"))
			Expect(result).To(ContainSubstring("{{- range .Values.manager.args }}"))
			Expect(result).NotTo(ContainSubstring("BUSYBOX_IMAGE"))
			Expect(result).NotTo(ContainSubstring("MEMCACHED_IMAGE"))
			Expect(result).To(ContainSubstring("image: " +
				"\"{{ .Values.manager.image.repository }}:{{ .Values.manager.image.tag | default .Chart.AppVersion }}\""))
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

		// Inside a '{{- with .Values.manager.imagePullSecrets }}' block, '.' is bound to the
		// imagePullSecrets slice, so using '.Values.manager.imagePullSecrets' inside the block
		// causes a Helm render error: "can't evaluate field Values in type []interface {}".
		// Insure usage of '.' for toYaml instead of the full path.
		It("should use '.' not '.Values.manager.imagePullSecrets' inside with block for imagePullSecrets", func() {
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
      - name: myregistrykey
      containers:
      - args:
        - --leader-elect`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			Expect(result).To(ContainSubstring("{{- with .Values.manager.imagePullSecrets }}"))
			// Must use '.' inside the with block, NOT '.Values.manager.imagePullSecrets'
			Expect(result).To(ContainSubstring("{{- toYaml . | nindent"))
			Expect(result).NotTo(ContainSubstring("{{- toYaml .Values.manager.imagePullSecrets | nindent"))
		})

		It("should use '.' not '.Values.manager.imagePullSecrets' inside with block when field is injected", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			// No imagePullSecrets in the content - the function will inject the block
			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      serviceAccountName: test-project-controller-manager
      containers:
      - args:
        - --leader-elect`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			Expect(result).To(ContainSubstring("{{- with .Values.manager.imagePullSecrets }}"))
			// Must use '.' inside the with block, NOT '.Values.manager.imagePullSecrets'
			Expect(result).To(ContainSubstring("{{- toYaml . | nindent"))
			Expect(result).NotTo(ContainSubstring("{{- toYaml .Values.manager.imagePullSecrets | nindent"))
		})

		It("should template deployment strategy", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      containers:
      - name: manager`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			Expect(result).To(ContainSubstring("{{- with .Values.manager.strategy }}"))
			Expect(result).To(ContainSubstring("strategy: {{ toYaml . | nindent"))
			Expect(result).NotTo(ContainSubstring("type: RollingUpdate"))
		})

		It("should template priorityClassName", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      priorityClassName: high-priority
      containers:
      - name: manager`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			Expect(result).To(ContainSubstring("{{- with .Values.manager.priorityClassName }}"))
			Expect(result).To(ContainSubstring("priorityClassName: {{ . | quote }}"))
			Expect(result).NotTo(ContainSubstring("priorityClassName: high-priority"))
		})

		It("should template topologySpreadConstraints", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: kubernetes.io/hostname
        whenUnsatisfiable: DoNotSchedule
      containers:
      - name: manager`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			Expect(result).To(ContainSubstring("{{- with .Values.manager.topologySpreadConstraints }}"))
			Expect(result).To(ContainSubstring("topologySpreadConstraints: {{ toYaml . | nindent"))
			Expect(result).NotTo(ContainSubstring("maxSkew: 1"))
		})

		It("should template terminationGracePeriodSeconds with nil check", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      serviceAccountName: controller-manager
      terminationGracePeriodSeconds: 10
      containers:
      - name: manager`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			Expect(result).To(ContainSubstring(
				"{{- if and (hasKey .Values.manager \"terminationGracePeriodSeconds\")"))
			Expect(result).To(ContainSubstring("(ne .Values.manager.terminationGracePeriodSeconds nil)"))
			Expect(result).To(ContainSubstring(
				"terminationGracePeriodSeconds: {{ .Values.manager.terminationGracePeriodSeconds }}"))
			Expect(result).NotTo(ContainSubstring("terminationGracePeriodSeconds: 10"))
		})

		It("should template terminationGracePeriodSeconds zero value", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      serviceAccountName: controller-manager
      terminationGracePeriodSeconds: 0
      containers:
      - name: manager`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			// Should use hasKey to support 0 values
			Expect(result).To(ContainSubstring("hasKey .Values.manager \"terminationGracePeriodSeconds\""))
			Expect(result).To(ContainSubstring(
				"terminationGracePeriodSeconds: {{ .Values.manager.terminationGracePeriodSeconds }}"))
			Expect(result).NotTo(ContainSubstring("terminationGracePeriodSeconds: 0"))
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

		It("should template ServiceMonitor port and scheme based on metrics.secure", func() {
			serviceMonitorResource := &unstructured.Unstructured{}
			serviceMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			serviceMonitorResource.SetKind("ServiceMonitor")
			serviceMonitorResource.SetName("test-project-controller-manager-metrics-monitor")

			content := `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-project-controller-manager-metrics-monitor
spec:
  endpoints:
  - port: https
    scheme: https`

			result := templater.ApplyHelmSubstitutions(content, serviceMonitorResource)

			// Port should be templated to conditional
			Expect(result).To(ContainSubstring("port: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}"))
			Expect(result).NotTo(ContainSubstring("port: https"))

			// Scheme should be templated to conditional
			Expect(result).To(ContainSubstring("scheme: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}"))
			Expect(result).NotTo(ContainSubstring("scheme: https"))
		})

		It("should wrap ServiceMonitor bearerTokenFile with metrics.secure conditional", func() {
			serviceMonitorResource := &unstructured.Unstructured{}
			serviceMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			serviceMonitorResource.SetKind("ServiceMonitor")
			serviceMonitorResource.SetName("test-project-controller-manager-metrics-monitor")

			content := `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-project-controller-manager-metrics-monitor
spec:
  endpoints:
  - bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    port: https`

			result := templater.ApplyHelmSubstitutions(content, serviceMonitorResource)

			// BearerTokenFile should be wrapped with conditional
			Expect(result).To(ContainSubstring("{{- if .Values.metrics.secure }}"))
			Expect(result).To(ContainSubstring("bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should wrap ServiceMonitor tlsConfig with metrics.secure conditional", func() {
			serviceMonitorResource := &unstructured.Unstructured{}
			serviceMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			serviceMonitorResource.SetKind("ServiceMonitor")
			serviceMonitorResource.SetName("test-project-controller-manager-metrics-monitor")

			content := `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-project-controller-manager-metrics-monitor
spec:
  endpoints:
  - port: https
    scheme: https
    tlsConfig:
      insecureSkipVerify: true`

			result := templater.ApplyHelmSubstitutions(content, serviceMonitorResource)

			// TLS config should be wrapped with conditional
			Expect(result).To(ContainSubstring("{{- if .Values.metrics.secure }}"))
			Expect(result).To(ContainSubstring("tlsConfig:"))
			Expect(result).To(ContainSubstring("insecureSkipVerify: true"))
			Expect(result).To(ContainSubstring("{{- end }}"))

			// Should have port and scheme templated too
			Expect(result).To(ContainSubstring("port: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}"))
			Expect(result).To(ContainSubstring("scheme: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}"))
		})

		It("should wrap ServiceMonitor with certManager.enable conditional when using default cert-manager secret", func() {
			serviceMonitorResource := &unstructured.Unstructured{}
			serviceMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			serviceMonitorResource.SetKind("ServiceMonitor")
			serviceMonitorResource.SetName("test-project-controller-manager-metrics-monitor")

			content := `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-project-controller-manager-metrics-monitor
spec:
  endpoints:
  - port: https
    scheme: https
    bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    tlsConfig:
      serverName: service.namespace.svc
      insecureSkipVerify: false
      ca:
        secret:
          name: metrics-server-cert
          key: ca.crt
      cert:
        secret:
          name: metrics-server-cert
          key: tls.crt
      keySecret:
        name: metrics-server-cert
        key: tls.key`

			result := templater.ApplyHelmSubstitutions(content, serviceMonitorResource)

			// Should have cert-manager conditional (using default cert-manager secret)
			Expect(result).To(ContainSubstring("{{- if .Values.certManager.enable }}"))

			// Should preserve secret names
			Expect(result).To(ContainSubstring("name: metrics-server-cert"))

			// Should have else branch with insecureSkipVerify
			Expect(result).To(ContainSubstring("{{- else }}"))
			Expect(result).To(ContainSubstring("insecureSkipVerify: true"))
		})

		It("should preserve custom cert secrets without cert-manager conditional", func() {
			serviceMonitorResource := &unstructured.Unstructured{}
			serviceMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			serviceMonitorResource.SetKind("ServiceMonitor")
			serviceMonitorResource.SetName("test-project-controller-manager-metrics-monitor")

			// Custom certs from Vault/manual secrets - NOT cert-manager
			content := `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-project-controller-manager-metrics-monitor
spec:
  endpoints:
  - port: https
    scheme: https
    bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    tlsConfig:
      serverName: custom-service.namespace.svc
      insecureSkipVerify: false
      ca:
        secret:
          name: vault-ca-secret
          key: ca.crt
      cert:
        secret:
          name: vault-client-cert
          key: tls.crt
      keySecret:
        name: vault-client-key
        key: tls.key`

			result := templater.ApplyHelmSubstitutions(content, serviceMonitorResource)

			// Should NOT have cert-manager conditional (custom secrets)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.certManager.enable }}"))
			Expect(result).NotTo(ContainSubstring("{{- else }}"))

			// Custom secret names should be preserved as-is
			Expect(result).To(ContainSubstring("name: vault-ca-secret"))
			Expect(result).To(ContainSubstring("name: vault-client-cert"))
			Expect(result).To(ContainSubstring("name: vault-client-key"))
			Expect(result).To(ContainSubstring("insecureSkipVerify: false"))

			// Should still have metrics.secure wrapper
			Expect(result).To(ContainSubstring("{{- if .Values.metrics.secure }}"))
		})

		It("should NOT add cert-manager conditional when ServiceMonitor only has insecureSkipVerify", func() {
			serviceMonitorResource := &unstructured.Unstructured{}
			serviceMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			serviceMonitorResource.SetKind("ServiceMonitor")
			serviceMonitorResource.SetName("test-project-controller-manager-metrics-monitor")

			content := `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-project-controller-manager-metrics-monitor
spec:
  endpoints:
  - port: https
    scheme: https
    bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    tlsConfig:
      insecureSkipVerify: true`

			result := templater.ApplyHelmSubstitutions(content, serviceMonitorResource)

			// Should NOT have cert-manager conditional (no cert-manager fields detected)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.certManager.enable }}"))

			// Should preserve insecureSkipVerify: true as-is
			Expect(result).To(ContainSubstring("insecureSkipVerify: true"))

			// Should still have metrics.secure wrapper
			Expect(result).To(ContainSubstring("{{- if .Values.metrics.secure }}"))
		})

		It("should convert insecureSkipVerify: false to true when no cert-manager fields present", func() {
			serviceMonitorResource := &unstructured.Unstructured{}
			serviceMonitorResource.SetAPIVersion("monitoring.coreos.com/v1")
			serviceMonitorResource.SetKind("ServiceMonitor")
			serviceMonitorResource.SetName("test-project-controller-manager-metrics-monitor")

			// Invalid config: insecureSkipVerify: false without certs
			content := `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: test-project-controller-manager-metrics-monitor
spec:
  endpoints:
  - port: https
    scheme: https
    tlsConfig:
      insecureSkipVerify: false`

			result := templater.ApplyHelmSubstitutions(content, serviceMonitorResource)

			// Should convert to insecureSkipVerify: true (can't have false without certs)
			Expect(result).To(ContainSubstring("insecureSkipVerify: true"))
			Expect(result).NotTo(ContainSubstring("insecureSkipVerify: false"))

			// Should NOT have cert-manager conditional (no cert-manager fields)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.certManager.enable }}"))
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

			// Should be wrapped with certManager, metrics, AND metrics.secure conditionals
			// Metrics certs only needed when using HTTPS (metrics.secure=true)
			Expect(result).To(ContainSubstring(
				"{{- if and .Values.certManager.enable .Values.metrics.enable .Values.metrics.secure }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should add kind conditionals to essential ClusterRole resources", func() {
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

			// Should NOT have rbac.create conditional (always created)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.rbac.create }}"))
			// Should have kind conditional for namespace-scoped support
			Expect(result).To(ContainSubstring("{{- if .Values.rbac.namespaced }}"))
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

		It("should add crd.enable conditional and resource-policy annotation for CRDs", func() {
			crdResource := &unstructured.Unstructured{}
			crdResource.SetAPIVersion("apiextensions.k8s.io/v1")
			crdResource.SetKind("CustomResourceDefinition")
			crdResource.SetName("guestbooks.example.com")

			content := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: guestbooks.example.com
spec:
  group: example.com`

			result := templater.ApplyHelmSubstitutions(content, crdResource)

			// Should be wrapped with crd.enable conditional
			Expect(result).To(ContainSubstring("{{- if .Values.crd.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
			// Should have resource-policy annotation for helm uninstall protection
			Expect(result).To(ContainSubstring("{{- if .Values.crd.keep }}"))
			Expect(result).To(ContainSubstring(`"helm.sh/resource-policy": keep`))
			// Injected annotations should use 2-space indentation matching sigs.k8s.io/yaml output.
			// annotations: at 2-space indent, values at 4-space indent.
			expectedAnnotations := "  annotations:\n" +
				"    {{- if .Values.crd.keep }}\n" +
				"    \"helm.sh/resource-policy\": keep\n" +
				"    {{- end }}"
			Expect(result).To(ContainSubstring(expectedAnnotations))
		})

		It("should add resource-policy annotation to CRDs that already have annotations", func() {
			crdResource := &unstructured.Unstructured{}
			crdResource.SetAPIVersion("apiextensions.k8s.io/v1")
			crdResource.SetKind("CustomResourceDefinition")
			crdResource.SetName("configs.example.com")

			content := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.17.0
  name: configs.example.com
spec:
  group: example.com`

			result := templater.ApplyHelmSubstitutions(content, crdResource)

			// Should be wrapped with crd.enable conditional
			Expect(result).To(ContainSubstring("{{- if .Values.crd.enable }}"))
			// Should have resource-policy annotation
			Expect(result).To(ContainSubstring("{{- if .Values.crd.keep }}"))
			Expect(result).To(ContainSubstring(`"helm.sh/resource-policy": keep`))
			// Should preserve existing annotation
			Expect(result).To(ContainSubstring("controller-gen.kubebuilder.io/version"))
			// Injected annotation should be at same indent as existing annotations (4 spaces)
			expectedAnnotations := "  annotations:\n" +
				"    {{- if .Values.crd.keep }}\n" +
				"    \"helm.sh/resource-policy\": keep\n" +
				"    {{- end }}\n" +
				"    controller-gen.kubebuilder.io/version"
			Expect(result).To(ContainSubstring(expectedAnnotations))
		})

		It("should add manager.enabled conditional for manager Deployments", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  replicas: 1`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			// Should be wrapped with manager.enabled conditional that defaults to true when key is absent
			expectedConditional := `{{- if or (not (hasKey .Values.manager "enabled")) (.Values.manager.enabled) }}`
			Expect(result).To(ContainSubstring(expectedConditional))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should not add manager.enabled conditional for non-manager Deployments", func() {
			deploymentResource := &unstructured.Unstructured{}
			deploymentResource.SetAPIVersion("apps/v1")
			deploymentResource.SetKind("Deployment")
			deploymentResource.SetName("other-deployment")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: other-deployment
  namespace: test-project-system
spec:
  replicas: 1`

			result := templater.ApplyHelmSubstitutions(content, deploymentResource)

			// Should NOT be wrapped with manager.enabled conditional
			expectedConditional := `{{- if or (not (hasKey .Values.manager "enabled")) (.Values.manager.enabled) }}`
			Expect(result).NotTo(ContainSubstring(expectedConditional))
			Expect(result).NotTo(ContainSubstring(".Values.manager.enabled"))
		})
	})

	Context("helper RBAC wrapping", func() {
		It("should add rbac.helpers conditional for helper RBAC roles", func() {
			clusterRoleResource := &unstructured.Unstructured{}
			clusterRoleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			clusterRoleResource.SetKind("ClusterRole")
			clusterRoleResource.SetName("test-project-memcached-editor-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-memcached-editor-role`

			result := templater.ApplyHelmSubstitutions(content, clusterRoleResource)

			// Should be wrapped with rbac.helpers conditional
			Expect(result).To(ContainSubstring("{{- if .Values.rbac.helpers.enable }}"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should add rbac.helpers conditional for helper ClusterRoleBindings", func() {
			bindingResource := &unstructured.Unstructured{}
			bindingResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			bindingResource.SetKind("ClusterRoleBinding")
			bindingResource.SetName("test-project-memcached-viewer-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: test-project-memcached-viewer-rolebinding`

			result := templater.ApplyHelmSubstitutions(content, bindingResource)

			// Should be wrapped with rbac.helpers conditional
			Expect(result).To(ContainSubstring("{{- if .Values.rbac.helpers.enable }}"))
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

			// Should template with test-project.serviceAccountName helper
			expected := `name: {{ include "test-project.serviceAccountName" . }}`
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

		It("should template secretRef name inside envFrom", func() {
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
      containers:
      - name: manager
        envFrom:
        - secretRef:
            name: test-project-manager-secrets`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// secretRef.name inside envFrom must be templated, not left as a hardcoded string
			expectedSecretRef := `name: {{ include "test-project.resourceName" ` +
				`(dict "suffix" "manager-secrets" "context" $) }}`
			Expect(result).To(ContainSubstring(expectedSecretRef))
			Expect(result).NotTo(ContainSubstring("name: test-project-manager-secrets"))
		})

		It("should template configMapRef name inside envFrom", func() {
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
      containers:
      - name: manager
        envFrom:
        - configMapRef:
            name: test-project-manager-config`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// configMapRef.name inside envFrom must be templated
			expectedConfigMapRef := `name: {{ include "test-project.resourceName" ` +
				`(dict "suffix" "manager-config" "context" $) }}`
			Expect(result).To(ContainSubstring(expectedConfigMapRef))
			Expect(result).NotTo(ContainSubstring("name: test-project-manager-config"))
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

		It("should not double-escape quotes already escaped by yaml.Marshal in double-quoted YAML scalars", func() {
			// Regression test: yaml.Marshal represents literal " inside a {{ }} expression as \"
			// in a double-quoted YAML scalar, so a second pass escaping " to \" produced \\" which
			// broke Helm's template parser by closing the string literal early (U+002D '-' error).
			crdResource := &unstructured.Unstructured{}
			crdResource.SetAPIVersion("apiextensions.k8s.io/v1")
			crdResource.SetKind("CustomResourceDefinition")
			crdResource.SetName("webrequestcommitstatuses.promoter.argoproj.io")

			// Simulate the raw YAML text that yaml.Marshal produces for a double-quoted scalar
			// containing a " character where the inner " appears as \" in the YAML text.
			content := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  versions:
  - schema:
      openAPIV3Schema:
        properties:
          spec:
            description: "example: {{ index .NamespaceMetadata.Labels \"asset-id\" }}"`

			result := templater.ApplyHelmSubstitutions(content, crdResource)

			// The escaped form must be valid Go template syntax: \" (single backslash+quote),
			// NOT \\" (double backslash+quote) which would terminate the string literal early.
			Expect(result).To(ContainSubstring(`{{ "{{ index .NamespaceMetadata.Labels \"asset-id\" }}" }}`),
				"pre-escaped YAML quotes must not be double-escaped to \\\\ which breaks Helm template parsing")
			Expect(result).NotTo(ContainSubstring(`\\"asset-id\\"`),
				"double-escaped quotes (\\\\\") must not appear in the output")
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

		It("should add conditional namespace for ClusterRole when rendering as Role", func() {
			clusterRoleResource := &unstructured.Unstructured{}
			clusterRoleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			clusterRoleResource.SetKind("ClusterRole")
			clusterRoleResource.SetName("test-project-manager-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-manager-role`

			result := templater.ApplyHelmSubstitutions(content, clusterRoleResource)

			// Should add conditional namespace for Role variant
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
			// Namespace should be conditional (only when rbac.namespaced is true)
			Expect(result).To(ContainSubstring("{{- if .Values.rbac.namespaced }}"))
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
				roleNamespaces:   nil,
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
				roleNamespaces:   nil,
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
				roleNamespaces:   nil,
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

			// Deployment name uses test-project.resourceName
			expectedName := `name: {{ include "test-project.resourceName" (dict "suffix" "controller-manager" "context" $) }}`
			// ServiceAccount reference uses test-project.serviceAccountName
			expectedSA := `serviceAccountName: {{ include "test-project.serviceAccountName" . }}`
			Expect(result).To(ContainSubstring(expectedName))
			Expect(result).To(ContainSubstring(expectedSA))
			Expect(result).NotTo(ContainSubstring("name: test-project-controller-manager"))
		})

		It("should handle custom kustomize prefix", func() {
			customPrefixTemplater := &HelmTemplater{
				detectedPrefix:   "ln",           // Custom short prefix from kustomize
				chartName:        "test-project", // Chart/project name
				managerNamespace: "ln-system",    // Manager namespace
				roleNamespaces:   nil,
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

			// RoleBinding and Role use test-project.resourceName
			expectedRB := `name: {{ include "test-project.resourceName" ` +
				`(dict "suffix" "leader-election-rolebinding" "context" $) }}`
			expectedRole := `name: {{ include "test-project.resourceName" ` +
				`(dict "suffix" "leader-election-role" "context" $) }}`
			// ServiceAccount subject uses test-project.serviceAccountName
			expectedSA := `name: {{ include "test-project.serviceAccountName" . }}`
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

			// Deployment with custom container name "controller-test" using default-container annotation
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  namespace: test-project-system
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: controller-test
    spec:
      containers:
      - name: controller-test
        image: controller:latest
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
				`image: "{{ .Values.manager.image.repository }}:{{ .Values.manager.image.tag | default .Chart.AppVersion }}"`))
			Expect(result).NotTo(ContainSubstring("image: controller:latest"))

			// Should template imagePullPolicy
			Expect(result).To(ContainSubstring("imagePullPolicy: {{ .Values.manager.image.pullPolicy }}"))
			Expect(result).NotTo(ContainSubstring("imagePullPolicy: Always"))

			// Should template resources
			Expect(result).To(ContainSubstring("{{- if .Values.manager.resources }}"))
			Expect(result).To(ContainSubstring("{{- toYaml .Values.manager.resources | nindent"))

			// Env list + envOverrides (--set). Secret refs go in env list.
			Expect(result).To(ContainSubstring(".Values.manager.env"))
			Expect(result).To(ContainSubstring("toYaml .Values.manager.env"))
			Expect(result).To(ContainSubstring("envOverrides"))

			// Should template args
			Expect(result).To(ContainSubstring("{{- range .Values.manager.args }}"))

			// Container name should remain "controller-test"
			Expect(result).To(ContainSubstring("name: controller-test"))
		})

		It("should append extraVolumes and extraVolumeMounts when lists are present", func() {
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
      volumes:
      - name: webhook-certs
        secret:
          secretName: webhook-server-cert
      containers:
      - name: manager
        image: controller:latest
        volumeMounts:
        - name: webhook-certs
          mountPath: /tmp/k8s-webhook-server/serving-certs
          readOnly: true
`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			Expect(result).To(ContainSubstring(".Values.manager.extraVolumes"))
			Expect(result).To(ContainSubstring(".Values.manager.extraVolumeMounts"))
		})

		It("should inject extraVolumes/extraVolumeMounts when Kustomize has volumeMounts: [] and volumes: []", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Default Kustomize output: single-line empty lists (no webhook/metrics patches)
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
        volumeMounts: []
      volumes: []`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			Expect(result).To(ContainSubstring(".Values.manager.extraVolumes"))
			Expect(result).To(ContainSubstring(".Values.manager.extraVolumeMounts"))
		})

		It("should append only extraVolumes from values and keep webhook/metrics conditional", func() {
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
      volumes:
      - name: webhook-certs
        secret:
          secretName: webhook-server-cert
      - name: metrics-certs
        secret:
          secretName: metrics-server-cert
      - name: app-secret-1
        secret:
          secretName: app-secret-1
      containers:
      - name: manager
        image: controller:latest
        volumeMounts:
        - name: webhook-certs
          mountPath: /tmp/k8s-webhook-server/serving-certs
          readOnly: true
        - name: metrics-certs
          mountPath: /tmp/k8s-metrics-server/metrics-certs
          readOnly: true
        - name: app-secret-1
          mountPath: /etc/secrets
          readOnly: true
`
			result := templater.ApplyHelmSubstitutions(content, deployment)
			Expect(result).To(ContainSubstring(".Values.certManager.enable"))
			Expect(result).To(ContainSubstring(".Values.manager.extraVolumes"))
			Expect(result).To(ContainSubstring("app-secret-1"))
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
				`image: "{{ .Values.manager.image.repository }}:{{ .Values.manager.image.tag | default .Chart.AppVersion }}"`))
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

	// templateBasicWithStatement must correctly consume entire YAML sequence blocks
	// (like tolerations) because their list items start at the same indentation as
	// the parent key, distinguished only by the leading "- " marker.
	Context("scheduling fields templating (nodeSelector / affinity / tolerations)", func() {
		It("should replace an existing multi-item tolerations block with a single Helm stanza", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
        operator: Exists
      - effect: NoExecute
        key: another-key
        value: somevalue
      securityContext:
        runAsNonRoot: true
      serviceAccountName: test-project-controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Exactly one Helm-templated tolerations stanza must be present.
			Expect(strings.Count(result, "tolerations:")).To(Equal(1),
				"Expected exactly one tolerations: line in output")
			Expect(result).To(ContainSubstring("{{- with .Values.manager.tolerations }}"))
			Expect(result).To(ContainSubstring("tolerations: {{ toYaml . | nindent"))
			Expect(result).To(ContainSubstring("{{- end }}"))

			// The raw list items must be gone.
			Expect(result).NotTo(ContainSubstring("- effect: NoSchedule"))
			Expect(result).NotTo(ContainSubstring("- effect: NoExecute"))
			Expect(result).NotTo(ContainSubstring("key: node-role.kubernetes.io/control-plane"))
			Expect(result).NotTo(ContainSubstring("key: another-key"))

			// Fields after tolerations must still be present (templated).
			Expect(result).To(ContainSubstring("securityContext:"))
			Expect(result).To(ContainSubstring(".Values.manager.podSecurityContext"))
		})

		It("should insert a tolerations Helm stanza when the field is absent", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
      securityContext:
        runAsNonRoot: true`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			Expect(result).To(ContainSubstring("{{- with .Values.manager.tolerations }}"))
			Expect(result).To(ContainSubstring("tolerations: {{ toYaml . | nindent"))
			Expect(result).To(ContainSubstring("{{- end }}"))
		})

		It("should be idempotent: running templating twice does not duplicate stanzas", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
        operator: Exists`

			first := templater.ApplyHelmSubstitutions(content, deployment)
			second := templater.ApplyHelmSubstitutions(first, deployment)

			Expect(strings.Count(second, "{{- with .Values.manager.tolerations }}")).To(Equal(1),
				"Expected exactly one {{- with .Values.manager.tolerations }} after two passes")
			Expect(strings.Count(second, "tolerations:")).To(Equal(1),
				"Expected exactly one tolerations: line after two passes")
		})

		It("should correctly template nodeSelector (map type) without regression", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
      nodeSelector:
        kubernetes.io/os: linux
        custom-label: my-node`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			Expect(strings.Count(result, "nodeSelector:")).To(Equal(1))
			Expect(result).To(ContainSubstring("{{- with .Values.manager.nodeSelector }}"))
			Expect(result).To(ContainSubstring("nodeSelector: {{ toYaml . | nindent"))
			// Raw entries must be removed.
			Expect(result).NotTo(ContainSubstring("kubernetes.io/os: linux"))
			Expect(result).NotTo(ContainSubstring("custom-label: my-node"))
		})

		It("should correctly template affinity (map type) without regression", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/arch
                operator: In
                values:
                - amd64`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			Expect(strings.Count(result, "affinity:")).To(Equal(1))
			Expect(result).To(ContainSubstring("{{- with .Values.manager.affinity }}"))
			Expect(result).To(ContainSubstring("affinity: {{ toYaml . | nindent"))
			// Raw sub-fields must be removed.
			Expect(result).NotTo(ContainSubstring("nodeAffinity:"))
			Expect(result).NotTo(ContainSubstring("requiredDuringSchedulingIgnoredDuringExecution:"))
		})

		It("should not match a key that only shares a prefix with the target field name", func() {
			// Regression guard: key matching must require the trailing colon so that
			// e.g. "nodeSelector" does not accidentally match "nodeSelectorTerms:".
			// This YAML tests that nodeSelectorTerms inside an affinity block is not
			// treated as the nodeSelector field itself.
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/arch
                operator: In`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// The nodeSelector Helm stanza must be inserted (field was absent).
			Expect(result).To(ContainSubstring("{{- with .Values.manager.nodeSelector }}"))
			// The affinity block must only appear once and be Helm-templated.
			Expect(strings.Count(result, "affinity:")).To(Equal(1))
			Expect(result).To(ContainSubstring("{{- with .Values.manager.affinity }}"))
			// nodeSelectorTerms is part of the affinity value; it must be gone since
			// the whole affinity block is replaced by the Helm stanza.
			Expect(result).NotTo(ContainSubstring("nodeSelectorTerms:"))
		})
	})

	Context("custom labels and annotations", func() {
		It("should add custom labels and annotations to manager Deployment only", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: {{ include "test-project.name" . }}
    helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
    app.kubernetes.io/instance: {{ .Release.Name }}
    control-plane: controller-manager
  name: {{ include "test-project.resourceName" (dict "suffix" "controller-manager" "context" $) }}
  namespace: {{ .Release.Namespace }}
spec:
  replicas: {{ .Values.manager.replicas }}
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        app.kubernetes.io/name: {{ include "test-project.name" . }}
        helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
        app.kubernetes.io/instance: {{ .Release.Name }}
        app.kubernetes.io/managed-by: {{ .Release.Service }}
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should add manager.labels to Deployment (with automatic filtering of existing keys)
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.labels }}"),
				ContainSubstring("{{- if .Values.manager.labels }}"),
			))

			// Should add manager.pod.labels to Pod template (with automatic filtering)
			Expect(result).To(ContainSubstring("{{- with .Values.manager.pod }}"))
			Expect(result).To(ContainSubstring("{{- with .labels }}"))

			// Should add manager.annotations to Deployment
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.annotations }}"),
				ContainSubstring("{{- if .Values.manager.annotations }}"),
			))
			// Verify annotations are properly nested under metadata: (not top-level)
			Expect(regexp.MustCompile(`(?m)^metadata:\n(?:^[ ]{2}.*\n)*^[ ]{2}annotations:\n`).MatchString(result)).To(BeTrue(),
				"manager.annotations should be nested under Deployment metadata")
			Expect(regexp.MustCompile(`(?ms)^annotations:\n.*?^spec:`).MatchString(result)).To(BeFalse(),
				"annotations should not be emitted as a top-level block before spec")

			// Should add manager.pod.annotations to Pod template (with automatic filtering)
			Expect(result).To(ContainSubstring("{{- with .annotations }}"))

			// Each field should appear exactly once (check for either pattern: with/without omit)
			labelsCount := strings.Count(result, ".Values.manager.labels")
			Expect(labelsCount).To(BeNumerically(">=", 1), "Should have labels block in Deployment")

			podLabelsCount := strings.Count(result, ".labels")
			Expect(podLabelsCount).To(BeNumerically(">=", 1), "Should have pod.labels block in Pod template")

			annotationsCount := strings.Count(result, ".Values.manager.annotations")
			Expect(annotationsCount).To(BeNumerically(">=", 1), "Should have annotations block in Deployment")

			podAnnotationsCount := strings.Count(result, ".annotations")
			Expect(podAnnotationsCount).To(BeNumerically(">=", 1), "Should have pod.annotations block in Pod template")
		})

		It("should not add custom labels/annotations to non-manager Deployment", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("other-deployment")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: other
  name: other-deployment
spec:
  template:
    metadata:
      labels:
        app: other`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should not add any manager custom labels/annotations
			Expect(result).NotTo(ContainSubstring("{{- if .Values.manager.labels }}"))
			Expect(result).NotTo(ContainSubstring("{{- if .Values.manager.annotations }}"))
			Expect(result).NotTo(ContainSubstring("{{- with .Values.manager.pod }}"))
			Expect(result).NotTo(ContainSubstring("{{- with .labels }}"))
			Expect(result).NotTo(ContainSubstring("{{- with .annotations }}"))
		})

		It("should not duplicate labels/annotations blocks when applied twice", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    control-plane: controller-manager
  name: test-project-controller-manager
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager`

			// Apply templating twice
			result1 := templater.ApplyHelmSubstitutions(content, deployment)
			result2 := templater.ApplyHelmSubstitutions(result1, deployment)

			// Each field should appear exactly once (idempotency check)
			// Count manager.labels references (either with if or with omit)
			labelsCount := strings.Count(result2, ".Values.manager.labels")
			Expect(labelsCount).To(BeNumerically("<=", 2), "Should not duplicate labels blocks excessively")

			// Count .labels references in pod context
			podLabelsPattern := regexp.MustCompile(`{{- with (?:omit )?\.labels`)
			podLabelsMatches := podLabelsPattern.FindAllString(result2, -1)
			Expect(len(podLabelsMatches)).To(BeNumerically("<=", 2), "Should not duplicate pod.labels blocks excessively")

			// Count manager.annotations references
			annotationsCount := strings.Count(result2, ".Values.manager.annotations")
			Expect(annotationsCount).To(BeNumerically("<=", 2), "Should not duplicate annotations blocks excessively")

			// Count .annotations references in pod context
			podAnnotationsPattern := regexp.MustCompile(`{{- with (?:omit )?\.annotations`)
			podAnnotationsMatches := podAnnotationsPattern.FindAllString(result2, -1)
			Expect(len(podAnnotationsMatches)).To(BeNumerically("<=", 2),
				"Should not duplicate pod.annotations blocks excessively")
		})

		It("should add pod annotations even when pod template has no annotations field", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Pod template with labels but NO annotations field
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    control-plane: controller-manager
  name: test-project-controller-manager
spec:
  template:
    metadata:
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should create annotations section before labels
			Expect(result).To(ContainSubstring("annotations:"))
			Expect(result).To(ContainSubstring("{{- with .Values.manager.pod }}"))
			Expect(result).To(ContainSubstring("{{- with .annotations }}"))

			// Verify annotations come before labels in pod template
			// Find the pod annotations block (either pattern)
			annotationsPattern := regexp.MustCompile(`{{- with (?:omit )?\.annotations`)
			labelsPattern := regexp.MustCompile(`{{- with (?:omit )?\.labels`)
			annotationsMatch := annotationsPattern.FindStringIndex(result)
			labelsMatch := labelsPattern.FindStringIndex(result)
			if annotationsMatch != nil && labelsMatch != nil {
				Expect(annotationsMatch[0]).To(BeNumerically("<", labelsMatch[0]), "Annotations should come before labels")
			}
		})

		It("should inject custom annotations into existing Deployment metadata annotations", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Deployment with existing annotations in metadata (e.g., from commonAnnotations)
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    existing-annotation: from-kustomize
    another-annotation: value
  labels:
    control-plane: controller-manager
  name: test-project-controller-manager
spec:
  template:
    metadata:
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should inject custom annotations into existing annotations block (with filtering)
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.annotations }}"),
				ContainSubstring("{{- if .Values.manager.annotations }}"),
			))
			// Verify annotations are properly nested under metadata:
			Expect(regexp.MustCompile(`(?m)^metadata:\n(?:^[ ]{2}.*\n)*^[ ]{2}annotations:\n`).MatchString(result)).To(BeTrue(),
				"manager.annotations should be nested under Deployment metadata")
			Expect(regexp.MustCompile(`(?ms)^annotations:\n.*?^spec:`).MatchString(result)).To(BeFalse(),
				"annotations should not be emitted as a top-level block before spec")

			// Should NOT create duplicate annotations: key
			annotationsCount := strings.Count(result, "annotations:")
			Expect(annotationsCount).To(Equal(2),
				"Should have exactly 2 annotations blocks (1 for Deployment, 1 for Pod template)")

			// Existing annotations should still be present
			Expect(result).To(ContainSubstring("existing-annotation: from-kustomize"))
			Expect(result).To(ContainSubstring("another-annotation: value"))

			// Verify injection order: custom annotations should come after existing ones
			// Find the manager annotations block (either pattern)
			annotationsPattern := regexp.MustCompile(`{{- (?:with|if) \.Values\.manager\.annotations`)
			annotationsMatch := annotationsPattern.FindStringIndex(result)
			existingAnnotationIdx := strings.Index(result, "existing-annotation: from-kustomize")
			if annotationsMatch != nil {
				Expect(annotationsMatch[0]).To(BeNumerically(">", existingAnnotationIdx),
					"Custom annotations should be injected after existing annotations")
			}
		})

		It("should convert empty map annotations: {} to block style for injection", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Deployment with empty map annotations: {}
			// Should be converted to block style to allow custom annotation injection
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations: {}
  labels:
    control-plane: controller-manager
  name: test-project-controller-manager
spec:
  template:
    metadata:
      annotations: {}
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Empty map should be converted to block style (not preserved as {})
			Expect(result).NotTo(ContainSubstring("annotations: {}"))

			// Should inject custom annotations template
			Expect(result).To(Or(
				ContainSubstring("{{- if .Values.manager.annotations }}"),
				ContainSubstring("{{- with .Values.manager.annotations }}"),
			))

			// Should have block-style annotations: key (not flow-style)
			Expect(regexp.MustCompile(`(?m)^  annotations:\n`).MatchString(result)).To(BeTrue(),
				"Empty map annotations should be converted to block style")
		})

		It("should handle inline annotations with values in Deployment metadata", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Deployment with inline annotations containing a value
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations: {existing: value}
  labels:
    control-plane: controller-manager
  name: test-project-controller-manager
spec:
  template:
    metadata:
      annotations: {kubectl.kubernetes.io/default-container: manager}
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should inject custom annotations (with automatic filtering)
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.annotations }}"),
				ContainSubstring("{{- if .Values.manager.annotations }}"),
			))
			// Verify annotations are properly nested under metadata:
			Expect(regexp.MustCompile(`(?m)^metadata:\n(?:^[ ]{2}.*\n)*^[ ]{2}annotations:\n`).MatchString(result)).To(BeTrue(),
				"manager.annotations should be nested under Deployment metadata")
			Expect(regexp.MustCompile(`(?ms)^annotations:\n.*?^spec:`).MatchString(result)).To(BeFalse(),
				"annotations should not be emitted as a top-level block before spec")

			// Should inject pod annotations (with automatic filtering)
			Expect(result).To(ContainSubstring("{{- with .Values.manager.pod }}"))
			Expect(result).To(ContainSubstring("{{- with .annotations }}"))

			// Should NOT create duplicate annotations: key
			annotationsCount := strings.Count(result, "annotations:")
			Expect(annotationsCount).To(Equal(2),
				"Should have exactly 2 annotations blocks (1 for Deployment, 1 for Pod template)")
		})

		It("should convert flow-style annotations to block-style before injecting templates", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Deployment with flow-style annotations (inline format)
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations: {existing: value, another: test}
  labels:
    control-plane: controller-manager
  name: test-project-controller-manager
spec:
  template:
    metadata:
      annotations: {kubectl.kubernetes.io/default-container: manager, other: value}
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Flow-style should be converted to block-style
			Expect(result).NotTo(ContainSubstring("annotations: {"))

			// Should have block-style annotations with existing entries preserved
			Expect(result).To(ContainSubstring("existing: value"))
			Expect(result).To(ContainSubstring("another: test"))

			// Should inject custom annotations template blocks (with automatic filtering)
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.annotations }}"),
				ContainSubstring("{{- if .Values.manager.annotations }}"),
			))

			// Verify the structure is valid (annotations: key followed by indented content)
			lines := strings.Split(result, "\n")
			foundDeploymentAnnotations := false
			for i, line := range lines {
				if strings.TrimSpace(line) == "annotations:" && !foundDeploymentAnnotations {
					foundDeploymentAnnotations = true
					// Next lines should be indented (existing values or templates)
					if i+1 < len(lines) {
						nextLine := lines[i+1]
						Expect(nextLine).To(MatchRegexp(`^\s{2,}`), // At least 2 spaces of indentation
							"Content after annotations: should be indented")
					}
				}
			}
			Expect(foundDeploymentAnnotations).To(BeTrue(), "Should have Deployment annotations block")
		})

		It("should convert flow-style annotations without space to block-style", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Test YAML variant without space: annotations:{...} (also valid YAML)
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:{existing: value, another: test}
  labels:
    control-plane: controller-manager
  name: test-project-controller-manager
spec:
  template:
    metadata:
      annotations:{kubectl.kubernetes.io/default-container: manager}
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Flow-style should be converted to block-style (both with and without space)
			Expect(result).NotTo(ContainSubstring("annotations: {"))
			Expect(result).NotTo(ContainSubstring("annotations:{"))

			// Should have block-style annotations with existing entries preserved
			Expect(result).To(ContainSubstring("existing: value"))
			Expect(result).To(ContainSubstring("another: test"))

			// Should inject custom annotations template blocks
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.annotations }}"),
				ContainSubstring("{{- if .Values.manager.annotations }}"),
			))
		})

		It("should handle annotations as last field before spec in Deployment metadata", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Deployment with annotations as LAST field in metadata (before spec)
			// This tests the edge case where there's no subsequent field to trigger injection
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    control-plane: controller-manager
  annotations:
    existing-annotation: value1
    another-annotation: value2
spec:
  replicas: 1
  template:
    metadata:
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should inject custom annotations after existing ones (with filtering)
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.annotations }}"),
				ContainSubstring("{{- if .Values.manager.annotations }}"),
			))
			// Verify annotations are properly nested under metadata:
			Expect(regexp.MustCompile(`(?m)^metadata:\n(?:^[ ]{2}.*\n)*^[ ]{2}annotations:\n`).MatchString(result)).To(BeTrue(),
				"manager.annotations should be nested under Deployment metadata")
			Expect(regexp.MustCompile(`(?ms)^annotations:\n.*?^spec:`).MatchString(result)).To(BeFalse(),
				"annotations should not be emitted as a top-level block before spec")
			Expect(result).To(ContainSubstring("existing-annotation: value1"))
			Expect(result).To(ContainSubstring("another-annotation: value2"))

			// Verify injection order: custom annotations should come after existing ones
			// Find the manager annotations block (either pattern)
			annotationsPattern := regexp.MustCompile(`{{- (?:with|if) \.Values\.manager\.annotations`)
			annotationsMatch := annotationsPattern.FindStringIndex(result)
			existingAnnotation1Idx := strings.Index(result, "existing-annotation: value1")
			existingAnnotation2Idx := strings.Index(result, "another-annotation: value2")
			if annotationsMatch != nil {
				Expect(annotationsMatch[0]).To(BeNumerically(">", existingAnnotation1Idx),
					"Custom annotations should be injected after first existing annotation")
				Expect(annotationsMatch[0]).To(BeNumerically(">", existingAnnotation2Idx),
					"Custom annotations should be injected after last existing annotation")
			}
		})

		It("should handle annotations with varying indentation from kustomize", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Test with 4-space indentation (some kustomize configurations)
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
    labels:
        control-plane: controller-manager
    annotations:
        existing-annotation: value1
    name: test-project-controller-manager
spec:
    template:
        metadata:
            annotations:
                kubectl.kubernetes.io/default-container: manager
            labels:
                control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should handle different indentation correctly (with automatic filtering)
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.annotations }}"),
				ContainSubstring("{{- if .Values.manager.annotations }}"),
			))
			// Verify annotations are properly nested under metadata:
			Expect(regexp.MustCompile(`(?m)^metadata:\n(?:^[ ]{2}.*\n)*^[ ]{2}annotations:\n`).MatchString(result)).To(BeTrue(),
				"manager.annotations should be nested under Deployment metadata")
			Expect(regexp.MustCompile(`(?ms)^annotations:\n.*?^spec:`).MatchString(result)).To(BeFalse(),
				"annotations should not be emitted as a top-level block before spec")
			Expect(result).To(ContainSubstring("{{- with .Values.manager.pod }}"))

			// Verify custom annotations come after existing ones
			deploymentAnnotationsPattern := regexp.MustCompile(`{{- (?:with|if) \.Values\.manager\.annotations`)
			deploymentAnnotationsMatch := deploymentAnnotationsPattern.FindStringIndex(result)
			deploymentExistingIdx := strings.Index(result, "existing-annotation: value1")
			if deploymentAnnotationsMatch != nil {
				Expect(deploymentAnnotationsMatch[0]).To(BeNumerically(">", deploymentExistingIdx))
			}

			podCustomIdx := strings.LastIndex(result, "{{- with .Values.manager.pod }}")
			podDefaultIdx := strings.Index(result, "kubectl.kubernetes.io/default-container: manager")
			Expect(podCustomIdx).To(BeNumerically(">", podDefaultIdx))
		})

		It("should add deployment annotations when name appears before labels", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Edge case: metadata.name comes before metadata.labels
			// Previous implementation relied on name: to trigger annotations injection
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  labels:
    control-plane: controller-manager
spec:
  template:
    metadata:
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should still inject annotations template even when no annotations field exists
			// and name appears before labels
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.annotations }}"),
				ContainSubstring("{{- if .Values.manager.annotations }}"),
			))
			// Verify annotations are properly nested under metadata:
			Expect(regexp.MustCompile(`(?m)^metadata:\n(?:^[ ]{2}.*\n)*^[ ]{2}annotations:\n`).MatchString(result)).To(BeTrue(),
				"manager.annotations should be nested under Deployment metadata")
			Expect(regexp.MustCompile(`(?ms)^annotations:\n.*?^spec:`).MatchString(result)).To(BeFalse(),
				"annotations should not be emitted as a top-level block before spec")
			Expect(result).To(ContainSubstring("annotations:"))
		})

		It("should inject deployment labels when labels is the last metadata field before spec", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetAPIVersion("apps/v1")
			deployment.SetKind("Deployment")
			deployment.SetName("test-project-controller-manager")

			// Edge case: labels is the last field in metadata, immediately followed by spec
			// Previous implementation would update position to positionAfterDeploymentMetadata
			// before checking shouldInjectDeploymentLabels
			content := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
  labels:
    control-plane: controller-manager
spec:
  replicas: 1
  template:
    metadata:
      labels:
        control-plane: controller-manager`

			result := templater.ApplyHelmSubstitutions(content, deployment)

			// Should inject custom labels template
			Expect(result).To(Or(
				ContainSubstring("{{- with .Values.manager.labels }}"),
				ContainSubstring("{{- if .Values.manager.labels }}"),
			))

			// Verify labels come after existing label
			labelsPattern := regexp.MustCompile(`{{- (?:with|if) \.Values\.manager\.labels`)
			labelsMatch := labelsPattern.FindStringIndex(result)
			existingLabelIdx := strings.Index(result, "control-plane: controller-manager")
			if labelsMatch != nil {
				Expect(labelsMatch[0]).To(BeNumerically(">", existingLabelIdx),
					"Custom labels should be injected after existing labels")
			}
		})
	})

	Context("conditional RBAC kind rendering", func() {
		It("should add conditional kind for ClusterRole to support namespace-scoped deployment", func() {
			clusterRoleResource := &unstructured.Unstructured{}
			clusterRoleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			clusterRoleResource.SetKind("ClusterRole")
			clusterRoleResource.SetName("test-project-manager-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-manager-role
rules:
- apiGroups:
  - apps
  resources:
  - deployments
  verbs:
  - get`

			result := templater.ApplyHelmSubstitutions(content, clusterRoleResource)

			// Should NOT have rbac.create (always created)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.rbac.create }}"))
			// Should have conditional kind based on rbac.namespaced
			Expect(result).To(ContainSubstring("{{- if .Values.rbac.namespaced }}"))
			Expect(result).To(ContainSubstring("kind: Role"))
			Expect(result).To(ContainSubstring("{{- else }}"))
			Expect(result).To(ContainSubstring("kind: ClusterRole"))
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
		})

		It("should add conditional kind for ClusterRoleBinding to support namespace-scoped deployment", func() {
			bindingResource := &unstructured.Unstructured{}
			bindingResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			bindingResource.SetKind("ClusterRoleBinding")
			bindingResource.SetName("test-project-manager-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: test-project-manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: test-project-manager-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system`

			result := templater.ApplyHelmSubstitutions(content, bindingResource)

			// Should NOT have rbac.create (always created)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.rbac.create }}"))
			// Should have conditional kind for binding
			Expect(result).To(ContainSubstring("{{- if .Values.rbac.namespaced }}"))
			Expect(result).To(ContainSubstring("kind: RoleBinding"))
			Expect(result).To(ContainSubstring("{{- else }}"))
			Expect(result).To(ContainSubstring("kind: ClusterRoleBinding"))
			// Should also make roleRef.kind conditional
			Expect(result).To(ContainSubstring("kind: Role"))
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
		})

		It("should NOT add any conditionals to Role (always namespace-scoped)", func() {
			roleResource := &unstructured.Unstructured{}
			roleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			roleResource.SetKind("Role")
			roleResource.SetName("test-project-leader-election-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-project-leader-election-role
  namespace: test-project-system
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  verbs:
  - get`

			result := templater.ApplyHelmSubstitutions(content, roleResource)

			// Should NOT have any RBAC conditionals (always created as-is)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.rbac"))
			// Should preserve kind: Role
			Expect(result).To(ContainSubstring("kind: Role"))
			// Should NOT have ClusterRole anywhere
			Expect(result).NotTo(ContainSubstring("ClusterRole"))
		})

		It("should NOT add any conditionals to RoleBinding (always namespace-scoped)", func() {
			bindingResource := &unstructured.Unstructured{}
			bindingResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			bindingResource.SetKind("RoleBinding")
			bindingResource.SetName("test-project-leader-election-rolebinding")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-project-leader-election-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: test-project-leader-election-role
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager
  namespace: test-project-system`

			result := templater.ApplyHelmSubstitutions(content, bindingResource)

			// Should NOT have any RBAC conditionals (always created as-is)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.rbac"))
			// Should preserve kind: RoleBinding
			Expect(result).To(ContainSubstring("kind: RoleBinding"))
			// roleRef should stay as Role
			Expect(result).To(ContainSubstring("kind: Role"))
			// Should NOT have ClusterRoleBinding anywhere
			Expect(result).NotTo(ContainSubstring("ClusterRoleBinding"))
		})

		It("should wrap metrics-auth ClusterRole with metrics.enable AND metrics.secure conditional", func() {
			metricsRoleResource := &unstructured.Unstructured{}
			metricsRoleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			metricsRoleResource.SetKind("ClusterRole")
			metricsRoleResource.SetName("test-project-metrics-auth-role")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-project-metrics-auth-role
rules:
- apiGroups:
  - authentication.k8s.io
  resources:
  - tokenreviews
  verbs:
  - create`

			result := templater.ApplyHelmSubstitutions(content, metricsRoleResource)

			// Should wrap with metrics.enable AND metrics.secure conditional
			Expect(result).To(ContainSubstring("{{- if and .Values.metrics.enable .Values.metrics.secure }}"))
			// Should NOT have rbac.create
			Expect(result).NotTo(ContainSubstring("{{- if and .Values.rbac.create"))
			// Metrics auth role is ALWAYS ClusterRole (never namespace-scoped)
			// So it should NOT have kind conditional
			Expect(result).To(ContainSubstring("kind: ClusterRole"))
			// Should have only one kind: ClusterRole (not conditional)
			Expect(strings.Count(result, "kind: ClusterRole")).To(Equal(1))
			Expect(result).NotTo(ContainSubstring("kind: Role"))
		})

		It("should NOT add any conditionals to ServiceAccount (always created)", func() {
			saResource := &unstructured.Unstructured{}
			saResource.SetAPIVersion("v1")
			saResource.SetKind("ServiceAccount")
			saResource.SetName("test-project-controller-manager")

			content := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-project-controller-manager
  namespace: test-project-system`

			result := templater.ApplyHelmSubstitutions(content, saResource)

			// Should NOT have any conditionals (always created)
			Expect(result).NotTo(ContainSubstring("{{- if .Values.rbac"))
			// ServiceAccount kind never changes
			Expect(result).NotTo(ContainSubstring("{{- if .Values.rbac.namespaced }}"))
		})
	})

	Context("multi-namespace RBAC support", func() {
		It("should preserve role-specific namespace deployments using .Values.rbac.roleNamespaces", func() {
			// Simulate role-namespace mappings
			roleNamespaces := map[string]string{
				"manager-role-infrastructure": "infrastructure",
				"manager-role-users":          "users",
			}

			multiNsTemplater := NewHelmTemplater("test-project", "test-project", "test-project-system", roleNamespaces)

			// Role in infrastructure namespace
			infraRole := &unstructured.Unstructured{}
			infraRole.SetAPIVersion("rbac.authorization.k8s.io/v1")
			infraRole.SetKind("Role")
			infraRole.SetName("manager-role-infrastructure")
			infraRole.SetNamespace("infrastructure")

			infraContent := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: manager-role-infrastructure
  namespace: infrastructure
rules:
- apiGroups: [apps]
  resources: [deployments]
  verbs: [get, list, watch]`

			infraResult := multiNsTemplater.ApplyHelmSubstitutions(infraContent, infraRole)

			// Should template to index .Values.rbac.roleNamespaces "manager-role-infrastructure" with default fallback
			expectedNs := `namespace: {{ index .Values.rbac.roleNamespaces ` +
				`"manager-role-infrastructure" | default "infrastructure" }}`
			Expect(infraResult).To(ContainSubstring(expectedNs))
			// Should NOT use .Release.Namespace
			Expect(infraResult).NotTo(ContainSubstring("namespace: {{ .Release.Namespace }}"))
		})

		It("should preserve namespace in RoleBinding subjects for multi-namespace RBAC", func() {
			roleNamespaces := map[string]string{
				"manager-rolebinding-users": "users",
			}

			multiNsTemplater := NewHelmTemplater("test-project", "test-project", "test-project-system", roleNamespaces)

			// RoleBinding in users namespace
			usersBinding := &unstructured.Unstructured{}
			usersBinding.SetAPIVersion("rbac.authorization.k8s.io/v1")
			usersBinding.SetKind("RoleBinding")
			usersBinding.SetName("manager-rolebinding-users")
			usersBinding.SetNamespace("users")

			bindingContent := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: manager-rolebinding-users
  namespace: users
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: manager-role-users
subjects:
- kind: ServiceAccount
  name: controller-manager
  namespace: test-project-system`

			result := multiNsTemplater.ApplyHelmSubstitutions(bindingContent, usersBinding)

			// RoleBinding namespace should use index .Values.rbac.roleNamespaces with binding name as key and default fallback
			Expect(result).To(ContainSubstring(
				`namespace: {{ index .Values.rbac.roleNamespaces "manager-rolebinding-users" | default "users" }}`))
			// Subject namespace should use .Release.Namespace (manager namespace)
			Expect(result).To(ContainSubstring("namespace: {{ .Release.Namespace }}"))
		})

		It("should handle multiple role-namespace mappings simultaneously", func() {
			roleNamespaces := map[string]string{
				"manager-role-infrastructure": "infrastructure",
				"manager-role-users":          "users",
				"manager-role-monitoring":     "monitoring",
			}

			multiNsTemplater := NewHelmTemplater("test-project", "test-project", "test-project-system", roleNamespaces)

			// Role in monitoring namespace
			monitoringRole := &unstructured.Unstructured{}
			monitoringRole.SetAPIVersion("rbac.authorization.k8s.io/v1")
			monitoringRole.SetKind("Role")
			monitoringRole.SetName("manager-role-monitoring")
			monitoringRole.SetNamespace("monitoring")

			monitoringContent := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: manager-role-monitoring
  namespace: monitoring
rules:
- apiGroups: [""]
  resources: [pods]
  verbs: [get, list]`

			result := multiNsTemplater.ApplyHelmSubstitutions(monitoringContent, monitoringRole)

			// Should preserve monitoring namespace with role-based template and default fallback
			Expect(result).To(ContainSubstring(
				`namespace: {{ index .Values.rbac.roleNamespaces "manager-role-monitoring" | default "monitoring" }}`))
		})

		It("should handle role names with hyphens correctly", func() {
			roleNamespaces := map[string]string{
				"manager-role": "app-infrastructure",
			}

			multiNsTemplater := NewHelmTemplater("test-project", "test-project", "test-project-system", roleNamespaces)

			roleResource := &unstructured.Unstructured{}
			roleResource.SetAPIVersion("rbac.authorization.k8s.io/v1")
			roleResource.SetKind("Role")
			roleResource.SetName("manager-role")
			roleResource.SetNamespace("app-infrastructure")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: manager-role
  namespace: app-infrastructure
rules:
- apiGroups: [apps]
  resources: [deployments]
  verbs: [get]`

			result := multiNsTemplater.ApplyHelmSubstitutions(content, roleResource)

			// Role name is used as key with default fallback
			Expect(result).To(ContainSubstring(
				`namespace: {{ index .Values.rbac.roleNamespaces "manager-role" | default "app-infrastructure" }}`))
		})

		It("should preserve DNS references for role-specific namespaces", func() {
			roleNamespaces := map[string]string{
				"manager-role": "infrastructure",
			}

			multiNsTemplater := NewHelmTemplater("test-project", "test-project", "test-project-system", roleNamespaces)

			configMap := &unstructured.Unstructured{}
			configMap.SetAPIVersion("v1")
			configMap.SetKind("ConfigMap")
			configMap.SetName("config")

			content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config
data:
  endpoint: "http://service.infrastructure.svc.cluster.local:8080"`

			result := multiNsTemplater.ApplyHelmSubstitutions(content, configMap)

			// ConfigMap is not in roleNamespaces map, so DNS refs won't be templated
			Expect(result).To(ContainSubstring("infrastructure.svc"))
		})

		It("should preserve resource references within role resources", func() {
			roleNamespaces := map[string]string{
				"manager-role": "infrastructure",
			}

			multiNsTemplater := NewHelmTemplater("test-project", "test-project", "test-project-system", roleNamespaces)

			// Role that has a reference to a resource in its namespace
			role := &unstructured.Unstructured{}
			role.SetAPIVersion("rbac.authorization.k8s.io/v1")
			role.SetKind("Role")
			role.SetName("manager-role")
			role.SetNamespace("infrastructure")

			content := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: manager-role
  namespace: infrastructure
  annotations:
    cert-manager.io/inject-ca-from: infrastructure/serving-cert
rules: []`

			result := multiNsTemplater.ApplyHelmSubstitutions(content, role)

			// Resource reference should be templated with index and default fallback
			Expect(result).To(ContainSubstring(
				`{{ index .Values.rbac.roleNamespaces "manager-role" | default "infrastructure" }}/serving-cert`))
		})
	})

	Context("ServiceAccount configuration", func() {
		Context("when managing ServiceAccount creation via values.yaml", func() {
			It("allows toggling ServiceAccount installation with enable flag", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				serviceAccount := &unstructured.Unstructured{}
				serviceAccount.SetAPIVersion("v1")
				serviceAccount.SetKind("ServiceAccount")
				serviceAccount.SetName("controller-manager")

				content := `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
    app.kubernetes.io/managed-by: kustomize
  name: controller-manager
  namespace: system`

				result := templater.ApplyHelmSubstitutions(content, serviceAccount)

				Expect(result).To(ContainSubstring("{{- if ne .Values.serviceAccount.enable false }}"))
				Expect(result).To(ContainSubstring("{{- end }}"))
			})

			It("supports custom annotations for cloud provider integrations", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				serviceAccount := &unstructured.Unstructured{}
				serviceAccount.SetAPIVersion("v1")
				serviceAccount.SetKind("ServiceAccount")
				serviceAccount.SetName("controller-manager")

				content := `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager`

				result := templater.ApplyHelmSubstitutions(content, serviceAccount)

				Expect(result).To(ContainSubstring("{{- with .Values.serviceAccount.annotations }}"))
				Expect(result).To(ContainSubstring("annotations:"))
				Expect(result).To(ContainSubstring("{{- toYaml . | nindent 4 }}"))
			})

			It("supports custom labels without duplicating existing standard labels", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				serviceAccount := &unstructured.Unstructured{}
				serviceAccount.SetAPIVersion("v1")
				serviceAccount.SetKind("ServiceAccount")
				serviceAccount.SetName("controller-manager")

				content := `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
    app.kubernetes.io/managed-by: kustomize
  name: controller-manager`

				result := templater.ApplyHelmSubstitutions(content, serviceAccount)

				Expect(result).To(ContainSubstring("{{- with .Values.serviceAccount.labels }}"))
				Expect(result).To(ContainSubstring(`{{- with omit .`))
				Expect(result).To(ContainSubstring(`"app.kubernetes.io/name"`))
				Expect(result).To(ContainSubstring(`"app.kubernetes.io/managed-by"`))
			})

			It("merges custom annotations with existing annotations without duplication", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				serviceAccount := &unstructured.Unstructured{}
				serviceAccount.SetAPIVersion("v1")
				serviceAccount.SetKind("ServiceAccount")
				serviceAccount.SetName("controller-manager")

				content := `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
  annotations:
    existing.annotation/key: "existing-value"
  name: controller-manager`

				result := templater.ApplyHelmSubstitutions(content, serviceAccount)

				// Should NOT have duplicate annotations: keys
				annotationsCount := strings.Count(result, "annotations:")
				Expect(annotationsCount).To(Equal(1), "Should have exactly one 'annotations:' key, found %d", annotationsCount)

				// Should preserve existing annotation
				Expect(result).To(ContainSubstring("existing.annotation/key"))

				// Should add template for custom annotations with duplicate filtering
				Expect(result).To(ContainSubstring("{{- with .Values.serviceAccount.annotations }}"))
				Expect(result).To(ContainSubstring(`{{- with omit .`))
				Expect(result).To(ContainSubstring(`"existing.annotation/key"`))
			})
		})

		Context("when using default ServiceAccount with nameOverride/fullnameOverride", func() {
			It("respects nameOverride and fullnameOverride for default ServiceAccount name", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				serviceAccount := &unstructured.Unstructured{}
				serviceAccount.SetAPIVersion("v1")
				serviceAccount.SetKind("ServiceAccount")
				serviceAccount.SetName("controller-manager")

				content := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager`

				result := templater.ApplyHelmSubstitutions(content, serviceAccount)

				Expect(result).To(ContainSubstring(`name: {{ include "test-project.serviceAccountName" . }}`))
			})

			It("ensures ServiceAccount name matches across all resource references", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				deployment := &unstructured.Unstructured{}
				deployment.SetAPIVersion("apps/v1")
				deployment.SetKind("Deployment")
				deployment.SetName("controller-manager")
				deployment.SetLabels(map[string]string{"control-plane": "controller-manager"})

				content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      serviceAccountName: controller-manager`

				result := templater.ApplyHelmSubstitutions(content, deployment)

				Expect(result).To(ContainSubstring(`serviceAccountName: {{ include "test-project.serviceAccountName" . }}`))
			})
		})

		Context("when binding RBAC permissions to ServiceAccount", func() {
			It("references ServiceAccount consistently in RoleBinding subjects", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				roleBinding := &unstructured.Unstructured{}
				roleBinding.SetAPIVersion("rbac.authorization.k8s.io/v1")
				roleBinding.SetKind("RoleBinding")
				roleBinding.SetName("leader-election-rolebinding")

				content := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: leader-election-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: leader-election-role
subjects:
- kind: ServiceAccount
  name: controller-manager
  namespace: system`

				result := templater.ApplyHelmSubstitutions(content, roleBinding)

				Expect(result).To(ContainSubstring(`- kind: ServiceAccount
  name: {{ include "test-project.serviceAccountName" . }}`))
			})

			It("references ServiceAccount consistently in ClusterRoleBinding subjects", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				clusterRoleBinding := &unstructured.Unstructured{}
				clusterRoleBinding.SetAPIVersion("rbac.authorization.k8s.io/v1")
				clusterRoleBinding.SetKind("ClusterRoleBinding")
				clusterRoleBinding.SetName("manager-rolebinding")

				content := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: manager-role
subjects:
- kind: ServiceAccount
  name: controller-manager
  namespace: system`

				result := templater.ApplyHelmSubstitutions(content, clusterRoleBinding)

				Expect(result).To(ContainSubstring(`- kind: ServiceAccount
  name: {{ include "test-project.serviceAccountName" . }}`))
			})
		})

		Context("when handling project names with prefixes", func() {
			It("templates ServiceAccount name correctly with project prefix", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				serviceAccount := &unstructured.Unstructured{}
				serviceAccount.SetAPIVersion("v1")
				serviceAccount.SetKind("ServiceAccount")
				serviceAccount.SetName("test-project-controller-manager")

				content := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-project-controller-manager`

				result := templater.ApplyHelmSubstitutions(content, serviceAccount)

				Expect(result).To(ContainSubstring(`name: {{ include "test-project.serviceAccountName" . }}`))
			})

			It("templates Deployment serviceAccountName field correctly with project prefix", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				deployment := &unstructured.Unstructured{}
				deployment.SetAPIVersion("apps/v1")
				deployment.SetKind("Deployment")
				deployment.SetName("controller-manager")
				deployment.SetLabels(map[string]string{"control-plane": "controller-manager"})

				content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      serviceAccountName: test-project-controller-manager`

				result := templater.ApplyHelmSubstitutions(content, deployment)

				Expect(result).To(ContainSubstring(`serviceAccountName: {{ include "test-project.serviceAccountName" . }}`))
			})

			It("templates RoleBinding subjects correctly with project prefix", func() {
				templater := NewHelmTemplater("test-project", "test-project", "test-project-system", nil)

				roleBinding := &unstructured.Unstructured{}
				roleBinding.SetAPIVersion("rbac.authorization.k8s.io/v1")
				roleBinding.SetKind("RoleBinding")
				roleBinding.SetName("manager-rolebinding")

				content := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
subjects:
- kind: ServiceAccount
  name: test-project-controller-manager`

				result := templater.ApplyHelmSubstitutions(content, roleBinding)

				Expect(result).To(ContainSubstring(`- kind: ServiceAccount
  name: {{ include "test-project.serviceAccountName" . }}`))
			})
		})

		Context("when ensuring Kubernetes resource name limits", func() {
			It("delegates truncation to resourceName helper for 63-character limit compliance", func() {
				templater := NewHelmTemplater("very-long-project-name-that-needs-truncation",
					"very-long-project-name-that-needs-truncation",
					"very-long-project-name-that-needs-truncation-system", nil)

				serviceAccount := &unstructured.Unstructured{}
				serviceAccount.SetAPIVersion("v1")
				serviceAccount.SetKind("ServiceAccount")
				serviceAccount.SetName("controller-manager")

				content := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager`

				result := templater.ApplyHelmSubstitutions(content, serviceAccount)

				Expect(result).To(ContainSubstring(
					`name: {{ include "very-long-project-name-that-needs-truncation.serviceAccountName" . }}`))
			})
		})
	})
})
