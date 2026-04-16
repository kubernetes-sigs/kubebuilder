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
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// TemplatePorts templates port numbers for Services and Deployments using values.yaml.
func TemplatePorts(yamlContent string, resource *unstructured.Unstructured) string {
	resourceName := resource.GetName()
	resourceKind := resource.GetKind()

	// Use suffix matching to avoid false positives when project name contains "webhook"
	isWebhook := resourceKind == common.KindService &&
		strings.HasSuffix(resourceName, "-webhook-service")

	// Use suffix matching to avoid false positives when project name contains "metrics"
	isMetrics := resourceKind == common.KindService &&
		(strings.HasSuffix(resourceName, "-controller-manager-metrics-service") ||
			strings.HasSuffix(resourceName, "-metrics-service"))

	// For Deployments, detect webhook ports from content
	if resourceKind == common.KindDeployment {
		if strings.Contains(yamlContent, "webhook-server") || strings.Contains(yamlContent, "name: webhook") {
			isWebhook = true
		}
	}

	// Template webhook ports
	if isWebhook {
		// Replace containerPort for webhook-server with template (matches any numeric port)
		if strings.Contains(yamlContent, "webhook-server") {
			yamlContent = regexp.MustCompile(`(?m)(\s*- )?containerPort:\s*\d+(\s*\n\s*name:\s*webhook-server)`).
				ReplaceAllString(yamlContent, "${1}containerPort: {{ .Values.webhook.port }}${2}")
		}

		// Replace targetPort with webhook.port template (matches any numeric port)
		yamlContent = regexp.MustCompile(`(\s*)targetPort:\s*\d+`).
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.webhook.port }}")
	}

	// Template metrics ports
	if isMetrics {
		// Replace port with metrics.port template (matches any numeric port)
		yamlContent = regexp.MustCompile(`(\s*)port:\s*\d+`).
			ReplaceAllString(yamlContent, "${1}port: {{ .Values.metrics.port }}")

		// Replace targetPort with metrics.port template (matches any numeric port)
		yamlContent = regexp.MustCompile(`(\s*)targetPort:\s*\d+`).
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.metrics.port }}")

		// Template port name based on metrics.secure (http vs https)
		// This ensures Service and ServiceMonitor use the correct scheme
		if resource.GetKind() == common.KindService {
			yamlContent = regexp.MustCompile(`(\s*)- name:\s*https(\s+port:)`).
				ReplaceAllString(yamlContent, `${1}- name: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}${2}`)
		}
	}

	// Template port-related arguments in Deployment
	if resource.GetKind() == common.KindDeployment {
		// Replace --metrics-bind-address with templated port
		// Supports :PORT, HOST:PORT, and IPv6 [::1]:PORT formats
		yamlContent = regexp.MustCompile(`--metrics-bind-address=(\[[^\]]*\]|[^\s:]*):([0-9]+)`).
			ReplaceAllString(yamlContent, "--metrics-bind-address=$1:{{ .Values.metrics.port }}")

		// Replace --webhook-port with templated version (matches any numeric port)
		yamlContent = regexp.MustCompile(`--webhook-port=([0-9]+)`).
			ReplaceAllString(yamlContent, "--webhook-port={{ .Values.webhook.port }}")
	}

	return yamlContent
}
