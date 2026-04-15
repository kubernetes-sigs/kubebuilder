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
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const (
	kindNamespace          = "Namespace"
	kindCertificate        = "Certificate"
	kindService            = "Service"
	kindServiceAccount     = "ServiceAccount"
	kindRole               = "Role"
	kindClusterRole        = "ClusterRole"
	kindRoleBinding        = "RoleBinding"
	kindClusterRoleBinding = "ClusterRoleBinding"
	kindServiceMonitor     = "ServiceMonitor"
	kindIssuer             = "Issuer"
	kindValidatingWebhook  = "ValidatingWebhookConfiguration"
	kindMutatingWebhook    = "MutatingWebhookConfiguration"
	kindDeployment         = "Deployment"
	kindCRD                = "CustomResourceDefinition"

	// API versions
	apiVersionCertManager = "cert-manager.io/v1"
	apiVersionMonitoring  = "monitoring.coreos.com/v1"

	// YAML keys
	yamlKeyAnnotations = "annotations:"
	yamlKeyLabels      = "labels:"

	// Standard Kubernetes/Helm label keys
	labelKeyAppName      = "app.kubernetes.io/name:"
	labelKeyAppInstance  = "app.kubernetes.io/instance:"
	labelKeyAppManagedBy = "app.kubernetes.io/managed-by:"
	labelKeyHelmChart    = "helm.sh/chart:"
)

// HelmTemplater handles converting YAML content to Helm templates
type HelmTemplater struct {
	detectedPrefix   string
	chartName        string
	managerNamespace string
	// roleNamespaces maps RBAC resource suffixes (without project prefix) to their target namespaces
	// for multi-namespace deployments. This enables Roles/RoleBindings to be deployed to specific
	// namespaces outside the manager namespace.
	// Example: {"manager-role-infrastructure": "infrastructure"} (key is suffix, not full name)
	// These are templated as: {{ index .Values.rbac.roleNamespaces "suffix" | default "namespace" }}
	roleNamespaces map[string]string
}

// NewHelmTemplater creates a new Helm templater
func NewHelmTemplater(
	detectedPrefix, chartName, managerNamespace string, roleNamespaces map[string]string,
) *HelmTemplater {
	return &HelmTemplater{
		detectedPrefix:   detectedPrefix,
		chartName:        chartName,
		managerNamespace: managerNamespace,
		roleNamespaces:   roleNamespaces,
	}
}

// getDefaultContainerName extracts the container name from kubectl.kubernetes.io/default-container annotation.
// This allows the Helm plugin to work with any container name, not just "manager".
// If the annotation is not found, it falls back to "manager" for backward compatibility.
func (t *HelmTemplater) getDefaultContainerName(yamlContent string) string {
	// Look for kubectl.kubernetes.io/default-container annotation
	pattern := regexp.MustCompile(`(?m)^\s*kubectl\.kubernetes\.io/default-container:\s+(\S+)`)
	matches := pattern.FindStringSubmatch(yamlContent)
	if len(matches) > 1 {
		return matches[1]
	}
	// Fallback to "manager" for backward compatibility with older scaffolds
	return "manager"
}

// resourceNameTemplate creates a Helm template for a resource name with 63-char safety.
// Uses <chartname>.resourceName helper which intelligently truncates when base + suffix > 63 chars.
// Template name is scoped to the chart to prevent collisions when used as a Helm dependency.
func (t *HelmTemplater) resourceNameTemplate(suffix string) string {
	return `{{ include "` + t.chartName + `.resourceName" (dict "suffix" "` + suffix + `" "context" $) }}`
}

// ApplyHelmSubstitutions converts YAML content to use Helm template syntax
func (t *HelmTemplater) ApplyHelmSubstitutions(yamlContent string, resource *unstructured.Unstructured) string {
	// Escape existing Go template syntax ({{ }}) FIRST before adding Helm templates.
	// Resources from install.yaml may contain templates that should be preserved as literal text.
	// For example: CRD default values, ConfigMap data, Secret URLs, annotations, etc.
	yamlContent = t.escapeExistingTemplateSyntax(yamlContent)

	// Apply conditional wrappers first
	yamlContent = t.addConditionalWrappers(yamlContent, resource)

	// Apply general project name substitutions
	yamlContent = t.substituteProjectNames(yamlContent, resource)

	// Apply namespace substitutions
	yamlContent = t.substituteNamespace(yamlContent, resource)

	// Apply cert-manager and webhook-specific templating AFTER other substitutions
	yamlContent = t.substituteCertManagerReferences(yamlContent, resource)

	yamlContent = t.substituteResourceNamesWithPrefix(yamlContent, resource)

	// Apply labels and annotations from Helm chart
	yamlContent = t.addHelmLabelsAndAnnotations(yamlContent, resource)

	// Apply resource-specific substitutions
	yamlContent = t.substituteRBACValues(yamlContent)

	// Apply ServiceAccount-specific templating
	if resource.GetKind() == kindServiceAccount {
		yamlContent = t.templateServiceAccount(yamlContent)
	}

	// Apply deployment-specific templating
	if resource.GetKind() == kindDeployment && isManagerDeployment(resource) {
		yamlContent = t.addCustomLabelsAndAnnotations(yamlContent)
		yamlContent = t.templateDeploymentFields(yamlContent)

		// Apply conditional logic for cert-manager related fields in deployments
		yamlContent = t.makeContainerArgsConditional(yamlContent)
		yamlContent = t.makeWebhookVolumeMountsConditional(yamlContent)
		yamlContent = t.makeWebhookVolumesConditional(yamlContent)
		yamlContent = t.makeMetricsVolumeMountsConditional(yamlContent)
		yamlContent = t.makeMetricsVolumesConditional(yamlContent)
	}

	// Apply port templating for Services and Deployments
	if resource.GetKind() == kindService || resource.GetKind() == kindDeployment {
		yamlContent = t.templatePorts(yamlContent, resource)
	}

	// Apply ServiceMonitor templating for metrics scheme and port
	if resource.GetKind() == kindServiceMonitor {
		yamlContent = t.templateServiceMonitor(yamlContent)
	}

	// Final tidy-up: avoid accidental blank lines after Helm if-block starts
	// Some replacements may introduce an empty line between a `{{- if ... }}`
	// and the following content; collapse that to ensure consistent formatting.
	yamlContent = t.collapseBlankLineAfterIf(yamlContent)

	return yamlContent
}

// escapeExistingTemplateSyntax escapes Go template syntax ({{ }}) in YAML to prevent
// Helm from parsing them. Converts existing templates to literal strings that Helm outputs as-is.
//
// Why this is needed:
// Resources from install.yaml may contain {{ }} in string fields that are NOT Helm templates.
// Without escaping, Helm will try to evaluate them and fail. For example:
//
//	CRD default: "Branch: {{ .Spec.Branch }}"  ->  ERROR: .Spec undefined
//
// How it works:
// Wraps non-Helm templates in string literals so Helm outputs them unchanged:
//
//	{{ .Field }}  ->  {{ "{{ .Field }}" }}
//
// When Helm renders this, it outputs the literal string: {{ .Field }}
//
// Smart detection:
// Only escapes templates that DON'T start with Helm keywords:
//   - .Release, .Values, .Chart (Helm built-ins)
//   - include, if, with, range, toYaml (Helm functions)
//
// This means our Helm templates work normally while existing templates are preserved.
func (t *HelmTemplater) escapeExistingTemplateSyntax(yamlContent string) string {
	// (?s) makes '.' match newlines so split-line templates produced by sigs.k8s.io/yaml's
	// ~80-column folding (e.g. "{{ .LongName\n    }}") are matched in a single pass.
	templatePattern := regexp.MustCompile(`(?s)\{\{(.*?)\}\}`)

	yamlContent = templatePattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
		// Extract content between {{ and }}
		content := strings.TrimPrefix(match, "{{")
		content = strings.TrimSuffix(content, "}}")
		trimmedContent := strings.TrimSpace(content)

		// Check if this is a Helm template (starts with Helm keyword)
		helmPatterns := []string{
			"include ", "- include ",
			".Release.", "- .Release.",
			".Values.", "- .Values.",
			".Chart.", "- .Chart.",
			"toYaml ", "- toYaml ",
			"if ", "- if ",
			"end ", "- end ",
			"with ", "- with ",
			"range ", "- range ",
			"else", "- else",
		}

		// If it's a Helm template, keep it as-is
		for _, pattern := range helmPatterns {
			if strings.HasPrefix(trimmedContent, pattern) {
				return match
			}
		}

		// Otherwise, escape it to preserve as literal text
		// Collapse any newline+indent that sigs.k8s.io/yaml may have introduced via line-wrapping.
		collapsed := regexp.MustCompile(`\n[ \t]+`).ReplaceAllString(content, " ")

		// Before re-escaping for Go template string literals, unescape any YAML double-quoted
		// scalar escape sequences. yaml.Marshal emits \" for a literal " inside a double-quoted
		// YAML scalar; without this step the subsequent " to \" replacement double-escapes them to
		// \\" which breaks Helm's Go template parser: \\ becomes one backslash, then the next "
		// closes the string prematurely, leaving tokens like "asset-id" outside where "-" is a
		// bad character (U+002D).
		unescaped := strings.ReplaceAll(collapsed, `\"`, `"`)
		escapedContent := strings.ReplaceAll(unescaped, `"`, `\"`)

		// Wrap in Helm string literal: {{ "{{...}}" }}
		return `{{ "{{` + escapedContent + `}}" }}`
	})

	return yamlContent
}

// substituteProjectNames keeps original YAML as much as possible - only add Helm templating
func (t *HelmTemplater) substituteProjectNames(yamlContent string, _ *unstructured.Unstructured) string {
	return yamlContent
}

// substituteNamespace replaces manager namespace references with {{ .Release.Namespace }}
// while preserving cross-namespace references (e.g., infrastructure, production).
//
// DESIGN RATIONALE:
// We use regex-based replacement (not YAML parsing) because the content already contains
// Helm templates from previous substitutions, which would break YAML parsing.
//
// SAFETY GUARANTEES:
// 1. Namespace fields: Only replaces `namespace: <exact-value>` (line-anchored regex)
// 2. DNS names: Only replaces `.<namespace>.` (dots on both sides prevent substring matches)
// 3. References: Only replaces `<namespace>/` (word boundary prevents false matches)
//
// TESTED SCENARIOS:
// - All standard K8s resource types (ConfigMap, Secret, Ingress, etc.)
// - All monitoring resources (ServiceMonitor, PodMonitor)
// - All RBAC resources (Role, RoleBinding, with cross-namespace support)
// - All DNS patterns (.svc, .svc.cluster.local, .pod, .endpoints)
// - Custom CRDs with any structure
// - Cross-namespace preservation (infrastructure, production, etc.)
// - Substring bug prevention (namespace "user" doesn't break resource "users")
func (t *HelmTemplater) substituteNamespace(yamlContent string, resource *unstructured.Unstructured) string {
	managerNamespace := t.managerNamespace
	namespaceTemplate := "{{ .Release.Namespace }}"

	// 0. MULTI-NAMESPACE RBAC: Handle specific role deployments first.
	//    For Roles/RoleBindings deployed to specific namespaces (not manager namespace),
	//    template them to use index-based access into .Values.rbac.roleNamespaces
	//    with the resource suffix (without project prefix) as the key.
	//    This makes keys stable across different release names and overrides.
	//    If the key is missing, fall back to the original namespace from kustomize.
	resourceName := resource.GetName()
	// Extract suffix by removing detected prefix
	suffix := strings.TrimPrefix(resourceName, t.detectedPrefix+"-")

	if targetNs, found := t.roleNamespaces[suffix]; found {
		// This resource should use role-based namespace template with fallback to original namespace
		// Use suffix as the lookup key (e.g., "manager-role-infrastructure" not "project-manager-role-infrastructure")
		roleTemplate := fmt.Sprintf("{{ index .Values.rbac.roleNamespaces %q | default %q }}", suffix, targetNs)

		// Replace namespace field for this resource
		nsPattern := regexp.MustCompile(`(?m)^(\s*)namespace:\s+` + regexp.QuoteMeta(targetNs) + `\s*$`)
		yamlContent = nsPattern.ReplaceAllString(yamlContent, "${1}namespace: "+roleTemplate)

		// Replace resource references for this namespace
		refPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(targetNs) + `/`)
		yamlContent = refPattern.ReplaceAllString(yamlContent, roleTemplate+"/")

		// Replace DNS names for this namespace
		dnsPattern := regexp.MustCompile(`\.` + regexp.QuoteMeta(targetNs) + `\.`)
		yamlContent = dnsPattern.ReplaceAllString(yamlContent, "."+roleTemplate+".")
	}

	// 1. NAMESPACE FIELDS: Replace `namespace: <manager-namespace>`
	//    Pattern: Line-anchored to prevent false matches
	//    Example: `namespace: project-system` becomes `namespace: {{ .Release.Namespace }}`
	namespaceFieldPattern := regexp.MustCompile(`(?m)^(\s*)namespace:\s+` + regexp.QuoteMeta(managerNamespace) + `\s*$`)
	yamlContent = namespaceFieldPattern.ReplaceAllString(yamlContent, "${1}namespace: "+namespaceTemplate)

	// 2. RESOURCE REFERENCES: Replace `<manager-namespace>/resource-name`
	//    Pattern: Word boundary ensures we don't match partial words
	//    Example: `cert-manager.io/inject-ca-from: project-system/cert` becomes `{{ .Release.Namespace }}/cert`
	//    Example: `configMapRef: project-system/config` becomes `{{ .Release.Namespace }}/config`
	refPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(managerNamespace) + `/`)
	yamlContent = refPattern.ReplaceAllString(yamlContent, namespaceTemplate+"/")

	// 3. DNS NAMES: Replace `.<manager-namespace>.` in Kubernetes DNS patterns
	//    Pattern: Dots on both sides ensure we only match DNS, not arbitrary strings
	//    Handles ALL K8s DNS patterns: .svc, .svc.cluster.local, .pod, .endpoints, etc.
	//    Example: `service.project-system.svc` becomes `service.{{ .Release.Namespace }}.svc`
	//    Example: `pod.project-system.pod.cluster.local` becomes `pod.{{ .Release.Namespace }}.pod.cluster.local`
	//
	//    SAFETY: This won't match:
	//    - Resource names: "users" (no dots around it)
	//    - Arbitrary strings: "my-application" (no dots)
	//    - Labels: "app=project-system" (no dots on both sides)
	dnsPattern := regexp.MustCompile(`\.` + regexp.QuoteMeta(managerNamespace) + `\.`)
	yamlContent = dnsPattern.ReplaceAllString(yamlContent, "."+namespaceTemplate+".")

	// 4. CERTIFICATE-SPECIFIC: Additional service name templating for cert-manager
	//    This is additive only and doesn't interfere with the above replacements
	if resource.GetKind() == kindCertificate {
		yamlContent = t.substituteCertificateDNSNames(yamlContent, resource)
	}

	return yamlContent
}

// substituteCertificateDNSNames replaces hardcoded DNS names in certificates with proper service templates
func (t *HelmTemplater) substituteCertificateDNSNames(yamlContent string, resource *unstructured.Unstructured) string {
	name := resource.GetName()

	// Replace service names with templated ones based on certificate type
	if strings.Contains(name, "metrics-cert") || strings.Contains(name, "metrics") {
		// Metrics certificates should point to metrics service
		// Use chart-specific resourceName helper for consistent naming with 63-char safety
		metricsServiceTemplate := "{{ include \"" + t.chartName + ".resourceName\" " +
			"(dict \"suffix\" \"controller-manager-metrics-service\" \"context\" $) }}"
		metricsServiceFQDN := metricsServiceTemplate + ".{{ include \"" + t.chartName + ".namespaceName\" $ }}.svc"
		metricsServiceFQDNCluster := metricsServiceTemplate +
			".{{ include \"" + t.chartName + ".namespaceName\" $ }}.svc.cluster.local"

		// Replace placeholders
		yamlContent = strings.ReplaceAll(yamlContent, "SERVICE_NAME.SERVICE_NAMESPACE.svc", metricsServiceFQDN)
		yamlContent = strings.ReplaceAll(yamlContent,
			"SERVICE_NAME.SERVICE_NAMESPACE.svc.cluster.local", metricsServiceFQDNCluster)

		// Also replace hardcoded service names
		hardcodedMetricsService := t.detectedPrefix + "-controller-manager-metrics-service"
		yamlContent = strings.ReplaceAll(yamlContent, hardcodedMetricsService, metricsServiceTemplate)
	} else if strings.Contains(name, "serving-cert") || strings.Contains(name, "webhook") {
		hardcodedWebhookServiceShort := t.detectedPrefix + "-webhook-service"
		yamlContent = strings.ReplaceAll(yamlContent, hardcodedWebhookServiceShort, t.resourceNameTemplate("webhook-service"))
	}

	return yamlContent
}

// substituteCertManagerReferences applies cert-manager specific template substitutions
func (t *HelmTemplater) substituteCertManagerReferences(
	yamlContent string,
	resource *unstructured.Unstructured,
) string {
	kind := resource.GetKind()

	if kind == kindIssuer || kind == kindCertificate {
		hardcodedIssuerRef := t.detectedPrefix + "-selfsigned-issuer"
		yamlContent = strings.ReplaceAll(yamlContent, hardcodedIssuerRef, t.resourceNameTemplate("selfsigned-issuer"))
	}

	if kind == kindValidatingWebhook || kind == kindMutatingWebhook || kind == kindCRD {
		hardcodedService := "name: " + t.detectedPrefix + "-webhook-service"
		templatedService := "name: " + t.resourceNameTemplate("webhook-service")
		yamlContent = strings.ReplaceAll(yamlContent, hardcodedService, templatedService)
	}

	yamlContent = t.substituteCertManagerAnnotations(yamlContent)
	return yamlContent
}

// substituteResourceNamesWithPrefix templates ALL resource names using the chart.resourceName helper.
// Generic regex-based approach works for any resource type without hardcoding specific names.
// Excludes container names and ServiceAccount metadata.name (handled separately).
//
//nolint:goconst
func (t *HelmTemplater) substituteResourceNamesWithPrefix(
	yamlContent string, resource *unstructured.Unstructured,
) string {
	namePattern := regexp.MustCompile(
		`(\s+)([a-zA-Z]*[Nn]ame):\s+` + regexp.QuoteMeta(t.detectedPrefix) + `(-[a-zA-Z0-9-]+)`)

	lines := strings.Split(yamlContent, "\n")
	result := make([]string, 0, len(lines))

	isServiceAccount := resource.GetKind() == kindServiceAccount

	// Extract actual container names from the structured object so
	// they can be skipped without backward text-scanning.
	containerNameSet := extractContainerNames(resource)

	for i, line := range lines {
		if !namePattern.MatchString(line) {
			result = append(result, line)
			continue
		}

		isContainerName := false
		if len(containerNameSet) > 0 && strings.Contains(line, "name:") {
			parts := namePattern.FindStringSubmatch(line)
			if len(parts) >= 4 {
				candidateName := t.detectedPrefix + parts[3]
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
				if prevIndent == currentIndent-2 && trimmed == "metadata:" {
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

				return indent + fieldName + ": " + t.resourceNameTemplate(suffix)
			})
			result = append(result, templatedLine)
		}
	}

	return strings.Join(result, "\n")
}

// addHelmLabelsAndAnnotations replaces kustomize managed-by labels with Helm equivalents
func (t *HelmTemplater) addHelmLabelsAndAnnotations(
	yamlContent string,
	resource *unstructured.Unstructured,
) string {
	// Replace app.kubernetes.io/managed-by: kustomize with Helm template
	// Use regex to handle different whitespace patterns
	managedByRegex := regexp.MustCompile(`(\s*)app\.kubernetes\.io/managed-by:\s+kustomize`)
	yamlContent = managedByRegex.ReplaceAllString(yamlContent, "${1}app.kubernetes.io/managed-by: {{ .Release.Service }}")

	hardcodedNameLabel := "app.kubernetes.io/name: " + t.detectedPrefix
	templatedNameLabel := "app.kubernetes.io/name: {{ include \"" + t.chartName + ".name\" . }}"
	yamlContent = strings.ReplaceAll(yamlContent, hardcodedNameLabel, templatedNameLabel)

	// Add standard Helm labels to metadata.labels and selectors
	yamlContent = t.addStandardHelmLabels(yamlContent, resource)

	return yamlContent
}

// checkExistingLabels checks if standard Helm labels already exist in a labels section
// by looking both backward and forward from the current position
func checkExistingLabels(lines []string, currentIndex int, indent string) (hasChart, hasInstance, hasManagedBy bool) {
	// Look backward from current position (managed-by often appears before name in kustomize output)
	for j := currentIndex - 1; j >= 0 && j >= currentIndex-10; j-- {
		backLine := lines[j]
		backTrimmed := strings.TrimSpace(backLine)
		backIndent, _ := leadingWhitespace(backLine)

		// Stop if we've moved out of the labels section
		if backTrimmed == yamlKeyLabels {
			break
		}
		if backTrimmed != "" && len(backIndent) < len(indent) {
			break
		}

		if strings.Contains(backLine, labelKeyHelmChart) {
			hasChart = true
		}
		if strings.Contains(backLine, labelKeyAppInstance) {
			hasInstance = true
		}
		if strings.Contains(backLine, labelKeyAppManagedBy) {
			hasManagedBy = true
		}
	}

	// Look ahead from current position
	for j := currentIndex + 1; j < len(lines) && j < currentIndex+10; j++ {
		nextLine := lines[j]
		nextTrimmed := strings.TrimSpace(nextLine)
		nextIndent, _ := leadingWhitespace(nextLine)

		// Stop if we've moved to a new section
		if nextTrimmed != "" && len(nextIndent) < len(indent) {
			break
		}

		if strings.Contains(nextLine, labelKeyHelmChart) {
			hasChart = true
		}
		if strings.Contains(nextLine, labelKeyAppInstance) {
			hasInstance = true
		}
		if strings.Contains(nextLine, labelKeyAppManagedBy) {
			hasManagedBy = true
		}
	}

	return hasChart, hasInstance, hasManagedBy
}

// addStandardHelmLabels adds standard Helm labels (helm.sh/chart, app.kubernetes.io/instance,
// and app.kubernetes.io/managed-by) to all labels sections except selectors (which must be immutable)
//

func (t *HelmTemplater) addStandardHelmLabels(yamlContent string, _ *unstructured.Unstructured) string {
	lines := strings.Split(yamlContent, "\n")
	result := make([]string, 0, len(lines)+10) // Pre-allocate with extra space for added labels
	inSelector := false

	for i := range lines {
		line := lines[i]
		result = append(result, line)

		// Track if we're in a selector section (matchLabels or spec.selector for Services)
		trimmed := strings.TrimSpace(line)
		isMatchLabels := trimmed == "matchLabels:"
		isSelectorWithoutMatchLabels := trimmed == "selector:" && i+1 < len(lines) &&
			!strings.Contains(lines[i+1], "matchLabels")
		if isMatchLabels || isSelectorWithoutMatchLabels {
			inSelector = true
		}

		// Exit selector section when we hit a line with less indentation
		if inSelector && trimmed != "" && !strings.HasPrefix(trimmed, "app.kubernetes.io/") &&
			!strings.HasPrefix(trimmed, "control-plane:") && strings.Contains(trimmed, ":") {
			inSelector = false
		}

		// Add standard Helm labels to any labels section (metadata.labels, template.metadata.labels)
		// but NOT to selectors (which must remain immutable)
		if !inSelector && strings.Contains(line, labelKeyAppName) {
			indent, _ := leadingWhitespace(line)

			// Check if we're in a labels section by looking backwards
			isInLabelsSection := false
			for j := i - 1; j >= 0 && j >= i-5; j-- {
				if strings.TrimSpace(lines[j]) == yamlKeyLabels {
					isInLabelsSection = true
					break
				}
				if strings.TrimSpace(lines[j]) == "metadata:" {
					break
				}
			}

			if !isInLabelsSection {
				continue
			}

			// Check if standard labels already exist in this labels section
			hasHelmChart, hasInstance, hasManagedBy := checkExistingLabels(lines, i, indent)

			// Add helm.sh/chart if it doesn't exist
			if !hasHelmChart {
				result = append(result, indent+labelKeyHelmChart+" {{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}")
			}

			// Add app.kubernetes.io/instance if it doesn't exist
			if !hasInstance {
				result = append(result, indent+labelKeyAppInstance+" {{ .Release.Name }}")
			}

			// Add app.kubernetes.io/managed-by if it doesn't exist (per Helm best practices)
			if !hasManagedBy {
				result = append(result, indent+labelKeyAppManagedBy+" {{ .Release.Service }}")
			}
		}
	}

	return strings.Join(result, "\n")
}

// substituteRBACValues applies RBAC-specific template substitutions
func (t *HelmTemplater) substituteRBACValues(yamlContent string) string {
	roleRefBlockPattern := regexp.MustCompile(
		`(?s)(roleRef:\s*\n(?:\s+\w+:.*\n)*?)(\s+)name:\s+` +
			regexp.QuoteMeta(t.detectedPrefix) + `-manager-role`)
	yamlContent = roleRefBlockPattern.ReplaceAllString(
		yamlContent, `${1}${2}name: `+t.resourceNameTemplate("manager-role"))

	roleRefBlockPatternSimple := regexp.MustCompile(
		`(?s)(roleRef:\s*\n(?:\s+\w+:.*\n)*?)(\s+)name:\s+manager-role`)
	yamlContent = roleRefBlockPatternSimple.ReplaceAllString(
		yamlContent, `${1}${2}name: `+t.resourceNameTemplate("manager-role"))

	yamlContent = t.templateServiceAccountNameInBindings(yamlContent)

	return yamlContent
}

// templateServiceAccountNameInBindings templates SA name in RoleBinding/ClusterRoleBinding subjects
func (t *HelmTemplater) templateServiceAccountNameInBindings(yamlContent string) string {
	replacement := `{{ include "` + t.chartName + `.serviceAccountName" . }}`

	// Handle already-templated resourceName (from substituteResourceNamesWithPrefix)
	templatedPattern := regexp.MustCompile(
		`(?m)(subjects:\s*\n\s*-\s*kind:\s*ServiceAccount\s*\n\s+name:\s+)` +
			regexp.QuoteMeta(`{{ include "`+t.chartName+`.resourceName" (dict "suffix" "controller-manager" "context" $) }}`))
	yamlContent = templatedPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names with prefix
	subjectPattern := regexp.MustCompile(
		`(?m)(subjects:\s*\n\s*-\s*kind:\s*ServiceAccount\s*\n\s+name:\s+)` +
			regexp.QuoteMeta(t.detectedPrefix) + `-controller-manager`)
	yamlContent = subjectPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names without prefix
	subjectPatternSimple := regexp.MustCompile(
		`(?m)(subjects:\s*\n\s*-\s*kind:\s*ServiceAccount\s*\n\s+name:\s+)controller-manager`)
	yamlContent = subjectPatternSimple.ReplaceAllString(yamlContent, `${1}`+replacement)

	return yamlContent
}

// templateServiceAccountNameInDeployment templates serviceAccountName in Deployment spec
func (t *HelmTemplater) templateServiceAccountNameInDeployment(yamlContent string) string {
	replacement := `serviceAccountName: {{ include "` + t.chartName + `.serviceAccountName" . }}`

	// Handle already-templated resourceName (from substituteResourceNamesWithPrefix)
	templatedPattern := regexp.MustCompile(
		`(?m)^(\s*)serviceAccountName:\s+` +
			regexp.QuoteMeta(`{{ include "`+t.chartName+`.resourceName" (dict "suffix" "controller-manager" "context" $) }}`))
	yamlContent = templatedPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names with prefix
	serviceAccountPattern := regexp.MustCompile(
		`(?m)^(\s*)serviceAccountName:\s+` + regexp.QuoteMeta(t.detectedPrefix) + `-controller-manager\s*$`)
	yamlContent = serviceAccountPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names without prefix
	serviceAccountPatternSimple := regexp.MustCompile(
		`(?m)^(\s*)serviceAccountName:\s+controller-manager\s*$`)
	yamlContent = serviceAccountPatternSimple.ReplaceAllString(yamlContent, `${1}`+replacement)

	return yamlContent
}

// substituteCertManagerAnnotations replaces hardcoded certificate references in annotations
func (t *HelmTemplater) substituteCertManagerAnnotations(yamlContent string) string {
	hardcodedServingCert := t.detectedPrefix + "-serving-cert"
	yamlContent = strings.ReplaceAll(yamlContent, hardcodedServingCert, t.resourceNameTemplate("serving-cert"))

	hardcodedMetricsCert := t.detectedPrefix + "-metrics-certs"
	yamlContent = strings.ReplaceAll(yamlContent, hardcodedMetricsCert, t.resourceNameTemplate("metrics-certs"))

	return yamlContent
}

// templateDeploymentFields converts deployment-specific fields to Helm templates
func (t *HelmTemplater) templateDeploymentFields(yamlContent string) string {
	// Template replicas from values.yaml (manager.replicas)
	yamlContent = t.templateReplicas(yamlContent)
	// Template configuration fields
	yamlContent = t.templateImageReference(yamlContent)
	yamlContent = t.templateServiceAccountNameInDeployment(yamlContent)
	yamlContent = t.templateEnvironmentVariables(yamlContent)
	yamlContent = t.templateImagePullSecrets(yamlContent)
	yamlContent = t.templatePodSecurityContext(yamlContent)
	yamlContent = t.templateContainerSecurityContext(yamlContent)
	yamlContent = t.templateResources(yamlContent)
	yamlContent = t.templateSecurityContexts(yamlContent)
	yamlContent = t.templateVolumeMounts(yamlContent)
	yamlContent = t.templateVolumes(yamlContent)
	yamlContent = t.templateControllerManagerArgs(yamlContent)
	yamlContent = t.templateBasicWithStatement(
		yamlContent,
		"nodeSelector",
		"spec.template.spec",
		".Values.manager.nodeSelector",
	)
	yamlContent = t.templateBasicWithStatement(
		yamlContent,
		"affinity",
		"spec.template.spec",
		".Values.manager.affinity",
	)
	yamlContent = t.templateBasicWithStatement(
		yamlContent,
		"tolerations",
		"spec.template.spec",
		".Values.manager.tolerations",
	)

	// Optional Kubernetes features: deployment strategy and pod scheduling
	// Template conditionals are always created (even if field doesn't exist in kustomize)
	// so users can uncomment them in values.yaml without regenerating the chart
	yamlContent = t.templateBasicWithStatement(
		yamlContent,
		"strategy",
		"spec",
		".Values.manager.strategy",
	)
	yamlContent = t.templatePriorityClassName(yamlContent)
	yamlContent = t.templateBasicWithStatement(
		yamlContent,
		"topologySpreadConstraints",
		"spec.template.spec",
		".Values.manager.topologySpreadConstraints",
	)
	yamlContent = t.templateTerminationGracePeriodSeconds(yamlContent)

	return yamlContent
}

// templateReplicas replaces deployment spec.replicas with .Values.manager.replicas so the
// value in values.yaml is used. With leader election, only one replica is active at a time;
// multiple replicas are valid for HA (standby replicas).
func (t *HelmTemplater) templateReplicas(yamlContent string) string {
	if strings.Contains(yamlContent, ".Values.manager.replicas") {
		return yamlContent
	}
	// Match a line that is exactly "  replicas: <digits>" (deployment spec.replicas).
	// Preserve indentation so the replacement fits the existing YAML structure.
	replicasPattern := regexp.MustCompile(`(?m)^(\s*)replicas:\s*\d+\s*$`)
	return replicasPattern.ReplaceAllString(yamlContent, "${1}replicas: {{ .Values.manager.replicas }}")
}

// templateEnvironmentVariables exposes environment variables via values.yaml
func (t *HelmTemplater) templateEnvironmentVariables(yamlContent string) string {
	containerName := t.getDefaultContainerName(yamlContent)
	// Check for both literal container name and templated container name
	hasLiteralName := strings.Contains(yamlContent, "name: "+containerName)
	hasTemplatedName := strings.Contains(yamlContent, `name: {{ include "`) && strings.Contains(yamlContent, `"manager"`)
	if !hasLiteralName && !hasTemplatedName {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "env:" {
			continue
		}

		indentStr, indentLen := leadingWhitespace(lines[i])
		end := i + 1
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent < indentLen {
				break
			}
			if lineIndent == indentLen && !strings.HasPrefix(trimmed, "-") {
				break
			}
		}

		nextLine := ""
		if i+1 < len(lines) {
			nextLine = lines[i+1]
		}
		if strings.Contains(nextLine, ".Values.manager.env") || strings.Contains(nextLine, "envOverrides") {
			return yamlContent
		}

		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))
		// Env list + envOverrides (CLI --set). Secret refs go in env list.
		hasEnv := `{{- if or .Values.manager.env (and (kindIs "map" .Values.manager.envOverrides) ` +
			`(not (empty .Values.manager.envOverrides))) }}`
		block := make([]string, 0, 22)
		block = append(block,
			indentStr+"env:",
			hasEnv,
			childIndent+`{{- if .Values.manager.env }}`,
			childIndent+"{{- toYaml .Values.manager.env | nindent "+childIndentWidth+" }}",
			childIndent+`{{- end }}`,
			childIndent+`{{- if kindIs "map" .Values.manager.envOverrides }}`,
			childIndent+`{{- range $k, $v := .Values.manager.envOverrides }}`,
			childIndent+`- name: {{ $k }}`,
			childIndent+`  value: {{ $v | quote }}`,
			childIndent+`{{ end }}`,
			childIndent+`{{- end }}`,
			childIndent+`{{- else }}`,
			childIndent+"[]",
			childIndent+`{{- end }}`,
		)

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

// templateResources converts resource sections to Helm templates
func (t *HelmTemplater) templateResources(yamlContent string) string {
	containerName := t.getDefaultContainerName(yamlContent)
	// Check for both literal container name and templated container name
	hasLiteralName := strings.Contains(yamlContent, "name: "+containerName)
	hasTemplatedName := strings.Contains(yamlContent, `name: {{ include "`) && strings.Contains(yamlContent, `"manager"`)
	if (!hasLiteralName && !hasTemplatedName) || !strings.Contains(yamlContent, "resources:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "resources:" {
			continue
		}

		indentStr, indentLen := leadingWhitespace(lines[i])
		end := i + 1
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent < indentLen {
				break
			}
			// stop at same-level keys that are not part of the resources mapping
			if lineIndent == indentLen && !strings.Contains(trimmed, ":") {
				break
			}
			if lineIndent == indentLen && strings.HasSuffix(trimmed, ":") {
				break
			}
		}

		if i+1 < len(lines) && strings.Contains(lines[i+1], ".Values.manager.resources") {
			return yamlContent
		}

		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))

		block := []string{
			indentStr + "resources:",
			childIndent + "{{- if .Values.manager.resources }}",
			childIndent + "{{- toYaml .Values.manager.resources | nindent " + childIndentWidth + " }}",
			childIndent + "{{- else }}",
			childIndent + "{}",
			childIndent + "{{- end }}",
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

// templateSecurityContexts preserves security contexts from kustomize output
func (t *HelmTemplater) templateSecurityContexts(yamlContent string) string {
	// Security contexts are preserved as-is from the kustomize output to maintain
	// the exact security configuration without interfering with other container fields
	return yamlContent
}

// templateVolumeMounts appends .Values.manager.extraVolumeMounts. Webhook and metrics
// mounts are conditional (makeWebhookVolumeMountsConditional, makeMetricsVolumeMountsConditional).
func (t *HelmTemplater) templateVolumeMounts(yamlContent string) string {
	return t.appendToListFromValues(yamlContent, "volumeMounts:", ".Values.manager.extraVolumeMounts")
}

// templateVolumes appends .Values.manager.extraVolumes. Webhook and metrics volumes
// are conditional (makeWebhookVolumesConditional, makeMetricsVolumesConditional).
func (t *HelmTemplater) templateVolumes(yamlContent string) string {
	return t.appendToListFromValues(yamlContent, "volumes:", ".Values.manager.extraVolumes")
}

// appendToListFromValues finds "key:" or "key: []", and either appends values path to an existing list
// or replaces "key: []" with a template that outputs the values path when set. Idempotent if already present.
func (t *HelmTemplater) appendToListFromValues(yamlContent string, keyColon string, valuesPath string) string {
	if !strings.Contains(yamlContent, keyColon) {
		return yamlContent
	}
	if strings.Contains(yamlContent, valuesPath) {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	keyEmpty := keyColon + " []"

	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		indentStr, indentLen := leadingWhitespace(lines[i])
		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))

		if trimmed == keyEmpty {
			block := []string{
				indentStr + keyColon,
				childIndent + "{{- if " + valuesPath + " }}",
				childIndent + "{{- toYaml " + valuesPath + " | nindent " + childIndentWidth + " }}",
				childIndent + "{{- else }}",
				childIndent + "[]",
				childIndent + "{{- end }}",
			}
			newLines := append([]string{}, lines[:i]...)
			newLines = append(newLines, block...)
			newLines = append(newLines, lines[i+1:]...)
			return strings.Join(newLines, "\n")
		}

		if trimmed != keyColon {
			continue
		}

		end := i + 1
		for ; end < len(lines); end++ {
			tLine := strings.TrimSpace(lines[end])
			if tLine == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent <= indentLen {
				break
			}
		}

		block := []string{
			childIndent + "{{- if " + valuesPath + " }}",
			childIndent + "{{- toYaml " + valuesPath + " | nindent " + childIndentWidth + " }}",
			childIndent + "{{- end }}",
		}
		newLines := append([]string{}, lines[:end]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}
	return yamlContent
}

// templateImagePullSecrets exposes imagePullSecrets via values.yaml.
// This is an optional Kubernetes deployment field that affects registry authentication but not operator logic.
// Always injects a conditional template (even when field is missing from kustomize) so users can
// uncomment it in values.yaml without regenerating the chart.
// Handles list format with special logic to include all list items.
func (t *HelmTemplater) templateImagePullSecrets(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")

	// Check if field already exists
	if strings.Contains(yamlContent, "imagePullSecrets:") {
		for i := range lines {
			// Use prefix to allow `imagePullSecrets: []` to be preserved
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), "imagePullSecrets:") {
				continue
			}

			// Avoid duplicate templating
			if i+1 < len(lines) && strings.Contains(lines[i+1], ".Values.manager.imagePullSecrets") {
				return yamlContent
			}

			indentStr, indentLen := leadingWhitespace(lines[i])
			end := i + 1
			// Find end of imagePullSecrets block (including list items)
			for ; end < len(lines); end++ {
				trimmed := strings.TrimSpace(lines[end])
				if trimmed == "" {
					break
				}
				lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
				if lineIndent < indentLen {
					break
				}
				// Continue if same indent and starts with "-" (list item)
				if lineIndent == indentLen && !strings.HasPrefix(trimmed, "-") {
					break
				}
			}

			childIndent := indentStr + "  "
			childIndentWidth := strconv.Itoa(len(childIndent))

			block := []string{
				indentStr + "{{- with .Values.manager.imagePullSecrets }}",
				indentStr + "imagePullSecrets:",
				childIndent + "{{- toYaml . | nindent " + childIndentWidth + " }}",
				indentStr + "{{- end }}",
			}

			newLines := append([]string{}, lines[:i]...)
			newLines = append(newLines, block...)
			newLines = append(newLines, lines[end:]...)
			return strings.Join(newLines, "\n")
		}
	}

	// Field doesn't exist - inject it after finding parent block (spec.template.spec)
	// Look for "spec:" (pod spec) under template
	var insertAt int
	foundTemplate := false
	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "template:" { //nolint:goconst
			foundTemplate = true
			continue
		}
		if foundTemplate && trimmed == "spec:" { //nolint:goconst
			// Found pod spec, inject at next line
			insertAt = i + 1
			break
		}
	}

	if insertAt == 0 || insertAt >= len(lines) {
		// Couldn't find injection point
		return yamlContent
	}

	// Get indentation from the line after spec:
	_, indentLen := leadingWhitespace(lines[insertAt])
	indentStr := strings.Repeat(" ", indentLen)
	childIndent := indentStr + "  "
	childIndentWidth := strconv.Itoa(len(childIndent))

	// Create conditional block
	block := []string{
		indentStr + "{{- with .Values.manager.imagePullSecrets }}",
		indentStr + "imagePullSecrets:",
		childIndent + "{{- toYaml . | nindent " + childIndentWidth + " }}",
		indentStr + "{{- end }}",
	}

	// Insert block
	newLines := append([]string{}, lines[:insertAt]...)
	newLines = append(newLines, block...)
	newLines = append(newLines, lines[insertAt:]...)
	return strings.Join(newLines, "\n")
}

// templatePodSecurityContext exposes podSecurityContext via values.yaml
func (t *HelmTemplater) templatePodSecurityContext(yamlContent string) string {
	if !strings.Contains(yamlContent, "securityContext:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "securityContext:" {
			continue
		}

		indentStr, indentLen := leadingWhitespace(lines[i])
		end := i + 1
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent <= indentLen {
				break
			}
		}

		if end >= len(lines) {
			break
		}

		if !strings.HasPrefix(strings.TrimSpace(lines[end]), "serviceAccountName:") {
			continue
		}

		if i+1 < len(lines) && strings.Contains(lines[i+1], ".Values.manager.podSecurityContext") {
			return yamlContent
		}

		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))

		block := []string{
			indentStr + "securityContext:",
			childIndent + "{{- if .Values.manager.podSecurityContext }}",
			childIndent + "{{- toYaml .Values.manager.podSecurityContext | nindent " + childIndentWidth + " }}",
			childIndent + "{{- else }}",
			childIndent + "{}",
			childIndent + "{{- end }}",
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

// templateContainerSecurityContext exposes container securityContext via values.yaml
func (t *HelmTemplater) templateContainerSecurityContext(yamlContent string) string {
	containerName := t.getDefaultContainerName(yamlContent)
	// Check for both literal container name and templated container name
	hasLiteralName := strings.Contains(yamlContent, "name: "+containerName)
	hasTemplatedName := strings.Contains(yamlContent, `name: {{ include "`) && strings.Contains(yamlContent, `"manager"`)
	if (!hasLiteralName && !hasTemplatedName) || !strings.Contains(yamlContent, "securityContext:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "securityContext:" {
			continue
		}

		indentStr, indentLen := leadingWhitespace(lines[i])
		end := i + 1
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent <= indentLen {
				break
			}
		}

		if end >= len(lines) {
			break
		}

		if strings.HasPrefix(strings.TrimSpace(lines[end]), "serviceAccountName:") {
			continue
		}

		lookAheadEnd := min(end+5, len(lines))
		joined := strings.Join(lines[i:lookAheadEnd], "\n")
		if strings.Contains(joined, ".Values.manager.securityContext") {
			return yamlContent
		}

		childIndent := indentStr + "  "
		childIndentWidth := strconv.Itoa(len(childIndent))

		block := []string{
			indentStr + "securityContext:",
			childIndent + "{{- if .Values.manager.securityContext }}",
			childIndent + "{{- toYaml .Values.manager.securityContext | nindent " + childIndentWidth + " }}",
			childIndent + "{{- else }}",
			childIndent + "{}",
			childIndent + "{{- end }}",
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, block...)
		newLines = append(newLines, lines[end:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

func leadingWhitespace(line string) (string, int) {
	trimmed := strings.TrimLeft(line, " \t")
	indentLen := len(line) - len(trimmed)
	return line[:indentLen], indentLen
}

// extractContainerNames returns the set of container and initContainer names declared in a
// Deployment (or any Pod-template-bearing resource).
func extractContainerNames(resource *unstructured.Unstructured) map[string]bool {
	names := map[string]bool{}
	for _, fieldPath := range [][]string{
		{"spec", "template", "spec", "containers"},
		{"spec", "template", "spec", "initContainers"},
	} {
		val, found, err := unstructured.NestedFieldNoCopy(resource.Object, fieldPath...)
		if err != nil || !found {
			continue
		}
		containers, ok := val.([]any)
		if !ok {
			continue
		}
		for _, c := range containers {
			container, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if n, ok := container["name"].(string); ok && n != "" {
				names[n] = true
			}
		}
	}
	return names
}

// isManagerDeployment checks if a Deployment is the controller manager.
// It returns true if either the deployment name contains "controller-manager"
// OR the deployment has the label "control-plane: controller-manager".
func isManagerDeployment(resource *unstructured.Unstructured) bool {
	name := resource.GetName()
	labels := resource.GetLabels()
	return strings.Contains(name, "controller-manager") ||
		(labels != nil && labels["control-plane"] == "controller-manager")
}

// templateControllerManagerArgs exposes controller manager args via values.yaml while keeping core defaults
func (t *HelmTemplater) templateControllerManagerArgs(yamlContent string) string {
	containerName := t.getDefaultContainerName(yamlContent)
	// Check for both literal container name and templated container name
	hasLiteralName := strings.Contains(yamlContent, "name: "+containerName)
	hasTemplatedName := strings.Contains(yamlContent, `name: {{ include "`) && strings.Contains(yamlContent, `"manager"`)
	if !hasLiteralName && !hasTemplatedName {
		return yamlContent
	}

	argsPattern := regexp.MustCompile(`(?m)([ \t]+)args:\n((?:[ \t]+-.*\n)+)`)
	loc := argsPattern.FindStringSubmatchIndex(yamlContent)
	if loc == nil {
		return yamlContent
	}

	match := yamlContent[loc[0]:loc[1]]
	if strings.Contains(match, ".Values.manager.args") {
		return yamlContent
	}

	indent := yamlContent[loc[2]:loc[3]]
	itemsBlock := yamlContent[loc[4]:loc[5]]

	itemIndent := indent + "  "
	lines := strings.Split(itemsBlock, "\n")
	var (
		metricsLine    string
		metricsIndent  string
		healthLine     string
		preservedLines []string
	)

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if itemIndent == indent+"  " {
			if idx := strings.Index(line, "-"); idx > 0 {
				itemIndent = line[:idx]
			}
		}

		switch {
		case strings.Contains(trimmed, "--metrics-bind-address"):
			metricsLine = line
			if idx := strings.Index(line, "-"); idx > 0 {
				metricsIndent = line[:idx]
			}
		case strings.Contains(trimmed, "--health-probe-bind-address"):
			healthLine = line
		case strings.Contains(trimmed, "--webhook-cert-path"),
			strings.Contains(trimmed, "--metrics-cert-path"):
			preservedLines = append(preservedLines, line)
		default:
			// Remaining args will be handled through values.yaml
		}
	}

	var builder strings.Builder
	builder.WriteString(indent)
	builder.WriteString("args:\n")

	if metricsLine != "" {
		if metricsIndent == "" {
			metricsIndent = itemIndent
		}
		builder.WriteString(metricsIndent)
		builder.WriteString("{{- if .Values.metrics.enable }}\n")
		builder.WriteString(metricsLine)
		builder.WriteString("\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("{{- if not .Values.metrics.secure }}\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("- --metrics-secure=false\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("{{- end }}\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("{{- else }}\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("# Bind to :0 to disable the controller-runtime managed metrics server\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("- --metrics-bind-address=0\n")
		builder.WriteString(metricsIndent)
		builder.WriteString("{{- end }}\n")
	}
	if healthLine != "" {
		builder.WriteString(healthLine)
		builder.WriteString("\n")
	}

	builder.WriteString(itemIndent)
	builder.WriteString("{{- range .Values.manager.args }}\n")
	builder.WriteString(itemIndent)
	builder.WriteString("- {{ . }}\n")
	builder.WriteString(itemIndent)
	builder.WriteString("{{- end }}\n")

	for _, line := range preservedLines {
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	newBlock := strings.TrimRight(builder.String(), "\n") + "\n"

	return yamlContent[:loc[0]] + newBlock + yamlContent[loc[1]:]
}

// templateImageReference converts hardcoded image references to Helm templates
func (t *HelmTemplater) templateImageReference(yamlContent string) string {
	containerName := t.getDefaultContainerName(yamlContent)
	// Check for both literal container name and templated container name (which may have been
	// converted by substituteResourceNamesWithPrefix before this function runs)
	hasLiteralName := strings.Contains(yamlContent, "name: "+containerName)
	hasTemplatedName := strings.Contains(yamlContent, `name: {{ include "`) && strings.Contains(yamlContent, `"manager"`)
	if !hasLiteralName && !hasTemplatedName {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "image:") {
			continue
		}

		if strings.Contains(lines[i], ".Values.manager.image.repository") {
			return yamlContent
		}

		indentStr, indentLen := leadingWhitespace(lines[i])

		end := i + 1
		for ; end < len(lines); end++ {
			nextTrimmed := strings.TrimSpace(lines[end])
			if nextTrimmed == "" {
				break
			}
			lineIndent := len(lines[end]) - len(strings.TrimLeft(lines[end], " \t"))
			if lineIndent <= indentLen {
				break
			}
			// Stop when we reach a sibling key like env:, args:, etc.
			if lineIndent == indentLen+2 && strings.HasSuffix(nextTrimmed, ":") {
				if strings.Contains(nextTrimmed, "imagePullPolicy") {
					continue
				}
				break
			}
		}

		// Remove any existing imagePullPolicy line inside the block
		blockLines := lines[i+1 : end]
		filtered := make([]string, 0, len(blockLines))
		for _, line := range blockLines {
			if strings.Contains(strings.TrimSpace(line), "imagePullPolicy") {
				continue
			}
			filtered = append(filtered, line)
		}
		lines = append(lines[:i+1], append(filtered, lines[end:]...)...)
		end = i + 1 + len(filtered)

		imageLine := indentStr + "image: \"{{ .Values.manager.image.repository }}:" +
			"{{ .Values.manager.image.tag | default .Chart.AppVersion }}\""
		pullPolicyLine := indentStr + "imagePullPolicy: {{ .Values.manager.image.pullPolicy }}"

		remainder := lines[end:]
		if len(remainder) > 0 && strings.HasPrefix(strings.TrimSpace(remainder[0]), "imagePullPolicy:") {
			remainder = remainder[1:]
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, imageLine, pullPolicyLine)
		newLines = append(newLines, remainder...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

func (t *HelmTemplater) templateBasicWithStatement(
	yamlContent string,
	key string,
	parentKey string,
	valuePath string,
) string {
	if strings.Contains(yamlContent, valuePath) {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	yamlKey := fmt.Sprintf("%s:", key)

	var start, end int
	var indentLen int
	if !strings.Contains(yamlContent, yamlKey) {
		// Find parent block start if the key is missing
		pKeyParts := strings.Split(parentKey, ".")
		pKeyIdx := 0
		pKeyInit := false
		currIndent := 0
		for i := range len(lines) {
			_, lineIndent := leadingWhitespace(lines[i])
			if pKeyInit && lineIndent <= currIndent {
				return yamlContent
			}
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), pKeyParts[pKeyIdx]) {
				continue
			}

			// Parent key part found
			pKeyIdx++
			pKeyInit = true
			if pKeyIdx >= len(pKeyParts) {
				start = i + 1
				end = start
				break
			}
		}
		_, indentLen = leadingWhitespace(lines[start])
	} else {
		// Find the existing block - stop at the first match.
		for i := range len(lines) {
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), yamlKey) {
				continue
			}
			start = i
			end = i + 1
			trimmed := strings.TrimSpace(lines[i])
			if len(trimmed) == len(yamlKey) {
				_, indentLenSearch := leadingWhitespace(lines[i])
				end = len(lines)
				for j := i + 1; j < len(lines); j++ {
					trimmedJ := strings.TrimSpace(lines[j])
					_, indentLenLine := leadingWhitespace(lines[j])
					if indentLenLine < indentLenSearch {
						end = j
						break
					}
					if indentLenLine == indentLenSearch && !strings.HasPrefix(trimmedJ, "- ") {
						end = j
						break
					}
				}
			}
			break // use the first match only
		}
		_, indentLen = leadingWhitespace(lines[start])
	}

	indentStr := strings.Repeat(" ", indentLen)

	var builder strings.Builder
	builder.WriteString(indentStr)
	builder.WriteString("{{- with ")
	builder.WriteString(valuePath)
	builder.WriteString(" }}\n")
	builder.WriteString(indentStr)
	builder.WriteString(yamlKey)
	builder.WriteString(" {{ toYaml . | nindent ")
	builder.WriteString(strconv.Itoa(indentLen + 4))
	builder.WriteString(" }}\n")
	builder.WriteString(indentStr)
	builder.WriteString("{{- end }}\n")

	newBlock := strings.TrimRight(builder.String(), "\n")

	newLines := append([]string{}, lines[:start]...)
	newLines = append(newLines, strings.Split(newBlock, "\n")...)
	newLines = append(newLines, lines[end:]...)
	return strings.Join(newLines, "\n")
}

// makeWebhookAnnotationsConditional makes only cert-manager annotations conditional, not the entire webhook
func (t *HelmTemplater) makeWebhookAnnotationsConditional(yamlContent string) string {
	// Find cert-manager.io/inject-ca-from annotation and make it conditional
	if strings.Contains(yamlContent, "cert-manager.io/inject-ca-from") {
		// Replace the cert-manager annotation with conditional wrapper
		certManagerPattern := regexp.MustCompile(`(\s+)cert-manager\.io/inject-ca-from:\s*[^\n]+`)
		yamlContent = certManagerPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			// Extract the indentation
			indentMatch := regexp.MustCompile(`^(\s+)`).FindStringSubmatch(match)
			indent := ""
			if len(indentMatch) > 1 {
				indent = indentMatch[1]
			}

			// Extract the annotation line with proper indentation
			annotationLine := strings.TrimSpace(match)

			return fmt.Sprintf("%s{{- if .Values.certManager.enable }}\n%s%s\n%s{{- end }}",
				indent, indent, annotationLine, indent)
		})
	}

	return yamlContent
}

// makeContainerArgsConditional makes webhook-cert-path and metrics-cert-path args conditional on certManager.enable
func (t *HelmTemplater) makeContainerArgsConditional(yamlContent string) string {
	// Make webhook-cert-path arg conditional on certManager.enable
	if strings.Contains(yamlContent, "--webhook-cert-path") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		webhookArgPattern := regexp.MustCompile(`([ \t]+)-\s*--webhook-cert-path=[^\n]*`)
		yamlContent = webhookArgPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			indentMatch := regexp.MustCompile(`^(\s+)`).FindStringSubmatch(match)
			indent := ""
			if len(indentMatch) > 1 {
				indent = indentMatch[1]
			}

			argLine := strings.TrimSpace(match)
			return fmt.Sprintf("%s{{- if .Values.certManager.enable }}\n%s%s\n%s{{- end }}",
				indent, indent, argLine, indent)
		})
	}

	// Make metrics-cert-path arg conditional on certManager.enable AND metrics.enable
	if strings.Contains(yamlContent, "--metrics-cert-path") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		metricsArgPattern := regexp.MustCompile(`([ \t]+)-\s*--metrics-cert-path=[^\n]*`)
		yamlContent = metricsArgPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			indentMatch := regexp.MustCompile(`^(\s+)`).FindStringSubmatch(match)
			indent := ""
			if len(indentMatch) > 1 {
				indent = indentMatch[1]
			}

			argLine := strings.TrimSpace(match)
			return fmt.Sprintf("%s{{- if and .Values.certManager.enable .Values.metrics.enable }}\n%s%s\n%s{{- end }}",
				indent, indent, argLine, indent)
		})
	}

	return yamlContent
}

func makeYamlContent(match string) string {
	lines := strings.Split(match, "\n")
	if len(lines) > 0 {
		indent := ""
		if len(lines[0]) > 0 && lines[0][0] == ' ' {
			// Count leading spaces
			for _, char := range lines[0] {
				if char == ' ' {
					indent += " "
				} else {
					break
				}
			}
		}

		// Reconstruct the block with conditional wrapper
		var result strings.Builder
		result.WriteString(fmt.Sprintf("%s{{- if .Values.certManager.enable }}\n", indent))
		for _, line := range lines {
			result.WriteString(line + "\n")
		}
		result.WriteString(fmt.Sprintf("%s{{- end }}", indent))
		return result.String()
	}
	return match
}

// makeWebhookVolumesConditional makes webhook volumes conditional on certManager.enable
func (t *HelmTemplater) makeWebhookVolumesConditional(yamlContent string) string {
	// Make webhook volumes conditional on certManager.enable
	if strings.Contains(yamlContent, "webhook-certs") && strings.Contains(yamlContent, "secretName: webhook-server-cert") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		volumePattern := regexp.MustCompile(`([ \t]+)-\s*name:\s*webhook-certs[\s\S]*?secretName:\s*webhook-server-cert`)
		yamlContent = volumePattern.ReplaceAllStringFunc(yamlContent, makeYamlContent)
	}

	return yamlContent
}

// makeWebhookVolumeMountsConditional makes webhook volumeMounts conditional on certManager.enable
func (t *HelmTemplater) makeWebhookVolumeMountsConditional(yamlContent string) string {
	// Make webhook volumeMounts conditional on certManager.enable
	webhookCertsPath := "/tmp/k8s-webhook-server/serving-certs"
	if strings.Contains(yamlContent, "webhook-certs") && strings.Contains(yamlContent, webhookCertsPath) {
		// Match only spaces/tabs for indent to avoid consuming the newline
		mountPattern := regexp.MustCompile(
			`([ \t]+)-\s*mountPath:\s*/tmp/k8s-webhook-server/serving-certs[\s\S]*?readOnly:\s*true`)
		yamlContent = mountPattern.ReplaceAllStringFunc(yamlContent, makeYamlContent)
	}

	return yamlContent
}

// makeMetricsVolumesConditional makes metrics volumes conditional on certManager.enable AND metrics.enable
func (t *HelmTemplater) makeMetricsVolumesConditional(yamlContent string) string {
	// Make metrics volumes conditional on certManager.enable AND metrics.enable
	if strings.Contains(yamlContent, "metrics-certs") && strings.Contains(yamlContent, "secretName: metrics-server-cert") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		volumePattern := regexp.MustCompile(`([ \t]+)-\s*name:\s*metrics-certs[\s\S]*?secretName:\s*metrics-server-cert`)
		yamlContent = volumePattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			lines := strings.Split(match, "\n")
			if len(lines) > 0 {
				indent := ""
				if len(lines[0]) > 0 && lines[0][0] == ' ' {
					// Count leading spaces
					for _, char := range lines[0] {
						if char == ' ' {
							indent += " "
						} else {
							break
						}
					}
				}

				// Reconstruct the block with conditional wrapper
				var result strings.Builder
				result.WriteString(fmt.Sprintf("%s{{- if and .Values.certManager.enable .Values.metrics.enable }}\n", indent))
				for _, line := range lines {
					result.WriteString(line + "\n")
				}
				result.WriteString(fmt.Sprintf("%s{{- end }}", indent))
				return result.String()
			}
			return match
		})
	}

	return yamlContent
}

// injectCommonLabels adds a Helm template snippet to append user-provided common labels
// (.Values.commonLabels) to every metadata.labels block while preserving indentation.
// It avoids duplicate insertion by checking for an existing snippet nearby.
// no common labels injection; labels come from kustomize manifests

// makeMetricsVolumeMountsConditional makes metrics volumeMounts conditional on certManager.enable AND metrics.enable
func (t *HelmTemplater) makeMetricsVolumeMountsConditional(yamlContent string) string {
	// Make metrics volumeMounts conditional on certManager.enable AND metrics.enable
	metricsCertsPath := "/tmp/k8s-metrics-server/metrics-certs"
	if strings.Contains(yamlContent, "metrics-certs") && strings.Contains(yamlContent, metricsCertsPath) {
		// Match only spaces/tabs for indent to avoid consuming the newline
		mountPattern := regexp.MustCompile(
			`([ \t]+)-\s*mountPath:\s*/tmp/k8s-metrics-server/metrics-certs[\s\S]*?readOnly:\s*true`)
		yamlContent = mountPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
			lines := strings.Split(match, "\n")
			if len(lines) > 0 {
				indent := ""
				if len(lines[0]) > 0 && lines[0][0] == ' ' {
					// Count leading spaces
					for _, char := range lines[0] {
						if char == ' ' {
							indent += " "
						} else {
							break
						}
					}
				}

				// Reconstruct the block with conditional wrapper
				var result strings.Builder
				result.WriteString(fmt.Sprintf("%s{{- if and .Values.certManager.enable .Values.metrics.enable }}\n", indent))
				for _, line := range lines {
					result.WriteString(line + "\n")
				}
				result.WriteString(fmt.Sprintf("%s{{- end }}", indent))
				return result.String()
			}
			return match
		})
	}

	return yamlContent
}

// injectCRDResourcePolicyAnnotation adds the helm.sh/resource-policy: keep annotation
// to CRDs conditionally based on .Values.crd.keep. This prevents CRDs from being deleted
// on helm uninstall when crd.keep is true in values.yaml.
func (t *HelmTemplater) injectCRDResourcePolicyAnnotation(yamlContent string) string {
	// Check if metadata section exists
	if !strings.Contains(yamlContent, "metadata:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")

	// Check if annotations: already exists
	if strings.Contains(yamlContent, yamlKeyAnnotations) {
		// Find the annotations: line and determine its indentation
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == yamlKeyAnnotations || strings.HasPrefix(trimmed, yamlKeyAnnotations) {
				annotationsIndent, _ := leadingWhitespace(line)
				// Annotation values need one more level of indentation (2 spaces for sigs.k8s.io/yaml)
				valueIndent := annotationsIndent + "  "

				// Build the conditional annotation block
				resourcePolicyBlock := fmt.Sprintf(
					"%s{{- if .Values.crd.keep }}\n%s\"helm.sh/resource-policy\": keep\n%s{{- end }}",
					valueIndent, valueIndent, valueIndent)

				// Insert after the annotations: line
				result := make([]string, 0, len(lines)+3)
				result = append(result, lines[:i+1]...)
				result = append(result, resourcePolicyBlock)
				result = append(result, lines[i+1:]...)
				return strings.Join(result, "\n")
			}
		}
	} else {
		// No annotations section exists, need to add it after metadata:
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "metadata:" || strings.HasPrefix(trimmed, "metadata:") {
				metadataIndent, _ := leadingWhitespace(line)
				// Fields under metadata need one more level of indentation (2 spaces for sigs.k8s.io/yaml)
				fieldIndent := metadataIndent + "  "
				// Annotation values need two more levels (4 spaces total)
				valueIndent := metadataIndent + "    "

				// Build annotations section with conditional resource-policy
				annotationsSection := fmt.Sprintf(
					"%sannotations:\n%s{{- if .Values.crd.keep }}\n%s\"helm.sh/resource-policy\": keep\n%s{{- end }}",
					fieldIndent, valueIndent, valueIndent, valueIndent)

				// Insert after the metadata: line
				result := make([]string, 0, len(lines)+4)
				result = append(result, lines[:i+1]...)
				result = append(result, annotationsSection)
				result = append(result, lines[i+1:]...)
				return strings.Join(result, "\n")
			}
		}
	}

	return yamlContent
}

// addConditionalWrappers adds conditional Helm logic based on resource type
func (t *HelmTemplater) addConditionalWrappers(yamlContent string, resource *unstructured.Unstructured) string {
	kind := resource.GetKind()
	apiVersion := resource.GetAPIVersion()
	name := resource.GetName()

	switch {
	case kind == kindNamespace:
		return ""
	case kind == "CustomResourceDefinition":
		// CRDs need resource-policy annotation for helm uninstall protection
		yamlContent = t.injectCRDResourcePolicyAnnotation(yamlContent)
		// CRDs need crd.enable condition
		return fmt.Sprintf("{{- if .Values.crd.enable }}\n%s{{- end }}\n", yamlContent)
	case kind == kindCertificate && apiVersion == apiVersionCertManager:
		return t.handleCertificateConditionalWrappers(yamlContent, name)
	case kind == kindIssuer && apiVersion == apiVersionCertManager:
		// All cert-manager issuers need certManager enabled
		return fmt.Sprintf("{{- if .Values.certManager.enable }}\n%s{{- end }}", yamlContent)
	case kind == kindServiceMonitor && apiVersion == apiVersionMonitoring:
		// ServiceMonitors need prometheus enabled
		return fmt.Sprintf("{{- if .Values.prometheus.enable }}\n%s{{- end }}", yamlContent)
	case kind == kindServiceAccount, kind == kindRole, kind == kindClusterRole,
		kind == kindRoleBinding, kind == kindClusterRoleBinding:
		return t.handleRBACConditionalWrappers(yamlContent, kind, name)
	case kind == kindValidatingWebhook || kind == kindMutatingWebhook:
		// Webhook configurations should be conditional on webhook.enable
		yamlContent = t.makeWebhookAnnotationsConditional(yamlContent)
		return fmt.Sprintf("{{- if .Values.webhook.enable }}\n%s{{- end }}\n", yamlContent)
	case kind == kindService:
		return t.handleServiceConditionalWrappers(yamlContent, name)
	case kind == kindDeployment:
		// Only the manager deployment should be conditional on manager.enabled.
		// Enabled when the key is absent (backward compatibility) OR when it's true.
		// Disabled only when explicitly set to false.
		if isManagerDeployment(resource) {
			return fmt.Sprintf(
				"{{- if or (not (hasKey .Values.manager \"enabled\")) (.Values.manager.enabled) }}\n%s\n{{- end }}\n",
				yamlContent,
			)
		}
		// Other deployments don't need conditionals
		return yamlContent
	default:
		// No conditional wrapper needed for other unhandled resource kinds
		return yamlContent
	}
}

// handleCertificateConditionalWrappers handles conditional logic for Certificate resources
func (t *HelmTemplater) handleCertificateConditionalWrappers(yamlContent, name string) string {
	// Handle different certificate types using suffix matching to avoid false positives
	// when project name contains "metrics" (e.g., "metrics-operator")
	isMetricsCert := strings.HasSuffix(name, "-metrics-certs") || strings.HasSuffix(name, "-metrics-cert")
	if isMetricsCert {
		// Metrics certificates need certManager, metrics enabled, AND secure metrics (TLS)
		// When metrics.secure=false, metrics use HTTP so TLS certs are not needed
		return fmt.Sprintf(
			"{{- if and .Values.certManager.enable .Values.metrics.enable .Values.metrics.secure }}\n%s{{- end }}\n",
			yamlContent)
	}
	// Other certificates (webhook serving certs) only need certManager enabled
	return fmt.Sprintf("{{- if .Values.certManager.enable }}\n%s{{- end }}", yamlContent)
}

// handleServiceConditionalWrappers handles conditional logic for Service resources
func (t *HelmTemplater) handleServiceConditionalWrappers(yamlContent, name string) string {
	// Services need conditional logic based on their purpose.
	// Use suffix matching to avoid false positives when project name contains these substrings.
	if strings.HasSuffix(name, "-metrics-service") || strings.HasSuffix(name, "-controller-manager-metrics-service") {
		// Metrics services need metrics enabled
		return fmt.Sprintf("{{- if .Values.metrics.enable }}\n%s{{- end }}\n", yamlContent)
	}
	if strings.HasSuffix(name, "-webhook-service") {
		// Webhook services need webhook enabled
		return fmt.Sprintf("{{- if .Values.webhook.enable }}\n%s{{- end }}\n", yamlContent)
	}
	// Other services don't need conditionals
	return yamlContent
}

// handleRBACConditionalWrappers handles conditional logic for RBAC resources
func (t *HelmTemplater) handleRBACConditionalWrappers(yamlContent, kind, name string) string {
	// Distinguish between essential RBAC and helper RBAC.
	// Use suffix matching to avoid false positives when project name contains these substrings.
	// Check both roles (-admin-role, -editor-role, -viewer-role) and their bindings (-rolebinding)
	isHelper := strings.HasSuffix(name, "-admin-role") || strings.HasSuffix(name, "-editor-role") ||
		strings.HasSuffix(name, "-viewer-role") ||
		strings.HasSuffix(name, "-admin-rolebinding") || strings.HasSuffix(name, "-editor-rolebinding") ||
		strings.HasSuffix(name, "-viewer-rolebinding")

	// Check for specific Kubebuilder-scaffolded metrics RBAC resources
	isMetricsAuthRole := strings.HasSuffix(name, "-metrics-auth-role")
	isMetricsAuthBinding := strings.HasSuffix(name, "-metrics-auth-rolebinding")
	isMetricsReader := strings.HasSuffix(name, "-metrics-reader")

	// Apply kind-switching for ClusterRole/ClusterRoleBinding (except metrics-auth role/binding and metrics-reader)
	isClusterRoleKind := kind == kindClusterRole || kind == kindClusterRoleBinding
	needsKindSwitching := !isMetricsAuthRole && !isMetricsAuthBinding && !isMetricsReader
	if isClusterRoleKind && needsKindSwitching {
		// Metrics-auth-role/binding/reader must stay ClusterRole/ClusterRoleBinding (use cluster-scoped APIs/nonResourceURLs)
		yamlContent = t.makeRBACKindConditional(yamlContent, kind)
	}

	if isHelper {
		return fmt.Sprintf("{{- if .Values.rbac.helpers.enable }}\n%s{{- end }}\n", yamlContent)
	}
	if isMetricsAuthRole {
		// Only needed when secure metrics enabled (authn via TokenReview/SubjectAccessReview)
		return fmt.Sprintf("{{- if and .Values.metrics.enable .Values.metrics.secure }}\n%s{{- end }}\n", yamlContent)
	}
	if isMetricsReader {
		// Only needed when secure metrics enabled (uses nonResourceURLs for /metrics access)
		return fmt.Sprintf("{{- if and .Values.metrics.enable .Values.metrics.secure }}\n%s{{- end }}\n", yamlContent)
	}
	if isMetricsAuthBinding {
		// Binding for metrics-auth-role, only needed when secure metrics enabled
		return fmt.Sprintf("{{- if and .Values.metrics.enable .Values.metrics.secure }}\n%s{{- end }}\n", yamlContent)
	}
	// Essential RBAC (manager, leader-election) - always created
	return yamlContent
}

// makeRBACKindConditional adds conditional rendering for ClusterRole/ClusterRoleBinding
// to switch between cluster-scoped and namespace-scoped based on .Values.rbac.namespaced
func (t *HelmTemplater) makeRBACKindConditional(yamlContent string, kind string) string {
	var replacements []struct{ old, new string }

	if kind == kindClusterRole {
		// Replace kind: ClusterRole with conditional
		replacements = append(replacements, struct{ old, new string }{
			old: "kind: ClusterRole",
			new: "{{- if .Values.rbac.namespaced }}\nkind: Role\n{{- else }}\nkind: ClusterRole\n{{- end }}",
		})
		// Add namespace after metadata for Role variant
		replacements = append(replacements, struct{ old, new string }{
			old: "metadata:",
			new: "metadata:\n{{- if .Values.rbac.namespaced }}\n  namespace: {{ .Release.Namespace }}\n{{- end }}",
		})
	}

	if kind == kindClusterRoleBinding {
		// Replace kind: ClusterRoleBinding with conditional
		replacements = append(replacements, struct{ old, new string }{
			old: "kind: ClusterRoleBinding",
			new: "{{- if .Values.rbac.namespaced }}\nkind: RoleBinding\n{{- else }}\nkind: ClusterRoleBinding\n{{- end }}",
		})
		// Add namespace after metadata for RoleBinding variant
		replacements = append(replacements, struct{ old, new string }{
			old: "metadata:",
			new: "metadata:\n{{- if .Values.rbac.namespaced }}\n  namespace: {{ .Release.Namespace }}\n{{- end }}",
		})
		// Replace roleRef kind
		replacements = append(replacements, struct{ old, new string }{
			old: "  kind: ClusterRole",
			new: "  {{- if .Values.rbac.namespaced }}\n  kind: Role\n  {{- else }}\n  kind: ClusterRole\n  {{- end }}",
		})
	}

	result := yamlContent
	for _, r := range replacements {
		result = strings.Replace(result, r.old, r.new, 1)
	}

	return result
}

// collapseBlankLineAfterIf removes a single empty line that may appear
// immediately after a Helm if directive line, e.g. `{{- if ... }}`.
// This keeps templates compact and matches expected formatting in tests.
func (t *HelmTemplater) collapseBlankLineAfterIf(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	if len(lines) == 0 {
		return yamlContent
	}
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// If current line is an if, and next line is blank, skip the blank
		if strings.Contains(line, "{{- if ") {
			out = append(out, line)
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
				i++ // skip one blank line after if
			}
			continue
		}
		// If current line is blank, and next line is an end, skip the blank
		if strings.TrimSpace(line) == "" && i+1 < len(lines) && strings.Contains(lines[i+1], "{{- end }}") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// templatePriorityClassName wraps priorityClassName with conditional Helm logic.
// This is an optional Kubernetes field for pod scheduling priority that affects deployment
// behavior but not operator logic. Always injects a conditional template (even when field is
// missing from kustomize) so users can uncomment it in values.yaml without regenerating the chart.
// Uses simple inline value since priorityClassName is a string, not a YAML object.
func (t *HelmTemplater) templatePriorityClassName(yamlContent string) string {
	// Avoid duplicate templating
	if strings.Contains(yamlContent, ".Values.manager.priorityClassName") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")

	// Check if field already exists
	if strings.Contains(yamlContent, "priorityClassName:") {
		// Replace existing field with template (quote filter ensures proper YAML quoting)
		pattern := regexp.MustCompile(`(?m)^(\s*)priorityClassName:\s*"?([^"\n]*)"?\s*$`)
		yamlContent = pattern.ReplaceAllString(yamlContent,
			"${1}{{- with .Values.manager.priorityClassName }}\n"+
				"${1}priorityClassName: {{ . | quote }}\n"+
				"${1}{{- end }}")
		return yamlContent
	}

	// Field doesn't exist - inject it after finding parent block (spec.template.spec)
	// Look for "spec:" (pod spec) under template
	var insertAt int
	foundTemplate := false
	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "template:" {
			foundTemplate = true
			continue
		}
		if foundTemplate && trimmed == "spec:" {
			// Found pod spec, inject at next line
			insertAt = i + 1
			break
		}
	}

	if insertAt == 0 || insertAt >= len(lines) {
		// Couldn't find injection point
		return yamlContent
	}

	// Get indentation from the line after spec:
	_, indentLen := leadingWhitespace(lines[insertAt])
	indentStr := strings.Repeat(" ", indentLen)

	// Create conditional block (quote filter ensures proper YAML quoting)
	block := []string{
		indentStr + "{{- with .Values.manager.priorityClassName }}",
		indentStr + "priorityClassName: {{ . | quote }}",
		indentStr + "{{- end }}",
	}

	// Insert block
	newLines := append([]string{}, lines[:insertAt]...)
	newLines = append(newLines, block...)
	newLines = append(newLines, lines[insertAt:]...)
	return strings.Join(newLines, "\n")
}

// templateTerminationGracePeriodSeconds wraps terminationGracePeriodSeconds with conditional Helm logic.
// This is an optional Kubernetes field for graceful shutdown that affects deployment behavior but not operator logic.
// Always injects a conditional template (even when field is missing from kustomize) so users can
// uncomment it in values.yaml without regenerating the chart.
// Uses inline value reference since terminationGracePeriodSeconds is an integer, not a YAML object.
// Note: Uses hasKey to support 0 values (immediate termination is valid).
func (t *HelmTemplater) templateTerminationGracePeriodSeconds(yamlContent string) string {
	// Avoid duplicate templating
	if strings.Contains(yamlContent, ".Values.manager.terminationGracePeriodSeconds") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")

	// Check if field already exists
	if strings.Contains(yamlContent, "terminationGracePeriodSeconds:") {
		// Replace existing field with template (hasKey + ne nil supports 0 values while preventing <no value>)
		pattern := regexp.MustCompile(`(?m)^(\s*)terminationGracePeriodSeconds:\s*\d+\s*$`)
		yamlContent = pattern.ReplaceAllString(yamlContent,
			"${1}{{- if and (hasKey .Values.manager \"terminationGracePeriodSeconds\") "+
				"(ne .Values.manager.terminationGracePeriodSeconds nil) }}\n"+
				"${1}terminationGracePeriodSeconds: {{ .Values.manager.terminationGracePeriodSeconds }}\n"+
				"${1}{{- end }}")
		return yamlContent
	}

	// Field doesn't exist - inject it after finding serviceAccountName (typical location)
	// Look for "serviceAccountName:" in pod spec
	var insertAt int
	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "serviceAccountName:") {
			// Inject after serviceAccountName
			insertAt = i + 1
			break
		}
	}

	if insertAt == 0 || insertAt >= len(lines) {
		// Couldn't find injection point
		return yamlContent
	}

	// Get indentation from serviceAccountName line
	_, indentLen := leadingWhitespace(lines[insertAt-1])
	indentStr := strings.Repeat(" ", indentLen)

	// Create conditional block (hasKey + ne nil supports 0 values while preventing <no value>)
	block := []string{
		indentStr + "{{- if and (hasKey .Values.manager \"terminationGracePeriodSeconds\") " +
			"(ne .Values.manager.terminationGracePeriodSeconds nil) }}",
		indentStr + "terminationGracePeriodSeconds: {{ .Values.manager.terminationGracePeriodSeconds }}",
		indentStr + "{{- end }}",
	}

	// Insert block
	newLines := append([]string{}, lines[:insertAt]...)
	newLines = append(newLines, block...)
	newLines = append(newLines, lines[insertAt:]...)
	return strings.Join(newLines, "\n")
}

// templatePorts replaces hardcoded port values with Helm template references
// This makes ports configurable via values.yaml under webhook.port and metrics.port
func (t *HelmTemplater) templatePorts(yamlContent string, resource *unstructured.Unstructured) string {
	resourceName := resource.GetName()

	// Determine if this is a webhook-related resource
	isWebhook := strings.Contains(resourceName, "webhook")

	// Determine if this is a metrics-related resource
	isMetrics := strings.Contains(resourceName, "metrics")

	// For Deployments, check for webhook ports in the content
	if resource.GetKind() == kindDeployment {
		// Check if this deployment has webhook-server ports
		if strings.Contains(yamlContent, "webhook-server") || strings.Contains(yamlContent, "name: webhook") {
			isWebhook = true
		}
	}

	// Template webhook ports (9443 by default)
	if isWebhook {
		// Replace containerPort: 9443 (or any value) for webhook-server with template
		if strings.Contains(yamlContent, "webhook-server") {
			yamlContent = regexp.MustCompile(`(?m)(\s*- )?containerPort:\s*\d+(\s*\n\s*name:\s*webhook-server)`).
				ReplaceAllString(yamlContent, "${1}containerPort: {{ .Values.webhook.port }}${2}")
		}

		// Replace targetPort: 9443 with webhook.port template
		yamlContent = regexp.MustCompile(`(\s*)targetPort:\s*9443`).
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.webhook.port }}")
	}

	// Template metrics ports (8443 by default)
	if isMetrics {
		// Replace port: 8443 with metrics.port template
		yamlContent = regexp.MustCompile(`(\s*)port:\s*8443`).
			ReplaceAllString(yamlContent, "${1}port: {{ .Values.metrics.port }}")

		// Replace targetPort: 8443 with metrics.port template
		yamlContent = regexp.MustCompile(`(\s*)targetPort:\s*8443`).
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.metrics.port }}")

		// Template port name based on metrics.secure (http vs https)
		// This ensures Service and ServiceMonitor use the correct scheme
		if resource.GetKind() == kindService {
			yamlContent = regexp.MustCompile(`(\s*)- name:\s*https(\s+port:)`).
				ReplaceAllString(yamlContent, `${1}- name: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}${2}`)
		}
	}

	// Template port-related arguments in Deployment
	if resource.GetKind() == kindDeployment {
		// Replace --metrics-bind-address=:8443 with templated version
		yamlContent = regexp.MustCompile(`--metrics-bind-address=:[0-9]+`).
			ReplaceAllString(yamlContent, "--metrics-bind-address=:{{ .Values.metrics.port }}")

		// Replace --webhook-port=9443 with templated version (if present)
		yamlContent = regexp.MustCompile(`--webhook-port=[0-9]+`).
			ReplaceAllString(yamlContent, "--webhook-port={{ .Values.webhook.port }}")
	}

	return yamlContent
}

// addCustomLabelsAndAnnotations injects Helm templates for manager.labels, manager.annotations,
// manager.pod.labels, and manager.pod.annotations, with automatic duplicate key filtering.
// Each block is checked and added independently, allowing additive updates in partial/upgrade scenarios.
func (t *HelmTemplater) addCustomLabelsAndAnnotations(yamlContent string) string {
	// Check which blocks are already present to enable additive updates
	hasDeploymentLabels := strings.Contains(yamlContent, "{{- if .Values.manager.labels }}") ||
		strings.Contains(yamlContent, "{{- with .Values.manager.labels }}")
	hasDeploymentAnnotations := strings.Contains(yamlContent, "{{- if .Values.manager.annotations }}") ||
		strings.Contains(yamlContent, "{{- with .Values.manager.annotations }}")
	hasPodBlock := strings.Contains(yamlContent, "{{- with .Values.manager.pod }}")
	hasPodLabels := hasPodBlock && strings.Contains(yamlContent, "{{- with .labels }}")
	hasPodAnnotations := hasPodBlock && (strings.Contains(yamlContent, "{{- with .annotations }}") ||
		strings.Contains(yamlContent, "{{- if .Values.manager.pod.annotations }}"))

	lines := strings.Split(yamlContent, "\n")
	result := make([]string, 0, len(lines))
	state := &customFieldsState{
		position:                     positionStart,
		addedLabelsToDeployment:      hasDeploymentLabels,
		addedAnnotationsToDeployment: hasDeploymentAnnotations,
		addedPodLabels:               hasPodLabels,
		addedPodAnnotations:          hasPodAnnotations,
	}

	for i := range lines {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		indent, indentLen := leadingWhitespace(line)

		// Create missing annotations block if Deployment has none
		if state.position == positionDeploymentMetadata &&
			trimmed == "spec:" &&
			!state.addedAnnotationsToDeployment &&
			!state.hasDeploymentAnnotations {
			metadataChildIndent := strings.Repeat(" ", state.deploymentMetadataDepth) + "  "
			result = append(result, metadataChildIndent+"{{- if .Values.manager.annotations }}")
			result = append(result, metadataChildIndent+"annotations:")
			childIndent := metadataChildIndent + "  "
			childIndentWidth := strconv.Itoa(len(childIndent))
			result = append(result, childIndent+"{{- toYaml .Values.manager.annotations | nindent "+childIndentWidth+" }}")
			result = append(result, metadataChildIndent+"{{- end }}")
			state.addedAnnotationsToDeployment = true
		}

		t.updateMetadataTracking(state, lines, i, trimmed, indentLen)
		result = append(result, line)

		result = t.handleDeploymentAnnotations(state, result, line, trimmed, indent, indentLen)
		result = t.handleDeploymentLabels(state, result, line, trimmed, indentLen)
		result = t.handlePodAnnotations(state, result, line, trimmed, indent, indentLen)
		result = t.handlePodLabels(state, result, line, trimmed, indentLen)
	}

	return strings.Join(result, "\n")
}

// templateServiceAccount templates ServiceAccount name, labels, annotations, and conditional rendering
func (t *HelmTemplater) templateServiceAccount(yamlContent string) string {
	yamlContent = t.addServiceAccountLabelsAndAnnotations(yamlContent)
	yamlContent = t.templateServiceAccountName(yamlContent)
	yamlContent = t.wrapServiceAccountWithEnableConditional(yamlContent)
	return yamlContent
}

// templateServiceAccountName replaces SA name with serviceAccountName helper
func (t *HelmTemplater) templateServiceAccountName(yamlContent string) string {
	replacement := `${1}name: {{ include "` + t.chartName + `.serviceAccountName" . }}`

	// Handle name with prefix
	namePattern := regexp.MustCompile(
		`(?m)^(\s*)name:\s+` + regexp.QuoteMeta(t.detectedPrefix) + `-controller-manager\s*$`)
	yamlContent = namePattern.ReplaceAllString(yamlContent, replacement)

	// Handle name without prefix
	namePatternSimple := regexp.MustCompile(`(?m)^(\s*)name:\s+controller-manager\s*$`)
	yamlContent = namePatternSimple.ReplaceAllString(yamlContent, replacement)

	return yamlContent
}

// wrapServiceAccountWithEnableConditional wraps SA in serviceAccount.enable conditional
func (t *HelmTemplater) wrapServiceAccountWithEnableConditional(yamlContent string) string {
	// Ensure yamlContent ends with newline so {{- end }} is on its own line
	if !strings.HasSuffix(yamlContent, "\n") {
		yamlContent += "\n"
	}
	// Default to enabled, but allow an explicit false to disable ServiceAccount creation
	return "{{- if ne .Values.serviceAccount.enable false }}\n" + yamlContent + "{{- end }}\n"
}

// addServiceAccountLabelsAndAnnotations adds custom labels/annotations with omit() filtering
func (t *HelmTemplater) addServiceAccountLabelsAndAnnotations(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	result := make([]string, 0, len(lines))
	addedCustomFields := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Look for labels: in metadata
		if strings.HasPrefix(trimmed, yamlKeyLabels) && !addedCustomFields {
			result = append(result, line)
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))

			// Collect all existing label lines
			labelsStart := len(result)
			i++
			for i < len(lines) {
				nextLine := lines[i]
				nextTrimmed := strings.TrimSpace(nextLine)
				nextIndent := len(nextLine) - len(strings.TrimLeft(nextLine, " \t"))

				// Stop when we hit a line at same or less indentation (not comment/empty)
				if nextTrimmed != "" && !strings.HasPrefix(nextTrimmed, "#") && nextIndent <= currentIndent {
					break
				}
				result = append(result, nextLine)
				i++
			}

			// Extract existing label keys
			existingKeys := t.extractKeysFromLines(result[labelsStart:])
			childIndent := strings.Repeat(" ", currentIndent+2)

			// Add custom labels
			result = t.appendHelmMapBlock(result, childIndent, ".Values.serviceAccount.labels", existingKeys)

			// Merge into an existing annotations block when present; otherwise add one.
			indent := strings.Repeat(" ", currentIndent)
			if i < len(lines) {
				nextLine := lines[i]
				nextTrimmed := strings.TrimSpace(nextLine)
				nextIndent := len(nextLine) - len(strings.TrimLeft(nextLine, " \t"))

				if strings.HasPrefix(nextTrimmed, "annotations:") && nextIndent == currentIndent {
					result = append(result, nextLine)
					annotationsStart := len(result)
					i++
					for i < len(lines) {
						annotationLine := lines[i]
						annotationTrimmed := strings.TrimSpace(annotationLine)
						annotationIndent := len(annotationLine) - len(strings.TrimLeft(annotationLine, " \t"))

						// Stop when we hit a line at same or less indentation (not comment/empty)
						if annotationTrimmed != "" && !strings.HasPrefix(annotationTrimmed, "#") && annotationIndent <= currentIndent {
							break
						}
						result = append(result, annotationLine)
						i++
					}

					existingAnnotationKeys := t.extractKeysFromLines(result[annotationsStart:])
					result = t.appendHelmMapBlock(result, childIndent, ".Values.serviceAccount.annotations", existingAnnotationKeys)
				} else {
					result = append(result,
						indent+"{{- with .Values.serviceAccount.annotations }}",
						indent+"annotations:",
						childIndent+"{{- toYaml . | nindent "+strconv.Itoa(currentIndent+2)+" }}",
						indent+"{{- end }}",
					)
				}
			} else {
				result = append(result,
					indent+"{{- with .Values.serviceAccount.annotations }}",
					indent+"annotations:",
					childIndent+"{{- toYaml . | nindent "+strconv.Itoa(currentIndent+2)+" }}",
					indent+"{{- end }}",
				)
			}

			addedCustomFields = true
			i-- // Adjust because outer loop will increment
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// templateServiceMonitor templates ServiceMonitor scheme and port based on metrics.secure
func (t *HelmTemplater) templateServiceMonitor(yamlContent string) string {
	yamlContent = regexp.MustCompile(`(\s*)port:\s*https`).
		ReplaceAllString(yamlContent, `${1}port: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}`)

	yamlContent = regexp.MustCompile(`(\s*)scheme:\s*https`).
		ReplaceAllString(yamlContent, `${1}scheme: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}`)

	// Make bearer token and TLS config conditional on metrics.secure
	yamlContent = t.makeServiceMonitorBearerTokenConditional(yamlContent)
	yamlContent = t.makeServiceMonitorTLSConditional(yamlContent)

	return yamlContent
}

// makeServiceMonitorTLSConditional wraps ServiceMonitor tlsConfig with metrics.secure check
func (t *HelmTemplater) makeServiceMonitorTLSConditional(yamlContent string) string {
	// Use line-based parsing to avoid over-capturing (regex would capture selector block)
	lines := strings.Split(yamlContent, "\n")
	var result []string
	inTLSConfig := false
	tlsConfigIndent := 0
	var tlsBlock []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))

		if strings.HasPrefix(trimmed, "tlsConfig:") {
			inTLSConfig = true
			tlsConfigIndent = currentIndent
			tlsBlock = []string{line}
			continue
		}

		if inTLSConfig {
			// Stop when we hit a line with same or less indentation (not empty/comment)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && currentIndent <= tlsConfigIndent {
				indentStr := strings.Repeat(" ", tlsConfigIndent)
				result = append(result, fmt.Sprintf("%s{{- if .Values.metrics.secure }}", indentStr))
				result = append(result, tlsBlock...)
				result = append(result, fmt.Sprintf("%s{{- end }}", indentStr))
				inTLSConfig = false
				tlsBlock = nil
				result = append(result, line)
			} else {
				tlsBlock = append(tlsBlock, line)
			}
		} else {
			result = append(result, line)
		}

		if inTLSConfig && i == len(lines)-1 {
			indentStr := strings.Repeat(" ", tlsConfigIndent)
			result = append(result, fmt.Sprintf("%s{{- if .Values.metrics.secure }}", indentStr))
			result = append(result, tlsBlock...)
			result = append(result, fmt.Sprintf("%s{{- end }}", indentStr))
		}
	}

	return strings.Join(result, "\n")
}

// handleDeploymentAnnotations handles injection of custom Deployment annotations.
func (t *HelmTemplater) handleDeploymentAnnotations(
	state *customFieldsState, result []string, line, trimmed, indent string, indentLen int,
) []string {
	if state.position == positionDeploymentMetadata &&
		state.currentBlock == blockNone &&
		(trimmed == yamlKeyAnnotations || strings.HasPrefix(trimmed, yamlKeyAnnotations)) {
		state.hasDeploymentAnnotations = true
		state.currentBlock = blockDeploymentAnnotations
		state.currentBlockIndent = indentLen
		return t.handleFlowStyleAnnotations(result, line, indent)
	}

	if t.shouldInjectDeploymentAnnotations(state, trimmed, indentLen) {
		result = result[:len(result)-1]

		existingKeys := t.extractKeysFromLines(result)
		parentIndent := strings.Repeat(" ", state.currentBlockIndent)
		childIndent := t.detectChildIndent(result, parentIndent)

		if len(existingKeys) == 0 {
			// Empty annotations block (e.g., from "annotations: {}") - wrap header in conditional
			// to avoid rendering "annotations: null" when values not set
			result = result[:len(result)-1]
			childIndentWidth := strconv.Itoa(len(childIndent))
			result = append(result,
				parentIndent+"{{- if .Values.manager.annotations }}",
				parentIndent+"annotations:",
				childIndent+"{{- toYaml .Values.manager.annotations | nindent "+childIndentWidth+" }}",
				parentIndent+"{{- end }}",
			)
		} else {
			// Has existing annotations - inject additional ones with omit() filtering
			result = t.injectDeploymentAnnotations(result, childIndent)
		}

		result = append(result, line)
		state.addedAnnotationsToDeployment = true
		state.currentBlock = blockNone
	}

	return result
}

// handlePodAnnotations handles injection of custom Pod template annotations.
func (t *HelmTemplater) handlePodAnnotations(
	state *customFieldsState, result []string, line, trimmed, indent string, indentLen int,
) []string {
	if state.position == positionPodMetadata &&
		state.currentBlock == blockNone &&
		(trimmed == yamlKeyAnnotations || strings.HasPrefix(trimmed, yamlKeyAnnotations)) {
		state.currentBlock = blockPodAnnotations
		state.currentBlockIndent = indentLen
		return t.handleFlowStyleAnnotations(result, line, indent)
	}

	if t.shouldInjectPodAnnotations(state, trimmed, indentLen) {
		result = result[:len(result)-1]

		existingKeys := t.extractKeysFromLines(result)
		parentIndent := strings.Repeat(" ", state.currentBlockIndent)
		childIndent := t.detectChildIndent(result, parentIndent)

		if len(existingKeys) == 0 {
			// Empty annotations block (e.g., from "annotations: {}") - wrap header in conditional
			// to avoid rendering "annotations: null" when values not set
			result = result[:len(result)-1]
			childIndentWidth := strconv.Itoa(len(childIndent))
			result = append(result,
				parentIndent+"{{- with .Values.manager.pod }}",
				parentIndent+"{{- with .annotations }}",
				parentIndent+"annotations:",
				childIndent+"{{- toYaml . | nindent "+childIndentWidth+" }}",
				parentIndent+"{{- end }}",
				parentIndent+"{{- end }}",
			)
		} else {
			// Has existing annotations - inject additional ones with omit() filtering
			result = t.addPodAnnotations(result, childIndent)
		}

		result = append(result, line)
		state.addedPodAnnotations = true
		state.currentBlock = blockNone
	}

	if state.position == positionPodMetadata && !state.addedPodAnnotations && trimmed == yamlKeyLabels {
		result = result[:len(result)-1]
		result = append(result, indent+"{{- if .Values.manager.pod.annotations }}")
		result = append(result, indent+"annotations:")
		result = t.addPodAnnotations(result, indent+"  ")
		result = append(result, indent+"{{- end }}")
		result = append(result, indent+yamlKeyLabels)
		state.addedPodAnnotations = true
	}

	return result
}

// shouldInjectDeploymentAnnotations checks if we should inject Deployment annotations.
func (t *HelmTemplater) shouldInjectDeploymentAnnotations(
	state *customFieldsState, trimmed string, indentLen int,
) bool {
	return (state.position == positionDeploymentMetadata || state.position == positionAfterDeploymentMetadata) &&
		state.currentBlock == blockDeploymentAnnotations &&
		!state.addedAnnotationsToDeployment &&
		indentLen <= state.currentBlockIndent &&
		trimmed != "" &&
		trimmed != yamlKeyAnnotations &&
		!strings.HasPrefix(trimmed, yamlKeyAnnotations+" {")
}

// shouldInjectPodAnnotations checks if we should inject Pod annotations.
func (t *HelmTemplater) shouldInjectPodAnnotations(state *customFieldsState, trimmed string, indentLen int) bool {
	return (state.position == positionPodMetadata || state.position == positionAfterDeploymentMetadata) &&
		state.currentBlock == blockPodAnnotations &&
		!state.addedPodAnnotations &&
		indentLen <= state.currentBlockIndent &&
		trimmed != "" &&
		trimmed != yamlKeyAnnotations &&
		!strings.HasPrefix(trimmed, yamlKeyAnnotations+" {")
}

// handleDeploymentLabels handles injection of custom Deployment labels.
func (t *HelmTemplater) handleDeploymentLabels(
	state *customFieldsState, result []string, line, trimmed string, indentLen int,
) []string {
	if state.position == positionDeploymentMetadata &&
		state.currentBlock == blockNone &&
		trimmed == yamlKeyLabels {
		state.currentBlock = blockDeploymentLabels
		state.currentBlockIndent = indentLen
		return result
	}

	if t.shouldInjectDeploymentLabels(state, trimmed, indentLen) {
		result = result[:len(result)-1]
		parentIndent := strings.Repeat(" ", state.currentBlockIndent)
		childIndent := t.detectChildIndent(result, parentIndent)
		result = t.injectDeploymentLabels(result, childIndent)
		result = append(result, line)
		state.addedLabelsToDeployment = true
		state.currentBlock = blockNone
	}

	return result
}

// handlePodLabels handles injection of custom Pod template labels.
func (t *HelmTemplater) handlePodLabels(
	state *customFieldsState, result []string, line, trimmed string, indentLen int,
) []string {
	if state.position == positionPodMetadata &&
		state.currentBlock == blockNone &&
		trimmed == yamlKeyLabels {
		state.currentBlock = blockPodLabels
		state.currentBlockIndent = indentLen
		return result
	}

	if t.shouldInjectPodLabels(state, trimmed, indentLen) {
		result = result[:len(result)-1]
		parentIndent := strings.Repeat(" ", state.currentBlockIndent)
		childIndent := t.detectChildIndent(result, parentIndent)
		result = t.injectPodLabels(result, childIndent)
		result = append(result, line)
		state.addedPodLabels = true
		state.currentBlock = blockNone
	}

	return result
}

// shouldInjectDeploymentLabels checks if we should inject Deployment labels.
func (t *HelmTemplater) shouldInjectDeploymentLabels(
	state *customFieldsState, trimmed string, indentLen int,
) bool {
	return (state.position == positionDeploymentMetadata || state.position == positionAfterDeploymentMetadata) &&
		state.currentBlock == blockDeploymentLabels &&
		!state.addedLabelsToDeployment &&
		indentLen <= state.currentBlockIndent &&
		trimmed != "" &&
		trimmed != yamlKeyLabels
}

// shouldInjectPodLabels checks if we should inject Pod labels.
func (t *HelmTemplater) shouldInjectPodLabels(
	state *customFieldsState, trimmed string, indentLen int,
) bool {
	return (state.position == positionPodMetadata || state.position == positionAfterDeploymentMetadata) &&
		state.currentBlock == blockPodLabels &&
		!state.addedPodLabels &&
		indentLen <= state.currentBlockIndent &&
		trimmed != "" &&
		trimmed != yamlKeyLabels
}

// appendHelmMapBlock appends Helm template blocks for rendering YAML maps with optional key filtering.
// When existingKeys is empty, uses simple {{- if }} conditional.
// When existingKeys is provided, uses nested {{- with }} blocks with omit() to filter duplicate keys.
func (t *HelmTemplater) appendHelmMapBlock(
	result []string,
	indent string,
	valuePath string,
	existingKeys []string,
) []string {
	childIndentWidth := strconv.Itoa(len(indent))

	if len(existingKeys) > 0 {
		omitKeys := strings.Join(existingKeys, "\" \"")
		return append(result,
			indent+"{{- with "+valuePath+" }}",
			indent+"{{- with omit . \""+omitKeys+"\" }}",
			indent+"{{- toYaml . | nindent "+childIndentWidth+" }}",
			indent+"{{- end }}",
			indent+"{{- end }}",
		)
	}

	return append(result,
		indent+"{{- if "+valuePath+" }}",
		indent+"{{- toYaml "+valuePath+" | nindent "+childIndentWidth+" }}",
		indent+"{{- end }}",
	)
}

// appendNestedHelmMapBlock appends nested Helm template blocks (e.g., .Values.manager.pod -> .labels).
// When existingKeys is empty, uses nested {{- with }} without omit().
// When existingKeys is provided, adds an extra {{- with omit() }} layer for key filtering.
func (t *HelmTemplater) appendNestedHelmMapBlock(
	result []string,
	indent string,
	outerPath string,
	innerPath string,
	existingKeys []string,
) []string {
	childIndentWidth := strconv.Itoa(len(indent))

	if len(existingKeys) > 0 {
		omitKeys := strings.Join(existingKeys, "\" \"")
		return append(result,
			indent+"{{- with "+outerPath+" }}",
			indent+"{{- with "+innerPath+" }}",
			indent+"{{- with omit . \""+omitKeys+"\" }}",
			indent+"{{- toYaml . | nindent "+childIndentWidth+" }}",
			indent+"{{- end }}",
			indent+"{{- end }}",
			indent+"{{- end }}",
		)
	}

	return append(result,
		indent+"{{- with "+outerPath+" }}",
		indent+"{{- with "+innerPath+" }}",
		indent+"{{- toYaml . | nindent "+childIndentWidth+" }}",
		indent+"{{- end }}",
		indent+"{{- end }}",
	)
}

// injectDeploymentLabels injects the Helm template block for custom Deployment labels.
func (t *HelmTemplater) injectDeploymentLabels(result []string, childIndent string) []string {
	existingKeys := t.extractKeysFromLines(result)
	return t.appendHelmMapBlock(result, childIndent, ".Values.manager.labels", existingKeys)
}

// injectPodLabels injects the Helm template block for custom Pod template labels.
func (t *HelmTemplater) injectPodLabels(result []string, childIndent string) []string {
	existingKeys := t.extractKeysFromLines(result)
	return t.appendNestedHelmMapBlock(result, childIndent, ".Values.manager.pod", ".labels", existingKeys)
}

// metadataPosition represents the current position in the deployment manifest.
type metadataPosition int

const (
	positionStart metadataPosition = iota
	positionDeploymentMetadata
	positionAfterDeploymentMetadata
	positionPodMetadata
)

// blockType represents which specific YAML block we're currently inside.
type blockType int

const (
	blockNone blockType = iota
	blockDeploymentLabels
	blockDeploymentAnnotations
	blockPodLabels
	blockPodAnnotations
)

// customFieldsState tracks parsing state for injecting custom labels and annotations.
type customFieldsState struct {
	// Current position in the manifest structure
	position                metadataPosition
	deploymentMetadataDepth int

	// Flags to track if we've already injected templates (prevent duplicates)
	addedLabelsToDeployment      bool
	addedPodLabels               bool
	addedAnnotationsToDeployment bool
	addedPodAnnotations          bool
	// Whether Deployment has an annotations field in the kustomize output
	hasDeploymentAnnotations bool

	// Current block being parsed and its indentation
	currentBlock       blockType
	currentBlockIndent int
}

// updateMetadataTracking updates the position state as we traverse the YAML structure.
func (t *HelmTemplater) updateMetadataTracking(
	state *customFieldsState, lines []string, i int, trimmed string, indentLen int,
) {
	// Track Deployment metadata section
	if trimmed == "metadata:" && i > 0 {
		prevLine := strings.TrimSpace(lines[i-1])
		if strings.HasPrefix(prevLine, "kind: Deployment") || prevLine == "kind: Deployment" {
			state.position = positionDeploymentMetadata
			state.deploymentMetadataDepth = indentLen
		} else if prevLine == "template:" {
			// Track Pod template metadata section
			state.position = positionPodMetadata
		}
	}

	// Exit deployment metadata when we reach spec:
	if state.position == positionDeploymentMetadata && trimmed == "spec:" && indentLen == state.deploymentMetadataDepth {
		state.position = positionAfterDeploymentMetadata
	}

	// Exit pod template metadata when we reach spec: (pod spec)
	if state.position == positionPodMetadata && trimmed == "spec:" {
		state.position = positionAfterDeploymentMetadata
	}
}

// detectChildIndent detects the actual child indentation from existing entries in the current block.
// Returns the detected indentation string, or parentIndent + "  " (2 spaces) as default.
func (t *HelmTemplater) detectChildIndent(lines []string, parentIndent string) string {
	// Scan backwards to find the first child entry with indentation > parent
	parentIndentLen := len(parentIndent)

	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and Helm template directives
		if trimmed == "" || strings.HasPrefix(trimmed, "{{") {
			continue
		}

		// Stop at section headers
		if trimmed == yamlKeyLabels || trimmed == yamlKeyAnnotations ||
			trimmed == "metadata:" || trimmed == "spec:" || trimmed == "template:" {
			break
		}

		// Find a line with indentation greater than parent (a child entry)
		indent, indentLen := leadingWhitespace(line)
		if indentLen > parentIndentLen && strings.Contains(line, ":") {
			return indent
		}
	}

	// Default to 2-space indentation (sigs.k8s.io/yaml standard)
	return parentIndent + "  "
}

// extractKeysFromLines extracts YAML keys from labels/annotations sections.
// Scans backwards to find the section header to avoid missing keys in large blocks.
func (t *HelmTemplater) extractKeysFromLines(lines []string) []string {
	keys := []string{}

	// Find section start by scanning backwards to the nearest header
	sectionStart := 0
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		// Stop at section headers - this is where our current section began
		if trimmed == yamlKeyLabels || trimmed == yamlKeyAnnotations {
			sectionStart = i + 1 // Start extracting from the line after the header
			break
		}
		// Also stop at other major structural boundaries
		if trimmed == "metadata:" || trimmed == "spec:" || trimmed == "template:" {
			sectionStart = i + 1
			break
		}
	}

	// Matches YAML keys: "  key: value" (supports dots, slashes, hyphens)
	keyPattern := regexp.MustCompile(`^\s+([a-zA-Z0-9._/-]+):\s+`)

	for i := sectionStart; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip Helm template directives (e.g., "{{- if ... }}", "{{- end }}"),
		// but still parse YAML key/value lines whose values contain templates.
		// This allows extracting keys like "app.kubernetes.io/name: {{ include ... }}"
		if strings.HasPrefix(trimmed, "{{") {
			continue
		}

		// Stop if we hit another section header
		if trimmed == yamlKeyLabels || trimmed == yamlKeyAnnotations || trimmed == "metadata:" ||
			trimmed == "spec:" || trimmed == "template:" {
			break
		}

		if matches := keyPattern.FindStringSubmatch(line); matches != nil {
			keys = append(keys, matches[1])
		}
	}

	return keys
}

// injectDeploymentAnnotations injects the Helm template block for custom Deployment annotations.
func (t *HelmTemplater) injectDeploymentAnnotations(result []string, indent string) []string {
	existingKeys := t.extractKeysFromLines(result)
	return t.appendHelmMapBlock(result, indent, ".Values.manager.annotations", existingKeys)
}

// addPodAnnotations adds custom annotations to the Pod template metadata.
func (t *HelmTemplater) addPodAnnotations(result []string, indent string) []string {
	existingKeys := t.extractKeysFromLines(result)
	return t.appendNestedHelmMapBlock(result, indent, ".Values.manager.pod", ".annotations", existingKeys)
}

// handleFlowStyleAnnotations detects and converts flow-style annotations to block-style.
// Flow-style example: "annotations: {key: value, key2: value2}"
// Block-style output:
//
//	annotations:
//	  key: value
//	  key2: value2
//
// This conversion is necessary because we cannot inject Helm template blocks after
// a flow-style mapping - it would produce invalid YAML.
func (t *HelmTemplater) handleFlowStyleAnnotations(
	result []string, line string, indent string,
) []string {
	trimmed := strings.TrimSpace(line)

	// Detect flow-style annotations: annotations:{} or annotations: {}
	flowPattern := regexp.MustCompile(`annotations:\s*\{`)
	if !flowPattern.MatchString(trimmed) {
		return result
	}

	// Extract the flow-style content
	annotationsStart := strings.Index(line, yamlKeyAnnotations)
	if annotationsStart == -1 {
		return result
	}

	// Find the content after "annotations: "
	contentStart := annotationsStart + len(yamlKeyAnnotations)
	flowContent := strings.TrimSpace(line[contentStart:])

	// Remove the flow-style line we just added
	result = result[:len(result)-1]

	// Add block-style annotations: key
	result = append(result, indent+yamlKeyAnnotations)

	// Parse and convert flow-style entries to block-style
	// Use YAML parser to properly handle quoted values and edge cases
	if strings.HasPrefix(flowContent, "{") && strings.HasSuffix(flowContent, "}") {
		var flowMap map[string]any
		if err := yaml.Unmarshal([]byte(flowContent), &flowMap); err == nil {
			childIndent := indent + "  "
			// When map is empty, leave just the header (annotations:) with no children.
			// Template injection will add conditional blocks underneath, avoiding invalid
			// YAML that would result from mixing a {} scalar with mapping entries.
			if len(flowMap) > 0 {
				// Sort keys to ensure deterministic output
				sortedKeys := make([]string, 0, len(flowMap))
				for key := range flowMap {
					sortedKeys = append(sortedKeys, key)
				}
				slices.Sort(sortedKeys)

				for _, key := range sortedKeys {
					value := flowMap[key]
					// Marshal the value to get proper YAML representation
					valueBytes, err := yaml.Marshal(value)
					if err != nil {
						continue
					}
					valueStr := strings.TrimSpace(string(valueBytes))
					result = append(result, fmt.Sprintf("%s%s: %s", childIndent, key, valueStr))
				}
			}
		} else {
			// Fallback: simple comma split (best effort for non-standard formats)
			flowContent = strings.TrimPrefix(flowContent, "{")
			flowContent = strings.TrimSuffix(flowContent, "}")
			flowContent = strings.TrimSpace(flowContent)
			if flowContent != "" {
				entries := strings.Split(flowContent, ",")
				childIndent := indent + "  "
				for _, entry := range entries {
					entry = strings.TrimSpace(entry)
					if entry != "" {
						result = append(result, childIndent+entry)
					}
				}
			}
		}
	}

	return result
}

// makeServiceMonitorBearerTokenConditional wraps bearerTokenFile with metrics.secure check
func (t *HelmTemplater) makeServiceMonitorBearerTokenConditional(yamlContent string) string {
	// Handle case where bearerTokenFile is first field in list: "  - bearerTokenFile: ..."
	listItemPattern := regexp.MustCompile(`(?m)^(\s*)-\s+bearerTokenFile:\s*([^\n]+)`)
	yamlContent = listItemPattern.ReplaceAllString(yamlContent,
		`$1- {{- if .Values.metrics.secure }}`+"\n"+`$1  bearerTokenFile: $2`+"\n"+`$1  {{- end }}`)

	return yamlContent
}
