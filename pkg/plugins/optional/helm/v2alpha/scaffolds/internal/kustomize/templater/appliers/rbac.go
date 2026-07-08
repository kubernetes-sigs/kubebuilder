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
	yamlContent = WrapServiceAccountWithEnabledConditional(yamlContent)
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

// WrapServiceAccountWithEnabledConditional wraps SA in serviceAccount.enabled conditional.
func WrapServiceAccountWithEnabledConditional(yamlContent string) string {
	// Ensure yamlContent ends with newline so {{- end }} is on its own line
	if !strings.HasSuffix(yamlContent, "\n") {
		yamlContent += "\n"
	}
	// Create the ServiceAccount only when serviceAccount.enabled is truthy. Plain truthiness matches
	// the serviceAccountName helper and every other section toggle in the chart.
	return "{{- if .Values.serviceAccount.enabled }}\n" + yamlContent + "{{- end }}\n"
}

const (
	valuesServiceAccountLabels      = ".Values.serviceAccount.labels"
	valuesServiceAccountAnnotations = ".Values.serviceAccount.annotations"
)

// AddServiceAccountLabelsAndAnnotations makes the ServiceAccount metadata honor
// .Values.serviceAccount.labels and .Values.serviceAccount.annotations. It merges into whichever
// labels and annotations blocks Kustomize already emitted, in either order, and injects the block
// that is missing. User-supplied values therefore always render and no metadata key is duplicated.
func AddServiceAccountLabelsAndAnnotations(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	merged := make([]string, 0, len(lines))

	metadataIndent := -1
	metadataLineIndex := -1
	labelsBlockEnd := -1
	annotationsBlockEnd := -1

	for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
		switch trimmed := strings.TrimSpace(lines[lineIndex]); {
		case trimmed == common.YamlKeyMetadata:
			_, metadataIndent = LeadingWhitespace(lines[lineIndex])
			metadataLineIndex = len(merged)
			merged = append(merged, lines[lineIndex])
		case isMetadataMapChildHeader(lines[lineIndex], common.YamlKeyLabels, metadataIndent):
			merged, lineIndex = mergeMetadataMapBlock(
				merged, lines, lineIndex, common.YamlKeyLabels, valuesServiceAccountLabels)
			labelsBlockEnd = len(merged)
		case isMetadataMapChildHeader(lines[lineIndex], common.YamlKeyAnnotations, metadataIndent):
			merged, lineIndex = mergeMetadataMapBlock(
				merged, lines, lineIndex, common.YamlKeyAnnotations, valuesServiceAccountAnnotations)
			annotationsBlockEnd = len(merged)
		default:
			merged = append(merged, lines[lineIndex])
		}
	}

	childIndent := 2
	if metadataIndent >= 0 {
		childIndent = metadataIndent + 2
	}

	merged = injectMissingMetadataBlocks(merged, childIndent, metadataLineIndex, labelsBlockEnd, annotationsBlockEnd)
	return strings.Join(merged, "\n")
}

// isMetadataMapHeader reports whether trimmed is the header for the given metadata map key
// (for example "labels:"), including the inline empty-map form "labels: {}".
func isMetadataMapHeader(trimmed, mapKey string) bool {
	return trimmed == mapKey || trimmed == mapKey+" {}"
}

// isMetadataMapChildHeader reports whether line is the mapKey header (labels: or annotations:)
// sitting directly under metadata:, indented metadataIndent + 2. It returns false before metadata:
// is seen, so a key nested deeper in the resource is left untouched instead of merged.
func isMetadataMapChildHeader(line, mapKey string, metadataIndent int) bool {
	if metadataIndent < 0 {
		return false
	}
	_, indent := LeadingWhitespace(line)
	if indent != metadataIndent+2 {
		return false
	}
	return isMetadataMapHeader(strings.TrimSpace(line), mapKey)
}

// injectMissingMetadataBlocks adds a guarded labels or annotations block for whichever one the
// source metadata did not contain. The new block is placed next to the block that is present, or
// right after "metadata:" when neither was present, so user-supplied values always render.
func injectMissingMetadataBlocks(
	merged []string,
	childIndent, metadataLineIndex, labelsBlockEnd, annotationsBlockEnd int,
) []string {
	labelsBlock := buildGuardedMetadataMapBlock(
		childIndent, common.YamlKeyLabels, valuesServiceAccountLabels)
	annotationsBlock := buildGuardedMetadataMapBlock(
		childIndent, common.YamlKeyAnnotations, valuesServiceAccountAnnotations)

	switch {
	case labelsBlockEnd >= 0 && annotationsBlockEnd < 0:
		merged = slices.Insert(merged, labelsBlockEnd, annotationsBlock...)
	case annotationsBlockEnd >= 0 && labelsBlockEnd < 0:
		merged = slices.Insert(merged, annotationsBlockEnd, labelsBlock...)
	case labelsBlockEnd < 0 && annotationsBlockEnd < 0 && metadataLineIndex >= 0:
		bothBlocks := append(annotationsBlock, labelsBlock...)
		merged = slices.Insert(merged, metadataLineIndex+1, bothBlocks...)
	}
	return merged
}

// mergeMetadataMapBlock re-emits the existing labels/annotations block that starts at headerIndex
// and appends a Helm block that merges the matching .Values map, omitting keys the block already
// defines so nothing is duplicated. An empty block (a bare header or an inline "{}") is replaced by
// a guarded block, so the header renders only when the user supplies values and never as a null
// key. It returns the updated output and the index of the last source line it consumed.
func mergeMetadataMapBlock(merged, lines []string, headerIndex int, mapKey, valuePath string) ([]string, int) {
	_, headerIndent := LeadingWhitespace(lines[headerIndex])

	bodyStart := headerIndex + 1
	bodyEnd := bodyStart
	for ; bodyEnd < len(lines); bodyEnd++ {
		trimmed := strings.TrimSpace(lines[bodyEnd])
		_, indent := LeadingWhitespace(lines[bodyEnd])
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && indent <= headerIndent {
			break
		}
	}

	body := lines[bodyStart:bodyEnd]
	existingKeys := extractKeysFromLines(body)
	if len(existingKeys) == 0 {
		merged = append(merged, buildGuardedMetadataMapBlock(headerIndent, mapKey, valuePath)...)
		return merged, bodyEnd - 1
	}

	merged = append(merged, strings.Repeat(" ", headerIndent)+mapKey)
	merged = append(merged, body...)
	valueIndent := strings.Repeat(" ", headerIndent+2)
	merged = appendHelmMapBlock(merged, valueIndent, valuePath, existingKeys)

	return merged, bodyEnd - 1
}

// buildGuardedMetadataMapBlock renders a labels/annotations block wrapped in
// "{{- with <valuePath> }}" so it appears only when the user sets that map. headerIndent is the
// indentation of the header key.
func buildGuardedMetadataMapBlock(headerIndent int, mapKey, valuePath string) []string {
	indent := strings.Repeat(" ", headerIndent)
	valueIndent := strings.Repeat(" ", headerIndent+2)
	return []string{
		indent + "{{- with " + valuePath + " }}",
		indent + mapKey,
		valueIndent + "{{- toYaml . | nindent " + strconv.Itoa(headerIndent+2) + " }}",
		indent + "{{- end }}",
	}
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

	// Matches YAML keys "  key: value" and empty-value keys "  key:" (supports dots, slashes, hyphens)
	keyPattern := regexp.MustCompile(`^\s+([a-zA-Z0-9._/-]+):(\s|$)`)

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
