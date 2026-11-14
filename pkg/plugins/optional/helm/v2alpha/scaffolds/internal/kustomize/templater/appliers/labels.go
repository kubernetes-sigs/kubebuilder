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
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

const (
	valuesServiceAccountLabels      = ".Values.serviceAccount.labels"
	valuesServiceAccountAnnotations = ".Values.serviceAccount.annotations"
	valuesPrometheusLabels          = ".Values.prometheus.labels"
	valuesPrometheusAnnotations     = ".Values.prometheus.annotations"
	valuesCRDLabels                 = ".Values.crd.labels"
	valuesCRDAnnotations            = ".Values.crd.annotations"

	// How many lines to scan around a label for its siblings before giving up.
	labelScanWindow = 10
)

// AddHelmLabelsAndAnnotations replaces kustomize managed-by labels with Helm equivalents.
func AddHelmLabelsAndAnnotations(detectedPrefix, chartName, yamlContent string) string {
	managedByRegex := regexp.MustCompile(`(\s*)app\.kubernetes\.io/managed-by:\s+kustomize`)
	yamlContent = managedByRegex.ReplaceAllString(yamlContent, "${1}app.kubernetes.io/managed-by: {{ .Release.Service }}")

	hardcodedNameLabel := "app.kubernetes.io/name: " + detectedPrefix
	templatedNameLabel := "app.kubernetes.io/name: {{ include \"" + chartName + ".name\" . }}"
	yamlContent = strings.ReplaceAll(yamlContent, hardcodedNameLabel, templatedNameLabel)

	// A matchExpressions requirement on the name label follows the name too, so it keeps matching.
	// The content comes from sigs.k8s.io/yaml: key, operator, then values as a block list.
	nameExpression := regexp.MustCompile(`(?m)(- key: app\.kubernetes\.io/name\n *operator: \w+\n *values:\n` +
		`(?: *- .*\n)*? *- )` + regexp.QuoteMeta(detectedPrefix) + `$`)
	yamlContent = nameExpression.ReplaceAllString(yamlContent, `${1}{{ include "`+chartName+`.name" . }}`)

	return AddStandardHelmLabels(yamlContent)
}

// CheckExistingLabels checks if standard Helm labels already exist in a labels section.
func CheckExistingLabels(lines []string, currentIndex int, indent string) (hasChart, hasInstance, hasManagedBy bool) {
	markFound := func(line string) {
		if strings.Contains(line, common.LabelKeyHelmChart) {
			hasChart = true
		}
		if strings.Contains(line, common.LabelKeyAppInstance) {
			hasInstance = true
		}
		if strings.Contains(line, common.LabelKeyAppManagedBy) {
			hasManagedBy = true
		}
	}

	// Look backward from current position (managed-by often appears before name in kustomize output)
	for j := currentIndex - 1; j >= 0 && j >= currentIndex-labelScanWindow; j-- {
		trimmed := strings.TrimSpace(lines[j])
		lineIndent, _ := LeadingWhitespace(lines[j])

		// Stop if we've moved out of the labels section
		if trimmed == common.YamlKeyLabels {
			break
		}
		if trimmed != "" && len(lineIndent) < len(indent) {
			break
		}
		markFound(lines[j])
	}

	// Look ahead from current position
	for j := currentIndex + 1; j < len(lines) && j < currentIndex+labelScanWindow; j++ {
		trimmed := strings.TrimSpace(lines[j])
		lineIndent, _ := LeadingWhitespace(lines[j])

		// Stop if we've moved to a new section
		if trimmed != "" && len(lineIndent) < len(indent) {
			break
		}
		markFound(lines[j])
	}

	return hasChart, hasInstance, hasManagedBy
}

// AddStandardHelmLabels adds standard Helm labels to all labels sections except selectors.
func AddStandardHelmLabels(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	inBlockScalar := blockScalarLines(lines)
	result := make([]string, 0, len(lines)+10) // slack for the labels added below
	inSelector := false

	for i := range lines {
		line := lines[i]
		result = append(result, line)
		// Lines of a multi-line value are text, not labels.
		if inBlockScalar[i] {
			continue
		}

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

			isInLabelsSection := false
			for j := i - 1; j >= 0 && j >= i-5; j-- {
				if inBlockScalar[j] {
					continue
				}
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

			hasHelmChart, hasInstance, hasManagedBy := CheckExistingLabels(lines, i, indent)

			if !hasHelmChart {
				result = append(result,
					indent+common.LabelKeyHelmChart+` {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}`)
			}

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
func AddServiceAccountLabelsAndAnnotations(chartName, yamlContent string) string {
	return addMetadataLabelsAndAnnotations(
		chartName, yamlContent, valuesServiceAccountLabels, valuesServiceAccountAnnotations, nil)
}

// AddServiceMonitorLabelsAndAnnotations makes the ServiceMonitor metadata honor
// .Values.prometheus.labels and .Values.prometheus.annotations, using the same merge
// semantics as AddServiceAccountLabelsAndAnnotations.
func AddServiceMonitorLabelsAndAnnotations(chartName, yamlContent string) string {
	return addMetadataLabelsAndAnnotations(
		chartName, yamlContent, valuesPrometheusLabels, valuesPrometheusAnnotations, nil)
}

// crdFixedAnnotations are the annotations of a CRD the values do not replace: crd.keep sets the
// resource policy, and controller-gen and the cert-manager injection write the other 2.
var crdFixedAnnotations = []string{
	"helm.sh/resource-policy", "controller-gen.kubebuilder.io/version", "cert-manager.io/inject-ca-from",
}

// AddCRDLabelsAndAnnotations makes CRD metadata honor .Values.crd.labels and .Values.crd.annotations,
// using the same merge semantics as AddServiceAccountLabelsAndAnnotations.
func AddCRDLabelsAndAnnotations(chartName, yamlContent string) string {
	return addMetadataLabelsAndAnnotations(
		chartName, yamlContent, valuesCRDLabels, valuesCRDAnnotations, crdFixedAnnotations)
}

// addMetadataLabelsAndAnnotations merges into the labels and annotations blocks under the resource's
// own (first) metadata, in either order, and injects the missing one, so no key is duplicated.
// Nested "metadata:" keys, such as those in a CRD's OpenAPI schema, are left untouched. The values
// replace the keys the blocks already have, but for the Helm labels and fixedAnnotations.
func addMetadataLabelsAndAnnotations(
	chartName, yamlContent, labelsValuePath, annotationsValuePath string, fixedAnnotations []string,
) string {
	lines := strings.Split(yamlContent, "\n")
	merged := make([]string, 0, len(lines))

	metadataIndent := -1
	metadataLineIndex := -1
	metadataClosed := false
	labelsBlockEnd := -1
	annotationsBlockEnd := -1

	for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
		trimmed := strings.TrimSpace(lines[lineIndex])
		if metadataIndent >= 0 && !metadataClosed && closesMetadataBlock(lines[lineIndex], metadataIndent) {
			metadataClosed = true
		}
		inMetadata := metadataIndent >= 0 && !metadataClosed

		switch {
		case trimmed == common.YamlKeyMetadata && metadataIndent < 0:
			_, metadataIndent = LeadingWhitespace(lines[lineIndex])
			metadataLineIndex = len(merged)
			merged = append(merged, lines[lineIndex])
		case inMetadata && isMetadataMapChildHeader(lines[lineIndex], common.YamlKeyLabels, metadataIndent):
			merged, lineIndex = mergeMetadataMapBlock(
				merged, lines, lineIndex, common.YamlKeyLabels, labelsValuePath, chartName, helmLabels)
			labelsBlockEnd = len(merged)
		case inMetadata && isMetadataMapChildHeader(lines[lineIndex], common.YamlKeyAnnotations, metadataIndent):
			merged, lineIndex = mergeMetadataMapBlock(
				merged, lines, lineIndex, common.YamlKeyAnnotations, annotationsValuePath, chartName, fixedAnnotations)
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
		labelsValuePath, annotationsValuePath, chartName)
	return strings.Join(merged, "\n")
}

// closesMetadataBlock reports whether line is a YAML key at or above metadataIndent, which ends
// the metadata block. Blank lines, comments and Helm actions do not end it.
func closesMetadataBlock(line string, metadataIndent int) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "{{") {
		return false
	}
	_, indent := LeadingWhitespace(line)
	return indent <= metadataIndent
}

// isMetadataMapHeader reports whether trimmed is the header for the given metadata map key
// (for example "labels:"), including the inline empty forms "labels: {}" and "labels: null".
// Callers rewrite the header as a plain "labels:", so an empty form never stays as a second key.
func isMetadataMapHeader(trimmed, mapKey string) bool {
	return trimmed == mapKey || trimmed == mapKey+" {}" || trimmed == mapKey+" null"
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
	labelsValuePath, annotationsValuePath, chartName string,
) []string {
	labelsBlock := buildGuardedMetadataMapBlock(
		childIndent, common.YamlKeyLabels, labelsValuePath, chartName)
	annotationsBlock := buildGuardedMetadataMapBlock(
		childIndent, common.YamlKeyAnnotations, annotationsValuePath, chartName)

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
// and appends a Helm block that merges the matching .Values map. A key set in the values replaces
// the existing one, so each key renders once. An empty block (a bare header or an inline "{}") is
// replaced by a guarded block, so the header renders only when the user supplies values and never
// as a null key. It returns the updated output and the index of the last source line it consumed.
func mergeMetadataMapBlock(
	merged, lines []string, headerIndex int, mapKey, valuePath, chartName string, fixedKeys []string,
) ([]string, int) {
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
	existingKeys := extractMapKeys(body)
	if len(existingKeys) == 0 {
		merged = append(merged, buildGuardedMetadataMapBlock(headerIndent, mapKey, valuePath, chartName)...)
		return merged, bodyEnd - 1
	}

	merged = append(merged, strings.Repeat(" ", headerIndent)+mapKey)
	start := len(merged)
	merged = append(merged, body...)
	valueIndent := strings.Repeat(" ", headerIndent+2)

	merged, existingKeys = letValuesReplace(merged, start, valueIndent, valuePath, chartName, fixedKeys)
	return appendHelmMapBlock(merged, valueIndent, valuePath, chartName, existingKeys), bodyEnd - 1
}

// letValuesReplace makes each entry in result[start:] give way to the same key in the map at
// valuePath, so a key renders once: with the value set for it, or else as config/ has it. An entry
// whose key is in fixedKeys stays as it is. It returns the keys of the entries, which the block
// that renders the rest of the map must omit.
func letValuesReplace(
	result []string, start int, indent, valuePath, chartName string, fixedKeys []string,
) ([]string, []string) {
	body := slices.Clone(result[start:])
	entries, ok := splitMapEntries(body)
	if !ok {
		// Not a map of entries, which kustomize never writes: the values do not replace any key.
		return result, extractMapKeys(body)
	}

	result = append(result[:start], body[:entries[0].start]...)
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.key)
		// The actions that wrap a neighbor entry stay where they are, after this one.
		lines, wrappers := splitTrailingActions(body[entry.start:entry.end])
		switch line, oneLine := valueOrDefaultEntry(lines, entry, valuePath, chartName); {
		case slices.Contains(fixedKeys, entry.key):
			result = append(result, lines...)
		case oneLine:
			result = append(result, line)
		default:
			result = appendUnlessSet(result, lines, entry.key, indent, valuePath, chartName)
		}
		result = append(result, wrappers...)
	}
	return result, keys
}

// splitTrailingActions splits off the lines at the end of an entry that are only a Helm action at
// the indentation of its key, such as the "{{- end }}" and "{{- if ... }}" that wrap a neighbor
// entry.
func splitTrailingActions(lines []string) ([]string, []string) {
	keyIndent, _ := LeadingWhitespace(lines[0])
	end := len(lines)
	for ; end > 1; end-- {
		indent, _ := LeadingWhitespace(lines[end-1])
		trimmed := strings.TrimSpace(lines[end-1])
		if loc := helmTemplateRegex.FindStringIndex(trimmed); indent != keyIndent || loc == nil ||
			loc[0] != 0 || loc[1] != len(trimmed) {
			break
		}
	}
	return lines[:end], lines[end:]
}

// helmLabels are the labels the chart sets for Helm. They are fixed keys of every labels block.
var helmLabels = []string{
	"app.kubernetes.io/managed-by",
	"app.kubernetes.io/name",
	"app.kubernetes.io/instance",
	"helm.sh/chart",
}

// valueOrDefault returns the call to the chart helper that renders the value set for key in the map
// at valuePath as a quoted string, or defaultText when the key is not set or is null.
func valueOrDefault(chartName, valuePath, key, defaultText string) string {
	return `{{ include "` + chartName + `.valueOrDefault" (list ` + valuePath + " " +
		strconv.Quote(key) + " " + strconv.Quote(defaultText) + ") }}"
}

// valueOrDefaultEntry rewrites a "key: value" line so the value set in the values replaces value.
// It returns false when the entry is not one line of plain ASCII text, or when its value holds a
// Helm action, which only renders as it is written.
func valueOrDefaultEntry(lines []string, entry mapEntry, valuePath, chartName string) (string, bool) {
	if len(lines) != 1 || entry.valueColumn <= 0 || entry.valueColumn > len(lines[0]) {
		return "", false
	}
	line := lines[0]
	key, value := line[:entry.valueColumn], line[entry.valueColumn:]
	// The value column counts characters, so it is the byte offset only in an ASCII line.
	if !strings.HasSuffix(strings.TrimSpace(key), ":") || strings.Contains(value, "{{") ||
		strings.ContainsFunc(line, func(r rune) bool { return r > unicode.MaxASCII }) {
		return "", false
	}
	return key + valueOrDefault(chartName, valuePath, entry.key, value), true
}

// appendUnlessSet appends the lines of an existing entry so they render when the map at valuePath
// does not set the same key, and the value set for the key when it does.
func appendUnlessSet(result, lines []string, key, indent, valuePath, chartName string) []string {
	quotedKey := strconv.Quote(key)
	elseAction, endAction := indent+"{{- else }}", indent+"{{- end }}"
	// A blank last line belongs to a |+ value. Actions that do not trim keep it.
	if last := len(lines) - 1; strings.Trim(lines[last], " \t") == "" {
		lines, elseAction, endAction = lines[:last], "{{ else }}", "{{ end }}"
	}

	result = append(result,
		indent+`{{- if kindIs "invalid" (index (`+valuePath+" | default dict) "+quotedKey+") }}")
	result = append(result, lines...)
	return append(result, elseAction,
		indent+quotedKey+": "+valueOrDefault(chartName, valuePath, key, ""), endAction)
}

// helmTemplateRegex matches a {{ }} action, including quoted strings containing {{ }}.
var helmTemplateRegex = regexp.MustCompile(`\{\{(?:"(?:[^"\\]|\\.)*"|[^"}]|\}[^}])*\}\}`)

// parseMap parses body as a YAML map, replacing Helm actions with placeholders.
// It returns nil when body is not a map.
func parseMap(body []string) *yaml.Node {
	cleaned := make([]string, 0, len(body))
	for _, line := range body {
		trimmed := strings.TrimSpace(line)
		if loc := helmTemplateRegex.FindStringIndex(trimmed); loc != nil && loc[0] == 0 && loc[1] == len(trimmed) {
			// A line that is only a Helm action (such as "{{- if ... }}") becomes a comment: a bare
			// placeholder there would break the map, while a comment keeps it valid and, inside a
			// block scalar, stays as content with the same indentation.
			indent, _ := LeadingWhitespace(line)
			cleaned = append(cleaned, indent+"#")
			continue
		}
		cleaned = append(cleaned, helmTemplateRegex.ReplaceAllString(line, "x"))
	}

	var document yaml.Node
	if err := yaml.Unmarshal([]byte(strings.Join(cleaned, "\n")), &document); err != nil ||
		len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	return document.Content[0]
}

// extractMapKeys extracts keys from a YAML map, replacing Helm actions with placeholders.
func extractMapKeys(body []string) []string {
	mapping := parseMap(body)
	if mapping == nil {
		return extractKeysFromLines(body)
	}

	keys := make([]string, 0, len(mapping.Content)/2)
	for i := 0; i < len(mapping.Content); i += 2 {
		keys = append(keys, mapping.Content[i].Value)
	}
	return keys
}

// mapEntry is a key of a YAML map and the lines body[start:end] it covers.
// valueColumn is where the value starts in the line of the key, or 0 when it is not on that line.
type mapEntry struct {
	key         string
	start, end  int
	valueColumn int
}

// splitMapEntries splits a block-style YAML map into its entries. It returns false when two keys
// share a line or body is not a block-style map.
func splitMapEntries(body []string) ([]mapEntry, bool) {
	mapping := parseMap(body)
	if mapping == nil || mapping.Style&yaml.FlowStyle != 0 || len(mapping.Content) == 0 {
		return nil, false
	}

	// YAML also breaks lines on \r, U+0085, U+2028 and U+2029, so map its line numbers to body.
	var bodyLine []int
	for i, line := range body {
		bodyLine = append(bodyLine, i)
		for _, r := range strings.TrimSuffix(line, "\r") {
			if r == '\r' || r == '\u0085' || r == '\u2028' || r == '\u2029' {
				bodyLine = append(bodyLine, i)
			}
		}
	}

	entries := make([]mapEntry, 0, len(mapping.Content)/2)
	for i := 0; i < len(mapping.Content); i += 2 {
		yamlLine := mapping.Content[i].Line - 1
		if yamlLine < 0 || yamlLine >= len(bodyLine) {
			return nil, false
		}
		start := bodyLine[yamlLine]
		if len(entries) > 0 && start <= entries[len(entries)-1].start {
			return nil, false
		}
		entry := mapEntry{key: mapping.Content[i].Value, start: start}
		if value := mapping.Content[i+1]; value.Kind == yaml.ScalarNode && value.Line == mapping.Content[i].Line {
			entry.valueColumn = value.Column - 1
		}
		entries = append(entries, entry)
	}
	for i := range entries {
		entries[i].end = len(body)
		if i+1 < len(entries) {
			entries[i].end = entries[i+1].start
		}
	}
	return entries, true
}

// stringValues returns the action that renders the map a Helm expression gives as labels or
// annotations: the chart's stringMap helper writes each value as a string and leaves out null ones.
func stringValues(chartName, expression string, indentWidth int) string {
	return `{{- include "` + chartName + `.stringMap" ` + expression + " | nindent " + strconv.Itoa(indentWidth) + " }}"
}

// buildGuardedMetadataMapBlock renders a labels/annotations block wrapped in
// "{{- with <valuePath> }}" so it appears only when the user sets that map. headerIndent is the
// indentation of the header key.
func buildGuardedMetadataMapBlock(headerIndent int, mapKey, valuePath, chartName string) []string {
	indent := strings.Repeat(" ", headerIndent)
	valueIndent := strings.Repeat(" ", headerIndent+2)
	return []string{
		indent + "{{- with " + valuePath + " }}",
		indent + mapKey,
		valueIndent + stringValues(chartName, ".", headerIndent+2),
		indent + "{{- end }}",
	}
}

// appendAfterExistingEntries appends block after the existing entries in result. When they end with a
// blank line, it ends a keep-chomped (|+) value: an unindented, untrimmed first action takes its
// place and renders it back, where an indented "{{-" action would eat it.
func appendAfterExistingEntries(result, block []string) []string {
	if last := len(result) - 1; last >= 0 && strings.TrimSpace(result[last]) == "" {
		result = result[:last]
		block[0] = strings.Replace(strings.TrimLeft(block[0], " "), "{{- ", "{{ ", 1)
	}
	return append(result, block...)
}

// quotedKeys returns keys as the arguments of a Helm function, each one a quoted string.
func quotedKeys(keys []string) string {
	quoted := make([]string, len(keys))
	for i, key := range keys {
		quoted[i] = strconv.Quote(key)
	}
	return strings.Join(quoted, " ")
}

// appendHelmMapBlock appends Helm template blocks for custom labels/annotations.
func appendHelmMapBlock(result []string, indent, valuePath, chartName string, existingKeys []string) []string {
	values := indent + stringValues(chartName, ".", len(indent))

	if len(existingKeys) > 0 {
		omitKeys := quotedKeys(existingKeys)
		return appendAfterExistingEntries(result, []string{
			indent + "{{- with " + valuePath + " }}",
			indent + "{{- with omit . " + omitKeys + " }}",
			values,
			indent + "{{- end }}",
			indent + "{{- end }}",
		})
	}

	return append(result,
		indent+"{{- with "+valuePath+" }}",
		values,
		indent+"{{- end }}",
	)
}

// appendNestedHelmMapBlock appends nested Helm template blocks (e.g., .Values.manager.pod -> .labels).
func appendNestedHelmMapBlock(
	result []string, indent, outerPath, innerPath, chartName string, existingKeys []string,
) []string {
	values := indent + stringValues(chartName, ".", len(indent))

	if len(existingKeys) > 0 {
		omitKeys := quotedKeys(existingKeys)
		return appendAfterExistingEntries(result, []string{
			indent + "{{- with " + outerPath + " }}",
			indent + "{{- with " + innerPath + " }}",
			indent + "{{- with omit . " + omitKeys + " }}",
			values,
			indent + "{{- end }}",
			indent + "{{- end }}",
			indent + "{{- end }}",
		})
	}

	return append(result,
		indent+"{{- with "+outerPath+" }}",
		indent+"{{- with "+innerPath+" }}",
		values,
		indent+"{{- end }}",
		indent+"{{- end }}",
	)
}

// extractKeysFromLines extracts YAML keys from labels/annotations sections.
func extractKeysFromLines(lines []string) []string {
	keys := []string{}

	sectionStart := 0
	for i, v := range slices.Backward(lines) {
		trimmed := strings.TrimSpace(v)
		if trimmed == common.YamlKeyLabels || trimmed == common.YamlKeyAnnotations {
			sectionStart = i + 1
			break
		}
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

		if trimmed == common.YamlKeyLabels || trimmed == common.YamlKeyAnnotations ||
			trimmed == common.YamlKeyMetadata || trimmed == common.YamlKeySpec || trimmed == common.YamlKeyTemplate {
			break
		}

		if matches := keyPattern.FindStringSubmatch(line); len(matches) > 1 {
			keys = append(keys, matches[1])
		}
	}

	return keys
}

// SelectorKeys returns the label keys the selector of a Deployment names, in matchLabels and in
// matchExpressions.
func SelectorKeys(deployment *unstructured.Unstructured) []string {
	matchLabels, _, _ := unstructured.NestedStringMap(deployment.Object, "spec", "selector", "matchLabels")
	expressions, _, _ := unstructured.NestedSlice(deployment.Object, "spec", "selector", "matchExpressions")

	keys := slices.Sorted(maps.Keys(matchLabels))
	for _, expression := range expressions {
		if requirement, ok := expression.(map[string]any); ok {
			if key, ok := requirement["key"].(string); ok {
				keys = append(keys, key)
			}
		}
	}
	return keys
}
