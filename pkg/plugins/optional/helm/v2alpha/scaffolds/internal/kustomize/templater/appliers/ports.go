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

// Port-templating regexes are compiled once at package initialization rather than on
// every call. Patterns that only differ by replacement string (port/targetPort) are
// shared across the webhook and metrics substitutions.
var (
	webhookContainerPortRE = regexp.MustCompile(`(?m)(\s*- )?containerPort:\s*\d+(\s*\n\s*name:\s*webhook-server)`)
	metricsBindAddressRE   = regexp.MustCompile(`--metrics-bind-address=(\[[^\]]*\]|[^\s:]*):([0-9]+)`)
	webhookPortArgRE       = regexp.MustCompile(`--webhook-port=([0-9]+)`)
	portRE                 = regexp.MustCompile(`(\s*)port:\s*\d+`)
	targetPortRE           = regexp.MustCompile(`(\s*)targetPort:\s*\d+`)
	metricsHTTPSNameRE     = regexp.MustCompile(`(\s*)- name:\s*https(\s+port:)`)

	healthProbeBindAddressRE = regexp.MustCompile(`--health-probe-bind-address=(\[[^\]]*\]|[^\s:]*):([0-9]+)`)
	healthContainerPortRE    = regexp.MustCompile(`(?m)(\s*- )?containerPort:\s*\d+(\s*\n\s*name:\s*health\b)`)
	healthProbeHTTPGetPortRE = regexp.MustCompile(`(path:\s*/(?:healthz|readyz)[ \t]*\n\s*port:\s*)\d+`)
)

// TemplatePorts templates port numbers for Services, Deployments, and NetworkPolicies using values.yaml.
func TemplatePorts(yamlContent string, resource *unstructured.Unstructured) string {
	// For Deployments, port/probe/argument templating is scoped to the manager
	// container so sidecars that happen to use the same ports, probe paths, or
	// bind-address flags are left untouched.
	if resource.GetKind() == common.KindDeployment {
		return applyToManagerContainer(yamlContent, templateManagerContainerPorts)
	}

	return templateServicePorts(yamlContent, resource)
}

// templateManagerContainerPorts templates the manager container's port-related fields:
// the webhook containerPort, the metrics and webhook bind-address arguments, and the
// health probe port. It is always applied to the manager container block only, so
// sidecar ports and probes are never rewritten.
func templateManagerContainerPorts(managerContainer string) string {
	// Replace containerPort for webhook-server with template (matches any numeric port).
	// The regex is self-guarding: it only matches a containerPort named "webhook-server".
	managerContainer = webhookContainerPortRE.
		ReplaceAllString(managerContainer, "${1}containerPort: {{ .Values.webhook.port }}${2}")

	// Replace --metrics-bind-address with templated port.
	// Supports :PORT, HOST:PORT, and IPv6 [::1]:PORT formats.
	managerContainer = metricsBindAddressRE.
		ReplaceAllString(managerContainer, "--metrics-bind-address=$1:{{ .Values.metrics.port }}")

	// Replace --webhook-port with templated version (matches any numeric port).
	managerContainer = webhookPortArgRE.
		ReplaceAllString(managerContainer, "--webhook-port={{ .Values.webhook.port }}")

	return templateHealthProbePort(managerContainer)
}

// templateServicePorts templates the ports of the webhook and metrics Services and
// NetworkPolicies. These resources have no containers, so they are identified by name.
func templateServicePorts(yamlContent string, resource *unstructured.Unstructured) string {
	resourceName := resource.GetName()
	resourceKind := resource.GetKind()

	// Use suffix matching to avoid false positives when project name contains "webhook"
	isWebhook := (resourceKind == common.KindService && strings.HasSuffix(resourceName, "-webhook-service")) ||
		(resourceKind == common.KindNetworkPolicy && strings.HasSuffix(resourceName, "allow-webhook-traffic"))

	// Use suffix matching to avoid false positives when project name contains "metrics"
	isMetrics := (resourceKind == common.KindService &&
		(strings.HasSuffix(resourceName, "-controller-manager-metrics-service") ||
			strings.HasSuffix(resourceName, "-metrics-service"))) ||
		(resourceKind == common.KindNetworkPolicy && strings.HasSuffix(resourceName, "allow-metrics-traffic"))

	// Template webhook ports
	if isWebhook {
		if resourceKind == common.KindNetworkPolicy {
			yamlContent = portRE.
				ReplaceAllString(yamlContent, "${1}port: {{ .Values.webhook.port }}")
			return yamlContent
		}

		// Replace targetPort with webhook.port template (matches any numeric port)
		yamlContent = targetPortRE.
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.webhook.port }}")
	}

	// Template metrics ports
	if isMetrics {
		// Replace port with metrics.port template (matches any numeric port)
		yamlContent = portRE.
			ReplaceAllString(yamlContent, "${1}port: {{ .Values.metrics.port }}")

		if resourceKind == common.KindNetworkPolicy {
			return yamlContent
		}

		// Replace targetPort with metrics.port template (matches any numeric port)
		yamlContent = targetPortRE.
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.metrics.port }}")

		// Template port name based on metrics.secure (http vs https)
		// This ensures Service and ServiceMonitor use the correct scheme
		if resourceKind == common.KindService {
			yamlContent = metricsHTTPSNameRE.
				ReplaceAllString(yamlContent, `${1}- name: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}${2}`)
		}
	}

	return yamlContent
}

// templateHealthProbePort templates the manager health probe port so it can be
// configured from values.yaml, mirroring how metrics and webhook ports are handled.
// It rewrites the four places the port appears in the manager Deployment: the
// --health-probe-bind-address arg, the "health" containerPort, and the liveness
// and readiness httpGet ports.
func templateHealthProbePort(yamlContent string) string {
	const healthPortTemplate = "{{ .Values.manager.healthProbe.port }}"

	// --health-probe-bind-address=:PORT (also HOST:PORT and IPv6 [::1]:PORT)
	yamlContent = healthProbeBindAddressRE.
		ReplaceAllString(yamlContent, "--health-probe-bind-address=$1:"+healthPortTemplate)

	// containerPort for the port named "health"
	yamlContent = healthContainerPortRE.
		ReplaceAllString(yamlContent, "${1}containerPort: "+healthPortTemplate+"${2}")

	// liveness (/healthz) and readiness (/readyz) httpGet ports
	yamlContent = healthProbeHTTPGetPortRE.
		ReplaceAllString(yamlContent, "${1}"+healthPortTemplate)

	return yamlContent
}
