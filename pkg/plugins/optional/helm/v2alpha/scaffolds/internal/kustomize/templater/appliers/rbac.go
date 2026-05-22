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
	"slices"
	"strconv"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// This file contains all RBAC and ServiceAccount transformations:
//  - SubstituteRBACValues: Role and RoleBinding name templating
//  - TemplateServiceAccountNameInBindings: SA name in RoleBinding/ClusterRoleBinding subjects
//  - TemplateServiceAccountNameInDeployment: SA name in Deployment spec
//  - TemplateServiceAccount: ServiceAccount-specific transformations (labels, annotations, conditionals)
//
// ServiceAccount is part of RBAC, so these transformations are logically grouped together.

// SubstituteRBACValues applies RBAC-specific template substitutions.
func SubstituteRBACValues(detectedPrefix, chartName, yamlContent string) string {
	roleRefBlockPattern := regexp.MustCompile(
		`(?s)(roleRef:\s*\n(?:\s+\w+:.*\n)*?)(\s+)name:\s+` +
			regexp.QuoteMeta(detectedPrefix) + `-manager-role`)
	yamlContent = roleRefBlockPattern.ReplaceAllString(
		yamlContent, `${1}${2}name: `+ResourceNameTemplate(chartName, "manager-role"))

	roleRefBlockPatternSimple := regexp.MustCompile(
		`(?s)(roleRef:\s*\n(?:\s+\w+:.*\n)*?)(\s+)name:\s+manager-role`)
	yamlContent = roleRefBlockPatternSimple.ReplaceAllString(
		yamlContent, `${1}${2}name: `+ResourceNameTemplate(chartName, "manager-role"))

	yamlContent = TemplateServiceAccountNameInBindings(detectedPrefix, chartName, yamlContent)

	return yamlContent
}

// TemplateServiceAccountNameInBindings templates SA name in RoleBinding/ClusterRoleBinding subjects.
func TemplateServiceAccountNameInBindings(detectedPrefix, chartName, yamlContent string) string {
	replacement := `{{ include "` + chartName + `.serviceAccountName" . }}`

	// Handle already-templated resourceName (from substituteResourceNamesWithPrefix)
	templatedPattern := regexp.MustCompile(
		`(?m)(subjects:\s*\n\s*-\s*kind:\s*ServiceAccount\s*\n\s+name:\s+)` +
			regexp.QuoteMeta(`{{ include "`+chartName+`.resourceName" (dict "suffix" "controller-manager" "context" $) }}`))
	yamlContent = templatedPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names with prefix
	subjectPattern := regexp.MustCompile(
		`(?m)(subjects:\s*\n\s*-\s*kind:\s*ServiceAccount\s*\n\s+name:\s+)` +
			regexp.QuoteMeta(detectedPrefix) + `-controller-manager`)
	yamlContent = subjectPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names without prefix
	subjectPatternSimple := regexp.MustCompile(
		`(?m)(subjects:\s*\n\s*-\s*kind:\s*ServiceAccount\s*\n\s+name:\s+)controller-manager`)
	yamlContent = subjectPatternSimple.ReplaceAllString(yamlContent, `${1}`+replacement)

	return yamlContent
}

// TemplateServiceAccountNameInDeployment templates serviceAccountName in Deployment spec.
func TemplateServiceAccountNameInDeployment(detectedPrefix, chartName, yamlContent string) string {
	replacement := `serviceAccountName: {{ include "` + chartName + `.serviceAccountName" . }}`

	// Handle already-templated resourceName (from substituteResourceNamesWithPrefix)
	templatedPattern := regexp.MustCompile(
		`(?m)^(\s*)serviceAccountName:\s+` +
			regexp.QuoteMeta(`{{ include "`+chartName+`.resourceName" (dict "suffix" "controller-manager" "context" $) }}`))
	yamlContent = templatedPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names with prefix
	serviceAccountPattern := regexp.MustCompile(
		`(?m)^(\s*)serviceAccountName:\s+` + regexp.QuoteMeta(detectedPrefix) + `-controller-manager\s*$`)
	yamlContent = serviceAccountPattern.ReplaceAllString(yamlContent, `${1}`+replacement)

	// Handle literal names without prefix
	serviceAccountPatternSimple := regexp.MustCompile(
		`(?m)^(\s*)serviceAccountName:\s+controller-manager\s*$`)
	yamlContent = serviceAccountPatternSimple.ReplaceAllString(yamlContent, `${1}`+replacement)

	return yamlContent
}

// TemplateServiceAccount applies all ServiceAccount-specific transformations.
func TemplateServiceAccount(detectedPrefix, chartName, yamlContent string) string {
	yamlContent = AddServiceAccountLabelsAndAnnotations(yamlContent)
	yamlContent = TemplateServiceAccountName(detectedPrefix, chartName, yamlContent)
	yamlContent = WrapServiceAccountWithEnableConditional(yamlContent)
	return yamlContent
}

// TemplateServiceAccountName replaces SA name with serviceAccountName helper.
func TemplateServiceAccountName(detectedPrefix, chartName, yamlContent string) string {
	replacement := `${1}name: {{ include "` + chartName + `.serviceAccountName" . }}`

	// Handle name with prefix
	namePattern := regexp.MustCompile(
		`(?m)^(\s*)name:\s+` + regexp.QuoteMeta(detectedPrefix) + `-controller-manager\s*$`)
	yamlContent = namePattern.ReplaceAllString(yamlContent, replacement)

	// Handle name without prefix
	namePatternSimple := regexp.MustCompile(`(?m)^(\s*)name:\s+controller-manager\s*$`)
	yamlContent = namePatternSimple.ReplaceAllString(yamlContent, replacement)

	return yamlContent
}

// WrapServiceAccountWithEnableConditional wraps SA in serviceAccount.enable conditional.
func WrapServiceAccountWithEnableConditional(yamlContent string) string {
	// Ensure yamlContent ends with newline so {{- end }} is on its own line
	if !strings.HasSuffix(yamlContent, "\n") {
		yamlContent += "\n"
	}
	// Default to enabled, but allow an explicit false to disable ServiceAccount creation
	return "{{- if ne .Values.serviceAccount.enable false }}\n" + yamlContent + "{{- end }}\n"
}

// AddServiceAccountLabelsAndAnnotations adds custom labels/annotations with omit() filtering.
func AddServiceAccountLabelsAndAnnotations(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	result := make([]string, 0, len(lines))
	addedCustomFields := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Look for labels: in metadata
		if strings.HasPrefix(trimmed, common.YamlKeyLabels) && !addedCustomFields {
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
			existingKeys := extractKeysFromLines(result[labelsStart:])
			childIndent := strings.Repeat(" ", currentIndent+2)

			// Add custom labels
			result = appendHelmMapBlock(result, childIndent, ".Values.serviceAccount.labels", existingKeys)

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

					existingAnnotationKeys := extractKeysFromLines(result[annotationsStart:])
					result = appendHelmMapBlock(result, childIndent, ".Values.serviceAccount.annotations", existingAnnotationKeys)
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

// appendHelmMapBlock appends Helm template blocks for custom labels/annotations.
func appendHelmMapBlock(
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

// extractKeysFromLines extracts YAML keys from labels/annotations sections.
func extractKeysFromLines(lines []string) []string {
	keys := []string{}

	// Find section start by scanning backwards to the nearest header
	sectionStart := 0
	for i, v := range slices.Backward(lines) {
		trimmed := strings.TrimSpace(v)
		// Stop at section headers - this is where our current section began
		if trimmed == common.YamlKeyLabels || trimmed == common.YamlKeyAnnotations {
			sectionStart = i + 1 // Start extracting from the line after the header
			break
		}
		// Also stop at other major structural boundaries
		if trimmed == common.YamlKeyMetadata || trimmed == common.YamlKeySpec || trimmed == common.YamlKeyTemplate {
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
		if strings.HasPrefix(trimmed, "{{") {
			continue
		}

		// Stop if we hit another section header
		if trimmed == common.YamlKeyLabels || trimmed == common.YamlKeyAnnotations ||
			trimmed == common.YamlKeyMetadata || trimmed == common.YamlKeySpec || trimmed == common.YamlKeyTemplate {
			break
		}

		// Extract the key name from "key: value" patterns
		if matches := keyPattern.FindStringSubmatch(line); len(matches) > 1 {
			keys = append(keys, matches[1])
		}
	}

	return keys
}
