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
	"slices"
	"strconv"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

type metadataPosition int

const (
	positionStart metadataPosition = iota
	positionDeploymentMetadata
	positionAfterDeploymentMetadata
	positionPodMetadata
)

type blockType int

const (
	blockNone blockType = iota
	blockDeploymentLabels
	blockDeploymentAnnotations
	blockPodLabels
	blockPodAnnotations
)

type customFieldsState struct {
	position                metadataPosition
	deploymentMetadataDepth int

	addedLabelsToDeployment      bool
	addedPodLabels               bool
	addedAnnotationsToDeployment bool
	addedPodAnnotations          bool
	hasDeploymentAnnotations     bool

	currentBlock       blockType
	currentBlockIndent int
}

// TemplateDeploymentFields applies all Deployment-specific transformations.
func TemplateDeploymentFields(detectedPrefix, chartName, yamlContent string) string {
	yamlContent = templateReplicas(yamlContent)
	yamlContent = templateImageReference(yamlContent)
	yamlContent = TemplateServiceAccountNameInDeployment(detectedPrefix, chartName, yamlContent)
	yamlContent = templateEnvironmentVariables(yamlContent)
	yamlContent = templateImagePullSecrets(yamlContent)
	yamlContent = templatePodSecurityContext(yamlContent)
	yamlContent = templateContainerSecurityContext(yamlContent)
	yamlContent = templateResources(yamlContent)
	yamlContent = templateSecurityContexts(yamlContent)
	yamlContent = templateVolumeMounts(yamlContent)
	yamlContent = templateVolumes(yamlContent)
	yamlContent = templateControllerManagerArgs(yamlContent)
	yamlContent = templateBasicWithStatement(
		yamlContent,
		"nodeSelector",
		"spec.template.spec",
		".Values.manager.nodeSelector",
	)
	yamlContent = templateBasicWithStatement(
		yamlContent,
		"affinity",
		"spec.template.spec",
		".Values.manager.affinity",
	)
	yamlContent = templateBasicWithStatement(
		yamlContent,
		"tolerations",
		"spec.template.spec",
		".Values.manager.tolerations",
	)

	// Always emit these conditionals so users can enable them in values.yaml without regenerating.
	yamlContent = templateBasicWithStatement(
		yamlContent,
		"strategy",
		"spec",
		".Values.manager.strategy",
	)
	yamlContent = templatePriorityClassName(yamlContent)
	yamlContent = templateBasicWithStatement(
		yamlContent,
		"topologySpreadConstraints",
		"spec.template.spec",
		".Values.manager.topologySpreadConstraints",
	)
	yamlContent = templateTerminationGracePeriodSeconds(yamlContent)

	return yamlContent
}

// isManagerContainerPresent reports whether yamlContent contains the manager container by literal or templated name.
func isManagerContainerPresent(yamlContent string) bool {
	containerName := GetDefaultContainerName(yamlContent)
	hasLiteralName := strings.Contains(yamlContent, "name: "+containerName)
	hasTemplatedName := strings.Contains(yamlContent, `name: {{ include "`) && strings.Contains(yamlContent, `"manager"`)
	return hasLiteralName || hasTemplatedName
}

func templateReplicas(yamlContent string) string {
	if strings.Contains(yamlContent, ".Values.manager.replicas") {
		return yamlContent
	}
	replicasPattern := regexp.MustCompile(`(?m)^(\s*)replicas:\s*\d+\s*$`)
	return replicasPattern.ReplaceAllString(yamlContent, "${1}replicas: {{ .Values.manager.replicas }}")
}

func AddCustomLabelsAndAnnotations(yamlContent string) string {
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
		indent, indentLen := LeadingWhitespace(line)

		// Create missing annotations block if Deployment has none
		if state.position == positionDeploymentMetadata &&
			trimmed == common.YamlKeySpec &&
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

		updateMetadataTracking(state, lines, i, trimmed, indentLen)
		result = append(result, line)

		result = handleDeploymentAnnotations(state, result, line, trimmed, indent, indentLen)
		result = handleDeploymentLabels(state, result, line, trimmed, indentLen)
		result = handlePodAnnotations(state, result, line, trimmed, indent, indentLen)
		result = handlePodLabels(state, result, line, trimmed, indentLen)
	}

	return strings.Join(result, "\n")
}

// managerEnvBlock renders the manager's environment. manager.env is the Kubernetes shape - an
// ordered list of EnvVar, so valueFrom and anything else the API accepts is written there unchanged
// - and manager.envOverrides addresses those entries by name, which is what --set can reach.
//
// The two are merged by name before rendering, so a name appears exactly once in the Deployment
// however many times the sources mention it. That is what Server-Side Apply requires, and it is why
// the list may be taken from kustomize output verbatim.
//
// Order is the list's own: a repeated name keeps its first position and its last value, and a name
// introduced only by an override renders after the list, alphabetically. Kubernetes owns $(VAR)
// expansion - references resolve against the whole container, including envFrom and anything
// injected into the Pod - so the chart reproduces the order it was given and inspects nothing.
func managerEnvBlock(indentStr string) []string {
	childIndent := indentStr + "  "
	childIndentWidth := strconv.Itoa(len(childIndent))

	return []string{
		// Kind, not truthiness: an empty list and an empty map are falsy but perfectly valid, while
		// false/0/"" are shapes to reject rather than treat as unset.
		indentStr + `{{- if not (or (kindIs "invalid" .Values.manager.env) (kindIs "slice" .Values.manager.env)) }}`,
		indentStr + `{{- fail (printf "manager.env must be a list of environment variables, got a %s. ` +
			`Set one variable with --set manager.envOverrides.NAME=value" (kindOf .Values.manager.env)) }}`,
		indentStr + `{{- end }}`,
		indentStr + `{{- if not (or (kindIs "invalid" .Values.manager.envOverrides) ` +
			`(kindIs "map" .Values.manager.envOverrides)) }}`,
		indentStr + `{{- fail (printf "manager.envOverrides must be a map keyed by variable name, got a %s" ` +
			`(kindOf .Values.manager.envOverrides)) }}`,
		indentStr + `{{- end }}`,
		// $order carries the list's order; $byName carries the entry each name resolves to.
		indentStr + `{{- $byName := dict }}`,
		indentStr + `{{- $order := list }}`,
		indentStr + `{{- range $entry := .Values.manager.env }}`,
		indentStr + `{{- if not (kindIs "map" $entry) }}`,
		indentStr + `{{- fail (printf "manager.env entries must be maps with a name, got a %s" (kindOf $entry)) }}`,
		indentStr + `{{- end }}`,
		indentStr + `{{- $name := get $entry "name" }}`,
		indentStr + `{{- if not (and (kindIs "string" $name) $name) }}`,
		indentStr + `{{- fail "every manager.env entry needs a non-empty name" }}`,
		indentStr + `{{- end }}`,
		// First position, last value: the position a reader sees, the value Kubernetes would use.
		indentStr + `{{- if not (hasKey $byName $name) }}`,
		indentStr + `{{- $order = append $order $name }}`,
		indentStr + `{{- end }}`,
		indentStr + `{{- $_ := set $byName $name $entry }}`,
		indentStr + `{{- end }}`,
		// Presence, not truthiness: --set manager.envOverrides.NAME=null leaves the key with a nil
		// value, and that tombstone is the only way the chart learns the variable was removed.
		indentStr + `{{- range $name, $value := .Values.manager.envOverrides }}`,
		indentStr + `{{- if kindIs "invalid" $value }}`,
		indentStr + `{{- $_ := unset $byName $name }}`,
		indentStr + `{{- else if or (kindIs "map" $value) (kindIs "slice" $value) }}`,
		indentStr + `{{- fail (printf "manager.envOverrides.%s must be a scalar or null, got a %s. ` +
			`A valueFrom entry belongs on manager.env" $name (kindOf $value)) }}`,
		indentStr + `{{- else }}`,
		// An override replaces the whole entry, so a scalar over a valueFrom leaves no source behind.
		indentStr + `{{- $_ := set $byName $name (dict "name" $name "value" (toString $value)) }}`,
		indentStr + `{{- end }}`,
		indentStr + `{{- end }}`,
		indentStr + `{{- $envVars := list }}`,
		indentStr + `{{- range $name := $order }}`,
		indentStr + `{{- if hasKey $byName $name }}`,
		indentStr + `{{- $envVars = append $envVars (get $byName $name) }}`,
		indentStr + `{{- end }}`,
		indentStr + `{{- end }}`,
		// Names the list never mentioned follow it, alphabetically, so additions are deterministic.
		indentStr + `{{- range $name := (keys $byName | sortAlpha) }}`,
		indentStr + `{{- if not (has $name $order) }}`,
		indentStr + `{{- $envVars = append $envVars (get $byName $name) }}`,
		indentStr + `{{- end }}`,
		indentStr + `{{- end }}`,
		// `with` owns the env: key, so a chart whose project declares no variables renders no env
		// block at all while still accepting --set manager.envOverrides.NAME=value.
		indentStr + `{{- with $envVars }}`,
		indentStr + "env:",
		childIndent + "{{- toYaml . | nindent " + childIndentWidth + " }}",
		indentStr + `{{- end }}`,
	}
}

// envBlockMarker is a generated action, so detecting it cannot be confused with a source Deployment
// that merely mentions .Values.manager.env in a value, an arg, or an annotation. It is matched as a
// whole line at the container's child indent - see hasManagerEnvBlock - never as a substring.
const envBlockMarker = `{{- $envVars := list }}`

// envBlockMarkerEscaped is what EscapeExistingTemplateSyntax makes of the marker. Escaping runs
// before the appliers, so a pass over already-generated output sees the escaped form; without this
// the block would be generated a second time. Derived, so the two cannot drift apart.
var envBlockMarkerEscaped = EscapeExistingTemplateSyntax(envBlockMarker)

// blockValueEnd returns the first line past a block-form value - a nested sequence or mapping -
// given the indent of the field that owns it. The block ends at the first non-blank line that is
// outdented past the field, or sits at the field's own indent without being a sequence item.
//
// A blank line is never a boundary by itself: inside a block scalar it is content, and ending there
// would truncate the value and leave its tail behind at an indent nothing owns. Blank lines are
// therefore skipped, and the returned position tracks the last line that actually belonged to the
// block - so blanks between two owned lines are included, while blanks trailing the block (the
// empty element every newline-terminated document splits into, among them) are left alone.
func blockValueEnd(lines []string, from, indentLen int) int {
	end := from
	for i := from; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		lineIndent := len(lines[i]) - len(strings.TrimLeft(lines[i], " \t"))
		if lineIndent < indentLen {
			break
		}
		if lineIndent == indentLen && !strings.HasPrefix(trimmed, "-") {
			break
		}
		end = i + 1
	}
	return end
}

// templateEnvironmentVariables replaces the manager container's env declaration with the templated
// block, in whichever form the serializer produced it: a block list, or an inline `env: []`/`env:
// null`, either of which may be folded onto the sequence dash. When the container declares no env
// at all - the default Go scaffold does not - the block is appended as the container's last field,
// so --set manager.env.NAME=value works on every generated chart rather than only on projects that
// already ship a variable.
func templateEnvironmentVariables(yamlContent string) string {
	if !isManagerContainerPresent(yamlContent) || hasManagerEnvBlock(yamlContent) {
		return yamlContent
	}

	if field, ok := FindManagerField(yamlContent, "env"); ok {
		return ReplaceManagerField(yamlContent, field, managerEnvBlock(field.Indent))
	}

	rangeStart, rangeEnd := FindManagerContainerRange(yamlContent)
	if rangeStart < 0 || rangeEnd < rangeStart {
		return yamlContent
	}

	// Append as the container's last field. Anchoring on a named field is not safe: the serializer
	// sorts keys, and a field whose value is a nested list would swallow the block.
	lines := strings.Split(yamlContent, "\n")
	dashIndent, _ := LeadingWhitespace(lines[rangeStart])
	at := min(rangeEnd+1, len(lines))

	newLines := append([]string{}, lines[:at]...)
	newLines = append(newLines, managerEnvBlock(dashIndent+"  ")...)
	newLines = append(newLines, lines[at:]...)
	return strings.Join(newLines, "\n")
}

func templateResources(yamlContent string) string {
	field, ok := FindManagerField(yamlContent, "resources")
	if !ok {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	if field.Line+1 < len(lines) && strings.Contains(lines[field.Line+1], ".Values.manager.resources") {
		return yamlContent
	}

	childIndent := field.Indent + "  "
	childIndentWidth := strconv.Itoa(len(childIndent))

	return ReplaceManagerField(yamlContent, field, []string{
		field.Indent + "resources:",
		childIndent + "{{- if .Values.manager.resources }}",
		childIndent + "{{- toYaml .Values.manager.resources | nindent " + childIndentWidth + " }}",
		childIndent + "{{- else }}",
		childIndent + "{}",
		childIndent + "{{- end }}",
	})
}

func templateSecurityContexts(yamlContent string) string {
	return yamlContent
}

func templateVolumeMounts(yamlContent string) string {
	return appendToListFromValues(yamlContent, "volumeMounts:", ".Values.manager.extraVolumeMounts")
}

func templateVolumes(yamlContent string) string {
	return appendToListFromValues(yamlContent, "volumes:", ".Values.manager.extraVolumes")
}

// appendToListFromValues injects a values reference into a YAML list field.
// Replaces "key: []" with a conditional template; appends to "key:" with existing items.
func appendToListFromValues(yamlContent string, keyColon string, valuesPath string) string {
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
		indentStr, indentLen := LeadingWhitespace(lines[i])
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

		// Deliberately not blockValueEnd. This end is an insertion point, not the end of a range
		// being replaced: getting it wrong moves the block within a list whose order is not
		// significant, where the replacing callers would strand content instead. It also stops at
		// items sitting at the field's own indent, which is what puts the block ahead of the
		// scaffolded entries; adopting the shared rule would reorder every generated chart for no
		// behavioural gain.
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

// templateImagePullSecrets injects imagePullSecrets; always emits the block so users can enable it in values.yaml.
func templateImagePullSecrets(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")

	if strings.Contains(yamlContent, "imagePullSecrets:") {
		for i := range lines {
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), "imagePullSecrets:") {
				continue
			}

			if i+1 < len(lines) && strings.Contains(lines[i+1], ".Values.manager.imagePullSecrets") {
				return yamlContent
			}

			indentStr, indentLen := LeadingWhitespace(lines[i])
			end := blockValueEnd(lines, i+1, indentLen)

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

	var insertAt int
	foundTemplate := false
	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == common.YamlKeyTemplate {
			foundTemplate = true
			continue
		}
		if foundTemplate && trimmed == common.YamlKeySpec {
			insertAt = i + 1
			break
		}
	}

	if insertAt == 0 || insertAt >= len(lines) {
		return yamlContent
	}

	_, indentLen := LeadingWhitespace(lines[insertAt])
	indentStr := strings.Repeat(" ", indentLen)
	childIndent := indentStr + "  "
	childIndentWidth := strconv.Itoa(len(childIndent))

	block := []string{
		indentStr + "{{- with .Values.manager.imagePullSecrets }}",
		indentStr + "imagePullSecrets:",
		childIndent + "{{- toYaml . | nindent " + childIndentWidth + " }}",
		indentStr + "{{- end }}",
	}

	newLines := append([]string{}, lines[:insertAt]...)
	newLines = append(newLines, block...)
	newLines = append(newLines, lines[insertAt:]...)
	return strings.Join(newLines, "\n")
}

func templatePodSecurityContext(yamlContent string) string {
	if !strings.Contains(yamlContent, "securityContext:") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != "securityContext:" {
			continue
		}

		indentStr, indentLen := LeadingWhitespace(lines[i])
		end := blockValueEnd(lines, i+1, indentLen)

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

// templateContainerSecurityContext templates the manager container's securityContext. The pod's own
// securityContext is a different field: it lives outside the container item, so the locator cannot
// reach it and no disambiguation is needed here.
func templateContainerSecurityContext(yamlContent string) string {
	field, ok := FindManagerField(yamlContent, "securityContext")
	if !ok {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	lookAheadEnd := min(field.Line+6, len(lines))
	if strings.Contains(strings.Join(lines[field.Line:lookAheadEnd], "\n"), ".Values.manager.securityContext") {
		return yamlContent
	}

	childIndent := field.Indent + "  "
	childIndentWidth := strconv.Itoa(len(childIndent))

	return ReplaceManagerField(yamlContent, field, []string{
		field.Indent + "securityContext:",
		childIndent + "{{- if .Values.manager.securityContext }}",
		childIndent + "{{- toYaml .Values.manager.securityContext | nindent " + childIndentWidth + " }}",
		childIndent + "{{- else }}",
		childIndent + "{}",
		childIndent + "{{- end }}",
	})
}

// templateControllerManagerArgs rewrites the manager's argument list so the flags the chart exposes
// as values become conditionals on those values, and everything else reaches the container through
// .Values.manager.args.
//
// Line space throughout. Locating the field structurally and then replacing it with a regexp meant
// carrying line indices and two sets of byte offsets at once, and asking whether a match began on
// the field's own line by looking for a newline in the text before it.
//
// ReplaceManagerField is deliberately not used: it re-emits a folded field's dash on a line of its
// own, which is right for a block that opens with a Helm action but would rewrite the `- args:` of
// every generated chart for no gain. Keeping the declaration line verbatim leaves it folded.
func templateControllerManagerArgs(yamlContent string) string {
	field, ok := FindManagerField(yamlContent, "args")
	// An inline `args: []` has no items to rewrite, and the values-driven block would be the whole
	// list rather than an addition to it.
	if !ok || field.Inline {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	itemsEnd := blockValueEnd(lines, field.Line+1, len(field.Indent))
	items := lines[field.Line+1 : itemsEnd]
	if len(items) == 0 {
		return yamlContent
	}
	if strings.Contains(strings.Join(lines[field.Line:itemsEnd], "\n"), ".Values.manager.args") {
		return yamlContent
	}

	out := append([]string{}, lines[:field.Line+1]...)
	out = append(out, managerArgsBlock(items)...)
	out = append(out, lines[itemsEnd:]...)
	return strings.Join(out, "\n")
}

// managerArgsBlock rewrites the manager's argument items. The three flags the chart owns as values
// are re-emitted under their own conditionals, the certificate paths are carried through for the
// cert-manager conditionals a later applier wraps them in, and every other argument is dropped in
// favour of .Values.manager.args.
func managerArgsBlock(items []string) []string {
	// Sequence items of one list share an indent, so the first item's is the block's.
	itemIndent, _ := LeadingWhitespace(items[0])

	var (
		metricsLine    string
		healthLine     string
		webhookLine    string
		preservedLines []string
	)

	for _, rawLine := range items {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		switch {
		case strings.Contains(trimmed, "--metrics-bind-address"):
			metricsLine = line
		case strings.Contains(trimmed, "--health-probe-bind-address"):
			healthLine = line
		case strings.Contains(trimmed, "--webhook-port"):
			webhookLine = line
		case strings.Contains(trimmed, "--webhook-cert-path"),
			strings.Contains(trimmed, "--metrics-cert-path"):
			preservedLines = append(preservedLines, line)
		default:
			// Remaining args will be handled through values.yaml
		}
	}

	block := make([]string, 0, len(items)+12)
	if metricsLine != "" {
		block = append(block,
			itemIndent+"{{- if .Values.metrics.enabled }}",
			metricsLine,
			itemIndent+"{{- if not .Values.metrics.secure }}",
			itemIndent+"- --metrics-secure=false",
			itemIndent+"{{- end }}",
			itemIndent+"{{- else }}",
			itemIndent+"# Bind to :0 to disable the controller-runtime managed metrics server",
			itemIndent+"- --metrics-bind-address=0",
			itemIndent+"{{- end }}",
		)
	}
	if healthLine != "" {
		block = append(block, healthLine)
	}
	if webhookLine != "" {
		block = append(block,
			itemIndent+"{{- if .Values.webhook.enabled }}",
			webhookLine,
			itemIndent+"{{- end }}",
		)
	}

	block = append(block,
		itemIndent+"{{- range .Values.manager.args }}",
		itemIndent+"- {{ tpl . $ }}",
		itemIndent+"{{- end }}",
	)

	return append(block, preservedLines...)
}

func templateImageReference(yamlContent string) string {
	field, ok := FindManagerField(yamlContent, "image")
	if !ok {
		return yamlContent
	}
	if strings.Contains(strings.Split(yamlContent, "\n")[field.Line], ".Values.manager.image.repository") {
		return yamlContent
	}

	// The generated block renders imagePullPolicy from values, so the scaffolded field has to go
	// first: dropping it afterwards would find the generated line instead. Removing it shifts the
	// image field, hence the second lookup.
	if pullPolicy, found := FindManagerField(yamlContent, "imagePullPolicy"); found {
		yamlContent = ReplaceManagerField(yamlContent, pullPolicy, nil)
		if field, ok = FindManagerField(yamlContent, "image"); !ok {
			return yamlContent
		}
	}

	indentStr := field.Indent
	return ReplaceManagerField(yamlContent, field, []string{
		indentStr + "image: \"{{ .Values.manager.image.repository | default \"controller\" }}" +
			"{{- if not (contains \"@\" (.Values.manager.image.repository | default \"controller\")) }}" +
			":{{ .Values.manager.image.tag | default .Chart.AppVersion }}{{- end }}\"",
		indentStr + "{{- with .Values.manager.image.pullPolicy }}",
		indentStr + "imagePullPolicy: {{ . }}",
		indentStr + "{{- end }}",
	})
}

func templateBasicWithStatement(
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
		pKeyParts := strings.Split(parentKey, ".")
		pKeyIdx := 0
		pKeyInit := false
		currIndent := 0
		for i := range len(lines) {
			_, lineIndent := LeadingWhitespace(lines[i])
			if pKeyInit && lineIndent <= currIndent {
				return yamlContent
			}
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), pKeyParts[pKeyIdx]) {
				continue
			}

			pKeyIdx++
			pKeyInit = true
			if pKeyIdx >= len(pKeyParts) {
				start = i + 1
				end = start
				break
			}
		}
		_, indentLen = LeadingWhitespace(lines[start])
	} else {
		for i := range len(lines) {
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), yamlKey) {
				continue
			}
			start = i
			end = i + 1
			trimmed := strings.TrimSpace(lines[i])
			if len(trimmed) == len(yamlKey) {
				_, indentLenSearch := LeadingWhitespace(lines[i])
				end = len(lines)
				for j := i + 1; j < len(lines); j++ {
					trimmedJ := strings.TrimSpace(lines[j])
					_, indentLenLine := LeadingWhitespace(lines[j])
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
			break
		}
		_, indentLen = LeadingWhitespace(lines[start])
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

func templatePriorityClassName(yamlContent string) string {
	if strings.Contains(yamlContent, ".Values.manager.priorityClassName") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")

	if strings.Contains(yamlContent, "priorityClassName:") {
		pattern := regexp.MustCompile(`(?m)^(\s*)priorityClassName:\s*"?([^"\n]*)"?\s*$`)
		yamlContent = pattern.ReplaceAllString(yamlContent,
			"${1}{{- with .Values.manager.priorityClassName }}\n"+
				"${1}priorityClassName: {{ . | quote }}\n"+
				"${1}{{- end }}")
		return yamlContent
	}

	var insertAt int
	foundTemplate := false
	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == common.YamlKeyTemplate {
			foundTemplate = true
			continue
		}
		if foundTemplate && trimmed == common.YamlKeySpec {
			insertAt = i + 1
			break
		}
	}

	if insertAt == 0 || insertAt >= len(lines) {
		return yamlContent
	}

	_, indentLen := LeadingWhitespace(lines[insertAt])
	indentStr := strings.Repeat(" ", indentLen)

	block := []string{
		indentStr + "{{- with .Values.manager.priorityClassName }}",
		indentStr + "priorityClassName: {{ . | quote }}",
		indentStr + "{{- end }}",
	}

	newLines := append([]string{}, lines[:insertAt]...)
	newLines = append(newLines, block...)
	newLines = append(newLines, lines[insertAt:]...)
	return strings.Join(newLines, "\n")
}

// templateTerminationGracePeriodSeconds injects terminationGracePeriodSeconds; uses hasKey to allow 0 values.
func templateTerminationGracePeriodSeconds(yamlContent string) string {
	if strings.Contains(yamlContent, ".Values.manager.terminationGracePeriodSeconds") {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")

	if strings.Contains(yamlContent, "terminationGracePeriodSeconds:") {
		pattern := regexp.MustCompile(`(?m)^(\s*)terminationGracePeriodSeconds:\s*\d+\s*$`)
		yamlContent = pattern.ReplaceAllString(yamlContent,
			"${1}{{- if and (hasKey .Values.manager \"terminationGracePeriodSeconds\") "+
				"(ne .Values.manager.terminationGracePeriodSeconds nil) }}\n"+
				"${1}terminationGracePeriodSeconds: {{ .Values.manager.terminationGracePeriodSeconds }}\n"+
				"${1}{{- end }}")
		return yamlContent
	}

	var insertAt int
	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "serviceAccountName:") {
			insertAt = i + 1
			break
		}
	}

	if insertAt == 0 || insertAt >= len(lines) {
		return yamlContent
	}

	_, indentLen := LeadingWhitespace(lines[insertAt-1])
	indentStr := strings.Repeat(" ", indentLen)

	block := []string{
		indentStr + "{{- if and (hasKey .Values.manager \"terminationGracePeriodSeconds\") " +
			"(ne .Values.manager.terminationGracePeriodSeconds nil) }}",
		indentStr + "terminationGracePeriodSeconds: {{ .Values.manager.terminationGracePeriodSeconds }}",
		indentStr + "{{- end }}",
	}

	newLines := append([]string{}, lines[:insertAt]...)
	newLines = append(newLines, block...)
	newLines = append(newLines, lines[insertAt:]...)
	return strings.Join(newLines, "\n")
}

func handleDeploymentAnnotations(
	state *customFieldsState, result []string, line, trimmed, indent string, indentLen int,
) []string {
	if state.position == positionDeploymentMetadata &&
		state.currentBlock == blockNone &&
		(trimmed == common.YamlKeyAnnotations || strings.HasPrefix(trimmed, common.YamlKeyAnnotations)) {
		state.hasDeploymentAnnotations = true
		state.currentBlock = blockDeploymentAnnotations
		state.currentBlockIndent = indentLen
		return handleFlowStyleAnnotations(result, line, indent)
	}

	if shouldInjectDeploymentAnnotations(state, trimmed, indentLen) {
		result = result[:len(result)-1]

		existingKeys := extractKeysFromLines(result)
		parentIndent := strings.Repeat(" ", state.currentBlockIndent)
		childIndent := detectChildIndent(result, parentIndent)

		if len(existingKeys) == 0 {
			result = result[:len(result)-1]
			childIndentWidth := strconv.Itoa(len(childIndent))
			result = append(result,
				parentIndent+"{{- if .Values.manager.annotations }}",
				parentIndent+"annotations:",
				childIndent+"{{- toYaml .Values.manager.annotations | nindent "+childIndentWidth+" }}",
				parentIndent+"{{- end }}",
			)
		} else {
			result = injectDeploymentAnnotations(result, childIndent)
		}

		result = append(result, line)
		state.addedAnnotationsToDeployment = true
		state.currentBlock = blockNone
	}

	return result
}

func handlePodAnnotations(
	state *customFieldsState, result []string, line, trimmed, indent string, indentLen int,
) []string {
	if state.position == positionPodMetadata &&
		state.currentBlock == blockNone &&
		(trimmed == common.YamlKeyAnnotations || strings.HasPrefix(trimmed, common.YamlKeyAnnotations)) {
		state.currentBlock = blockPodAnnotations
		state.currentBlockIndent = indentLen
		return handleFlowStyleAnnotations(result, line, indent)
	}

	if shouldInjectPodAnnotations(state, trimmed, indentLen) {
		result = result[:len(result)-1]

		existingKeys := extractKeysFromLines(result)
		parentIndent := strings.Repeat(" ", state.currentBlockIndent)
		childIndent := detectChildIndent(result, parentIndent)

		if len(existingKeys) == 0 {
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
			result = addPodAnnotations(result, childIndent)
		}

		result = append(result, line)
		state.addedPodAnnotations = true
		state.currentBlock = blockNone
	}

	if state.position == positionPodMetadata && !state.addedPodAnnotations && trimmed == common.YamlKeyLabels {
		result = result[:len(result)-1]
		result = append(result, indent+"{{- with .Values.manager.pod }}")
		result = append(result, indent+"{{- with .annotations }}")
		result = append(result, indent+"annotations:")
		result = addPodAnnotations(result, indent+"  ")
		result = append(result, indent+"{{- end }}")
		result = append(result, indent+"{{- end }}")
		result = append(result, indent+common.YamlKeyLabels)
		state.addedPodAnnotations = true
	}

	return result
}

func shouldInjectDeploymentAnnotations(
	state *customFieldsState, trimmed string, indentLen int,
) bool {
	return (state.position == positionDeploymentMetadata || state.position == positionAfterDeploymentMetadata) &&
		state.currentBlock == blockDeploymentAnnotations &&
		!state.addedAnnotationsToDeployment &&
		indentLen <= state.currentBlockIndent &&
		trimmed != "" &&
		trimmed != common.YamlKeyAnnotations &&
		!strings.HasPrefix(trimmed, common.YamlKeyAnnotations+" {")
}

func shouldInjectPodAnnotations(state *customFieldsState, trimmed string, indentLen int) bool {
	return (state.position == positionPodMetadata || state.position == positionAfterDeploymentMetadata) &&
		state.currentBlock == blockPodAnnotations &&
		!state.addedPodAnnotations &&
		indentLen <= state.currentBlockIndent &&
		trimmed != "" &&
		trimmed != common.YamlKeyAnnotations &&
		!strings.HasPrefix(trimmed, common.YamlKeyAnnotations+" {")
}

func handleDeploymentLabels(
	state *customFieldsState, result []string, line, trimmed string, indentLen int,
) []string {
	if state.position == positionDeploymentMetadata &&
		state.currentBlock == blockNone &&
		trimmed == common.YamlKeyLabels {
		state.currentBlock = blockDeploymentLabels
		state.currentBlockIndent = indentLen
		return result
	}

	if shouldInjectDeploymentLabels(state, trimmed, indentLen) {
		result = result[:len(result)-1]
		parentIndent := strings.Repeat(" ", state.currentBlockIndent)
		childIndent := detectChildIndent(result, parentIndent)
		result = injectDeploymentLabels(result, childIndent)
		result = append(result, line)
		state.addedLabelsToDeployment = true
		state.currentBlock = blockNone
	}

	return result
}

// handlePodLabels handles injection of custom Pod template labels.
func handlePodLabels(
	state *customFieldsState, result []string, line, trimmed string, indentLen int,
) []string {
	if state.position == positionPodMetadata &&
		state.currentBlock == blockNone &&
		trimmed == common.YamlKeyLabels {
		state.currentBlock = blockPodLabels
		state.currentBlockIndent = indentLen
		return result
	}

	if shouldInjectPodLabels(state, trimmed, indentLen) {
		result = result[:len(result)-1]
		parentIndent := strings.Repeat(" ", state.currentBlockIndent)
		childIndent := detectChildIndent(result, parentIndent)
		result = injectPodLabels(result, childIndent)
		result = append(result, line)
		state.addedPodLabels = true
		state.currentBlock = blockNone
	}

	return result
}

// shouldInjectDeploymentLabels checks if we should inject Deployment labels.
func shouldInjectDeploymentLabels(
	state *customFieldsState, trimmed string, indentLen int,
) bool {
	return (state.position == positionDeploymentMetadata || state.position == positionAfterDeploymentMetadata) &&
		state.currentBlock == blockDeploymentLabels &&
		!state.addedLabelsToDeployment &&
		indentLen <= state.currentBlockIndent &&
		trimmed != "" &&
		trimmed != common.YamlKeyLabels
}

// shouldInjectPodLabels checks if we should inject Pod labels.
func shouldInjectPodLabels(
	state *customFieldsState, trimmed string, indentLen int,
) bool {
	return (state.position == positionPodMetadata || state.position == positionAfterDeploymentMetadata) &&
		state.currentBlock == blockPodLabels &&
		!state.addedPodLabels &&
		indentLen <= state.currentBlockIndent &&
		trimmed != "" &&
		trimmed != common.YamlKeyLabels
}

func injectDeploymentLabels(result []string, childIndent string) []string {
	existingKeys := extractKeysFromLines(result)
	return appendHelmMapBlock(result, childIndent, ".Values.manager.labels", existingKeys)
}

func injectPodLabels(result []string, childIndent string) []string {
	existingKeys := extractKeysFromLines(result)
	return appendNestedHelmMapBlock(result, childIndent, ".Values.manager.pod", ".labels", existingKeys)
}

func injectDeploymentAnnotations(result []string, indent string) []string {
	existingKeys := extractKeysFromLines(result)
	return appendHelmMapBlock(result, indent, ".Values.manager.annotations", existingKeys)
}

func addPodAnnotations(result []string, indent string) []string {
	existingKeys := extractKeysFromLines(result)
	return appendNestedHelmMapBlock(result, indent, ".Values.manager.pod", ".annotations", existingKeys)
}

func handleFlowStyleAnnotations(
	result []string, line string, indent string,
) []string {
	trimmed := strings.TrimSpace(line)

	// Detect flow-style annotations: annotations:{} or annotations: {}
	flowPattern := regexp.MustCompile(`annotations:\s*\{`)
	if !flowPattern.MatchString(trimmed) {
		return result
	}

	// Extract the flow-style content
	annotationsStart := strings.Index(line, common.YamlKeyAnnotations)
	if annotationsStart == -1 {
		return result
	}

	// Find the content after "annotations: "
	contentStart := annotationsStart + len(common.YamlKeyAnnotations)
	flowContent := strings.TrimSpace(line[contentStart:])

	// Remove the flow-style line we just added
	result = result[:len(result)-1]

	// Add block-style annotations: key
	result = append(result, indent+common.YamlKeyAnnotations)

	// Parse and convert flow-style entries to block-style
	if strings.HasPrefix(flowContent, "{") && strings.HasSuffix(flowContent, "}") {
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

	return result
}

// Helper functions for custom labels/annotations injection

// updateMetadataTracking updates the position state as we traverse the YAML structure.
func updateMetadataTracking(
	state *customFieldsState, lines []string, i int, trimmed string, indentLen int,
) {
	// Track Deployment metadata section
	if trimmed == common.YamlKeyMetadata && i > 0 {
		prevLine := strings.TrimSpace(lines[i-1])
		if strings.HasPrefix(prevLine, "kind: Deployment") || prevLine == "kind: Deployment" {
			state.position = positionDeploymentMetadata
			state.deploymentMetadataDepth = indentLen
		} else if prevLine == common.YamlKeyTemplate {
			// Track Pod template metadata section
			state.position = positionPodMetadata
		}
	}

	// Exit deployment metadata when we reach spec:
	if state.position == positionDeploymentMetadata &&
		trimmed == common.YamlKeySpec && indentLen == state.deploymentMetadataDepth {
		state.position = positionAfterDeploymentMetadata
	}

	// Exit pod template metadata when we reach spec: (pod spec)
	if state.position == positionPodMetadata && trimmed == common.YamlKeySpec {
		state.position = positionAfterDeploymentMetadata
	}
}

// detectChildIndent detects the actual child indentation from existing entries in the current block.
func detectChildIndent(lines []string, parentIndent string) string {
	// Scan backwards to find the first child entry with indentation > parent
	parentIndentLen := len(parentIndent)

	for _, v := range slices.Backward(lines) {
		line := v
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and Helm template directives
		if trimmed == "" || strings.HasPrefix(trimmed, "{{") {
			continue
		}

		// Stop at section headers
		if trimmed == common.YamlKeyLabels || trimmed == common.YamlKeyAnnotations ||
			trimmed == common.YamlKeyMetadata || trimmed == common.YamlKeySpec || trimmed == common.YamlKeyTemplate {
			break
		}

		// Find a line with indentation greater than parent (a child entry)
		indent, indentLen := LeadingWhitespace(line)
		if indentLen > parentIndentLen && strings.Contains(line, ":") {
			return indent
		}
	}

	// Default to 2-space indentation (sigs.k8s.io/yaml standard)
	return parentIndent + "  "
}

// MakeContainerArgsConditional makes webhook-cert-path and metrics-cert-path args conditional.
func MakeContainerArgsConditional(yamlContent string) string {
	// Make webhook-cert-path arg conditional on certManager.enabled
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
			return fmt.Sprintf("%s{{- if .Values.certManager.enabled }}\n%s%s\n%s{{- end }}",
				indent, indent, argLine, indent)
		})
	}

	// Make metrics-cert-path arg conditional on certManager.enabled AND metrics.enabled AND metrics.secure
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
			return fmt.Sprintf(
				"%s{{- if and .Values.certManager.enabled .Values.metrics.enabled .Values.metrics.secure }}\n%s%s\n%s{{- end }}",
				indent, indent, argLine, indent)
		})
	}

	return yamlContent
}

// MakeWebhookVolumesConditional makes webhook volumes conditional on certManager.enabled.
func MakeWebhookVolumesConditional(yamlContent string) string {
	// Make webhook volumes conditional on certManager.enabled
	if strings.Contains(yamlContent, "webhook-certs") && strings.Contains(yamlContent, "secretName: webhook-server-cert") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		volumePattern := regexp.MustCompile(`([ \t]+)-\s*name:\s*webhook-certs[\s\S]*?secretName:\s*webhook-server-cert`)
		yamlContent = volumePattern.ReplaceAllStringFunc(yamlContent, MakeYamlContent)
	}

	return yamlContent
}

// MakeWebhookVolumeMountsConditional makes webhook volumeMounts conditional on certManager.enabled.
func MakeWebhookVolumeMountsConditional(yamlContent string) string {
	// Make webhook volumeMounts conditional on certManager.enabled
	webhookCertsPath := "/tmp/k8s-webhook-server/serving-certs"
	if strings.Contains(yamlContent, "webhook-certs") && strings.Contains(yamlContent, webhookCertsPath) {
		// Match only spaces/tabs for indent to avoid consuming the newline
		mountPattern := regexp.MustCompile(
			`([ \t]+)-\s*mountPath:\s*/tmp/k8s-webhook-server/serving-certs[\s\S]*?readOnly:\s*true`)
		yamlContent = mountPattern.ReplaceAllStringFunc(yamlContent, MakeYamlContent)
	}

	return yamlContent
}

// MakeMetricsVolumesConditional wraps metrics volumes with the metrics TLS conditional.
func MakeMetricsVolumesConditional(yamlContent string) string {
	if strings.Contains(yamlContent, "metrics-certs") && strings.Contains(yamlContent, "secretName: metrics-server-cert") {
		// [ \t]+ matches only spaces/tabs so the newline is not consumed by the regexp.
		pattern := regexp.MustCompile(`([ \t]+)-\s*name:\s*metrics-certs[\s\S]*?secretName:\s*metrics-server-cert`)
		yamlContent = wrapWithMetricsTLSConditional(pattern, yamlContent)
	}
	return yamlContent
}

// MakeMetricsVolumeMountsConditional wraps metrics volumeMounts with the metrics TLS conditional.
func MakeMetricsVolumeMountsConditional(yamlContent string) string {
	metricsCertsPath := "/tmp/k8s-metrics-server/metrics-certs"
	if strings.Contains(yamlContent, "metrics-certs") && strings.Contains(yamlContent, metricsCertsPath) {
		// [ \t]+ matches only spaces/tabs so the newline is not consumed by the regexp.
		pattern := regexp.MustCompile(
			`([ \t]+)-\s*mountPath:\s*/tmp/k8s-metrics-server/metrics-certs[\s\S]*?readOnly:\s*true`)
		yamlContent = wrapWithMetricsTLSConditional(pattern, yamlContent)
	}
	return yamlContent
}

func wrapWithMetricsTLSConditional(pattern *regexp.Regexp, yamlContent string) string {
	const metricsCondition = "{{- if and .Values.certManager.enabled .Values.metrics.enabled .Values.metrics.secure }}"
	return pattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
		return wrapBlock(match, metricsCondition)
	})
}
