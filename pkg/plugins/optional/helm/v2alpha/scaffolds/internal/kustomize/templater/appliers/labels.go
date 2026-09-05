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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

const (
	valuesServiceAccountLabels      = ".Values.serviceAccount.labels"
	valuesServiceAccountAnnotations = ".Values.serviceAccount.annotations"
	valuesPrometheusLabels          = ".Values.prometheus.labels"
	valuesPrometheusAnnotations     = ".Values.prometheus.annotations"
)

// AddHelmLabelsAndAnnotations replaces kustomize managed-by labels with Helm equivalents.
func AddHelmLabelsAndAnnotations(
	detectedPrefix, chartName string, yamlContent string, resource *unstructured.Unstructured,
) string {
	// Replace app.kubernetes.io/managed-by: kustomize with Helm template
	// Use regex to handle different whitespace patterns
	managedByRegex := regexp.MustCompile(`(\s*)app\.kubernetes\.io/managed-by:\s+kustomize`)
	yamlContent = managedByRegex.ReplaceAllString(yamlContent, "${1}app.kubernetes.io/managed-by: {{ .Release.Service }}")

	hardcodedNameLabel := "app.kubernetes.io/name: " + detectedPrefix
	templatedNameLabel := "app.kubernetes.io/name: {{ include \"" + chartName + ".name\" . }}"
	yamlContent = strings.ReplaceAll(yamlContent, hardcodedNameLabel, templatedNameLabel)

	// Add standard Helm labels to labels sections, excluding selectors/matchLabels.
	yamlContent = AddStandardHelmLabels(yamlContent, resource)

	return yamlContent
}

// CheckExistingLabels checks if standard Helm labels already exist in a labels section.
func CheckExistingLabels(lines []string, currentIndex int, indent string) (hasChart, hasInstance, hasManagedBy bool) {
	// Look backward from current position (managed-by often appears before name in kustomize output)
	for j := currentIndex - 1; j >= 0 && j >= currentIndex-10; j-- {
		backLine := lines[j]
		backTrimmed := strings.TrimSpace(backLine)
		backIndent, _ := LeadingWhitespace(backLine)

		// Stop if we've moved out of the labels section
		if backTrimmed == common.YamlKeyLabels {
			break
		}
		if backTrimmed != "" && len(backIndent) < len(indent) {
			break
		}

		if strings.Contains(backLine, common.LabelKeyHelmChart) {
			hasChart = true
		}
		if strings.Contains(backLine, common.LabelKeyAppInstance) {
			hasInstance = true
		}
		if strings.Contains(backLine, common.LabelKeyAppManagedBy) {
			hasManagedBy = true
		}
	}

	// Look ahead from current position
	for j := currentIndex + 1; j < len(lines) && j < currentIndex+10; j++ {
		nextLine := lines[j]
		nextTrimmed := strings.TrimSpace(nextLine)
		nextIndent, _ := LeadingWhitespace(nextLine)

		// Stop if we've moved to a new section
		if nextTrimmed != "" && len(nextIndent) < len(indent) {
			break
		}

		if strings.Contains(nextLine, common.LabelKeyHelmChart) {
			hasChart = true
		}
		if strings.Contains(nextLine, common.LabelKeyAppInstance) {
			hasInstance = true
		}
		if strings.Contains(nextLine, common.LabelKeyAppManagedBy) {
			hasManagedBy = true
		}
	}

	return hasChart, hasInstance, hasManagedBy
}

// AddStandardHelmLabels adds standard Helm labels to all labels sections except selectors.
func AddStandardHelmLabels(yamlContent string, _ *unstructured.Unstructured) string {
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
		if !inSelector && strings.Contains(line, common.LabelKeyAppName) {
			indent, _ := LeadingWhitespace(line)

			// Check if we're in a labels section by looking backwards
			isInLabelsSection := false
			for j := i - 1; j >= 0 && j >= i-5; j-- {
				if strings.TrimSpace(lines[j]) == common.YamlKeyLabels {
					isInLabelsSection = true
					break
				}
				if strings.TrimSpace(lines[j]) == common.YamlKeyMetadata {
					break
				}
			}

			if !isInLabelsSection {
				continue
			}

			// Check if standard labels already exist in this labels section
			hasHelmChart, hasInstance, hasManagedBy := CheckExistingLabels(lines, i, indent)

			// Add helm.sh/chart if it doesn't exist
			if !hasHelmChart {
				result = append(result,
					indent+common.LabelKeyHelmChart+` {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}`)
			}

			// Add app.kubernetes.io/instance if it doesn't exist
			if !hasInstance {
				result = append(result, indent+common.LabelKeyAppInstance+" {{ .Release.Name }}")
			}

			// Add app.kubernetes.io/managed-by if it doesn't exist (per Helm best practices)
			if !hasManagedBy {
				result = append(result, indent+common.LabelKeyAppManagedBy+" {{ .Release.Service }}")
			}
		}
	}

	return strings.Join(result, "\n")
}

// AddServiceAccountLabelsAndAnnotations makes the ServiceAccount metadata honor
// .Values.serviceAccount.labels and .Values.serviceAccount.annotations. It merges into whichever
// labels and annotations blocks Kustomize already emitted, in either order, and injects the block
// that is missing. User-supplied values therefore always render and no metadata key is duplicated.
func AddServiceAccountLabelsAndAnnotations(yamlContent string) string {
	return addMetadataLabelsAndAnnotations(yamlContent, valuesServiceAccountLabels, valuesServiceAccountAnnotations)
}

// AddServiceMonitorLabelsAndAnnotations makes the ServiceMonitor metadata honor
// .Values.prometheus.labels and .Values.prometheus.annotations, using the same merge
// semantics as AddServiceAccountLabelsAndAnnotations.
func AddServiceMonitorLabelsAndAnnotations(yamlContent string) string {
	return addMetadataLabelsAndAnnotations(yamlContent, valuesPrometheusLabels, valuesPrometheusAnnotations)
}

// addMetadataLabelsAndAnnotations merges into whichever labels and annotations blocks Kustomize
// already emitted under metadata, in either order, and injects the block that is missing.
// User-supplied values therefore always render and no metadata key is duplicated.
func addMetadataLabelsAndAnnotations(yamlContent, labelsValuePath, annotationsValuePath string) string {
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
				merged, lines, lineIndex, common.YamlKeyLabels, labelsValuePath)
			labelsBlockEnd = len(merged)
		case isMetadataMapChildHeader(lines[lineIndex], common.YamlKeyAnnotations, metadataIndent):
			merged, lineIndex = mergeMetadataMapBlock(
				merged, lines, lineIndex, common.YamlKeyAnnotations, annotationsValuePath)
			annotationsBlockEnd = len(merged)
		default:
			merged = append(merged, lines[lineIndex])
		}
	}

	childIndent := 2
	if metadataIndent >= 0 {
		childIndent = metadataIndent + 2
	}

	merged = injectMissingMetadataBlocks(
		merged, childIndent, metadataLineIndex, labelsBlockEnd, annotationsBlockEnd,
		labelsValuePath, annotationsValuePath)
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
	labelsValuePath, annotationsValuePath string,
) []string {
	labelsBlock := buildGuardedMetadataMapBlock(
		childIndent, common.YamlKeyLabels, labelsValuePath)
	annotationsBlock := buildGuardedMetadataMapBlock(
		childIndent, common.YamlKeyAnnotations, annotationsValuePath)

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

// appendNestedHelmMapBlock appends nested Helm template blocks (e.g., .Values.manager.pod -> .labels).
func appendNestedHelmMapBlock(
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
