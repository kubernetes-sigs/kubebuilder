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
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// SubstituteProjectNames keeps original YAML as much as possible - only add Helm templating.
func SubstituteProjectNames(yamlContent string, _ *unstructured.Unstructured) string {
	return yamlContent
}

// SubstituteNamespace templates the manager namespace to {{ .Release.Namespace }} while
// preserving cross-namespace references (e.g., infrastructure, production).
//
// Uses regex patterns to safely replace:
//   - Namespace fields: `namespace: project-system` becomes `namespace: {{ .Release.Namespace }}`
//   - DNS names: `.project-system.svc` becomes `.{{ .Release.Namespace }}.svc`
//   - References: `project-system/cert` becomes `{{ .Release.Namespace }}/cert`
//
// Note: Uses regex (not YAML parsing) because content contains Helm templates from prior steps.
func SubstituteNamespace(
	detectedPrefix, chartName, managerNamespace string,
	roleNamespaces map[string]string,
	yamlContent string,
	resource *unstructured.Unstructured,
) string {
	namespaceTemplate := "{{ .Release.Namespace }}"

	// Multi-namespace RBAC scenario: Operator watches multiple namespaces but roles must be
	// deployed to each watched namespace separately. Uses .Values.rbac.roleNamespaces map
	// to template namespace per-role, with fallback to the original namespace.
	// Example: Watch namespaces ["infra", "prod"] - roles go to those namespaces, not Release.Namespace.
	resourceName := resource.GetName()
	suffix := strings.TrimPrefix(resourceName, detectedPrefix+"-")

	if targetNs, found := roleNamespaces[suffix]; found {
		roleTemplate := fmt.Sprintf("{{ index .Values.rbac.roleNamespaces %q | default %q }}", suffix, targetNs)

		nsPattern := regexp.MustCompile(`(?m)^(\s*)namespace:\s+` + regexp.QuoteMeta(targetNs) + `\s*$`)
		yamlContent = nsPattern.ReplaceAllString(yamlContent, "${1}namespace: "+roleTemplate)

		refPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(targetNs) + `/`)
		yamlContent = refPattern.ReplaceAllString(yamlContent, roleTemplate+"/")

		dnsPattern := regexp.MustCompile(`\.` + regexp.QuoteMeta(targetNs) + `\.`)
		yamlContent = dnsPattern.ReplaceAllString(yamlContent, "."+roleTemplate+".")
	}

	// Replace namespace fields
	namespaceFieldPattern := regexp.MustCompile(`(?m)^(\s*)namespace:\s+` + regexp.QuoteMeta(managerNamespace) + `\s*$`)
	yamlContent = namespaceFieldPattern.ReplaceAllString(yamlContent, "${1}namespace: "+namespaceTemplate)

	// Replace resource references in format "namespace/resource"
	refPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(managerNamespace) + `/`)
	yamlContent = refPattern.ReplaceAllString(yamlContent, namespaceTemplate+"/")

	// Replace DNS names in format ".namespace.svc"
	// Dots on both sides prevent matching resource names or labels

	dnsPattern := regexp.MustCompile(`\.` + regexp.QuoteMeta(managerNamespace) + `\.`)
	yamlContent = dnsPattern.ReplaceAllString(yamlContent, "."+namespaceTemplate+".")

	// Certificate-specific DNS templating
	if resource.GetKind() == common.KindCertificate {
		yamlContent = SubstituteCertificateDNSNames(detectedPrefix, chartName, yamlContent, resource)
	}

	return yamlContent
}

// SubstituteCertificateDNSNames replaces hardcoded DNS names in certificates with proper service templates.
func SubstituteCertificateDNSNames(
	detectedPrefix, chartName string, yamlContent string, resource *unstructured.Unstructured,
) string {
	name := resource.GetName()

	isMetricsCert := strings.HasSuffix(name, "-metrics-certs") || strings.HasSuffix(name, "-metrics-cert")
	isServingCert := strings.HasSuffix(name, "-serving-cert")

	if isMetricsCert {
		metricsServiceTemplate := "{{ include \"" + chartName + ".resourceName\" " +
			"(dict \"suffix\" \"controller-manager-metrics-service\" \"context\" $) }}"
		metricsServiceFQDN := metricsServiceTemplate + ".{{ include \"" + chartName + ".namespaceName\" $ }}.svc"
		metricsServiceFQDNCluster := metricsServiceTemplate +
			".{{ include \"" + chartName + ".namespaceName\" $ }}.svc.cluster.local"

		yamlContent = strings.ReplaceAll(yamlContent, "SERVICE_NAME.SERVICE_NAMESPACE.svc", metricsServiceFQDN)
		yamlContent = strings.ReplaceAll(yamlContent,
			"SERVICE_NAME.SERVICE_NAMESPACE.svc.cluster.local", metricsServiceFQDNCluster)

		hardcodedMetricsService := detectedPrefix + "-controller-manager-metrics-service"
		yamlContent = strings.ReplaceAll(yamlContent, hardcodedMetricsService, metricsServiceTemplate)
	} else if isServingCert {
		hardcodedWebhookServiceShort := detectedPrefix + "-webhook-service"
		webhookServiceTemplate := ResourceNameTemplate(chartName, "webhook-service")
		yamlContent = strings.ReplaceAll(yamlContent, hardcodedWebhookServiceShort, webhookServiceTemplate)
	}

	return yamlContent
}

// ResourceNameTemplate creates a Helm template for a resource name with 63-char safety.
// Uses <chartname>.resourceName helper which intelligently truncates when base + suffix > 63 chars.
// Template name is scoped to the chart to prevent collisions when used as a Helm dependency.
func ResourceNameTemplate(chartName, suffix string) string {
	return `{{ include "` + chartName + `.resourceName" (dict "suffix" "` + suffix + `" "context" $) }}`
}

// SubstituteResourceNamesWithPrefix templates ALL resource names using the chart.resourceName helper.
// Generic regex-based approach works for any resource type without hardcoding specific names.
// Excludes container names and ServiceAccount metadata.name (handled separately).
func SubstituteResourceNamesWithPrefix(
	detectedPrefix, chartName string,
	yamlContent string,
	resource *unstructured.Unstructured,
) string {
	namePattern := regexp.MustCompile(
		`(\s+)([a-zA-Z]*[Nn]ame):\s+` + regexp.QuoteMeta(detectedPrefix) + `(-[a-zA-Z0-9-]+)`)

	lines := strings.Split(yamlContent, "\n")
	result := make([]string, 0, len(lines))

	isServiceAccount := resource.GetKind() == common.KindServiceAccount

	// Extract actual container names from the structured object so
	// they can be skipped without backward text-scanning.
	containerNameSet := ExtractContainerNames(resource)

	for i, line := range lines {
		if !namePattern.MatchString(line) {
			result = append(result, line)
			continue
		}

		isContainerName := false
		if len(containerNameSet) > 0 && strings.Contains(line, "name:") {
			parts := namePattern.FindStringSubmatch(line)
			if len(parts) >= 4 {
				candidateName := detectedPrefix + parts[3]
				if containerNameSet[candidateName] {
					isContainerName = true
				}
			}
		}

		// For ServiceAccount, skip metadata.name (handled by templateServiceAccountName)
		// but still process other name fields like imagePullSecrets[].name or secrets[].name
		isServiceAccountMetadataName := false
		if isServiceAccount && strings.Contains(line, "name:") {
			// Check if this is metadata.name by looking for "metadata:" with proper indentation
			// metadata.name has 2-space indentation (one level under metadata:)
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))

			// Look backward for "metadata:" - it should be at currentIndent-2
			for j := i - 1; j >= 0; j-- {
				prevLine := lines[j]
				trimmed := strings.TrimSpace(prevLine)

				// Skip empty lines and comments
				if trimmed == "" || strings.HasPrefix(trimmed, "#") {
					continue
				}

				prevIndent := len(prevLine) - len(strings.TrimLeft(prevLine, " \t"))

				// If we find metadata: at the parent indentation level, this is metadata.name
				if prevIndent == currentIndent-2 && trimmed == common.YamlKeyMetadata {
					isServiceAccountMetadataName = true
					break
				}

				// Stop if we hit a line at same or less indentation (different section)
				if prevIndent <= currentIndent-2 && strings.HasSuffix(trimmed, ":") {
					break
				}
			}
		}

		if isContainerName || isServiceAccountMetadataName {
			// Don't template container names or SA metadata.name - keep them as-is
			result = append(result, line)
		} else {
			// Template other resource names
			templatedLine := namePattern.ReplaceAllStringFunc(line, func(match string) string {
				parts := namePattern.FindStringSubmatch(match)
				if len(parts) < 4 {
					return match
				}

				indent := parts[1]
				fieldName := parts[2]
				suffix := parts[3][1:] // Remove leading dash

				return indent + fieldName + ": " + ResourceNameTemplate(chartName, suffix)
			})
			result = append(result, templatedLine)
		}
	}

	return strings.Join(result, "\n")
}
