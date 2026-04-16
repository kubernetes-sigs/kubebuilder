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
	"strconv"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

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

	// Optional Kubernetes features: deployment strategy and pod scheduling
	// Template conditionals are always created (even if field doesn't exist in kustomize)
	// so users can uncomment them in values.yaml without regenerating the chart
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

func templateReplicas(yamlContent string) string {
	if strings.Contains(yamlContent, ".Values.manager.replicas") {
		return yamlContent
	}
	// Replace spec.replicas with values.yaml reference, preserving indentation
	replicasPattern := regexp.MustCompile(`(?m)^(\s*)replicas:\s*\d+\s*$`)
	return replicasPattern.ReplaceAllString(yamlContent, "${1}replicas: {{ .Values.manager.replicas }}")
}

func AddCustomLabelsAndAnnotations(yamlContent string) string {
	// Check which blocks are present to avoid duplicates when re-running
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

func templateEnvironmentVariables(yamlContent string) string {
	containerName := GetDefaultContainerName(yamlContent)
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

		indentStr, indentLen := LeadingWhitespace(lines[i])
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
func templateResources(yamlContent string) string {
	containerName := GetDefaultContainerName(yamlContent)
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

		indentStr, indentLen := LeadingWhitespace(lines[i])
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

func templateSecurityContexts(yamlContent string) string {
	// Security contexts are preserved from kustomize output as-is
	return yamlContent
}

func templateVolumeMounts(yamlContent string) string {
	return appendToListFromValues(yamlContent, "volumeMounts:", ".Values.manager.extraVolumeMounts")
}

func templateVolumes(yamlContent string) string {
	return appendToListFromValues(yamlContent, "volumes:", ".Values.manager.extraVolumes")
}

// appendToListFromValues appends a values path to a YAML list field.
// For "key: []", replaces with conditional template. For "key:" with items, appends to the end.
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
func templateImagePullSecrets(yamlContent string) string {
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

			indentStr, indentLen := LeadingWhitespace(lines[i])
			end := i + 1
			// Find end of imagePullSecrets block
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

	// Field doesn't exist - inject into pod spec (spec.template.spec)
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

// templatePodSecurityContext exposes podSecurityContext via values.yaml
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
func templateContainerSecurityContext(yamlContent string) string {
	containerName := GetDefaultContainerName(yamlContent)
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

		indentStr, indentLen := LeadingWhitespace(lines[i])
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

// isManagerDeployment checks if a Deployment is the controller manager.
// It returns true if either the deployment name contains "controller-manager"
// OR the deployment has the label "control-plane: controller-manager".

// templateControllerManagerArgs exposes controller manager args via values.yaml while keeping core defaults
func templateControllerManagerArgs(yamlContent string) string {
	containerName := GetDefaultContainerName(yamlContent)
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
func templateImageReference(yamlContent string) string {
	containerName := GetDefaultContainerName(yamlContent)
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

		indentStr, indentLen := LeadingWhitespace(lines[i])

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

		imageLine := indentStr + "image: \"{{ .Values.manager.image.repository }}" +
			"{{- if not (contains \"@\" .Values.manager.image.repository) }}" +
			":{{ .Values.manager.image.tag | default .Chart.AppVersion }}{{- end }}\""
		pullPolicyLineStart := indentStr + "{{- with .Values.manager.image.pullPolicy }}"
		pullPolicyLine := indentStr + "imagePullPolicy: {{ . }}"
		pullPolicyLineEnd := indentStr + "{{- end }}"

		remainder := lines[end:]
		if len(remainder) > 0 && strings.HasPrefix(strings.TrimSpace(remainder[0]), "imagePullPolicy:") {
			remainder = remainder[1:]
		}

		newLines := append([]string{}, lines[:i]...)
		newLines = append(newLines, imageLine, pullPolicyLineStart, pullPolicyLine, pullPolicyLineEnd)
		newLines = append(newLines, remainder...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
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
		// Find parent block start if the key is missing
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

			// Parent key part found
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
		// Find the existing block - stop at the first match.
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
			break // use the first match only
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
		if trimmed == common.YamlKeyTemplate {
			foundTemplate = true
			continue
		}
		if foundTemplate && trimmed == common.YamlKeySpec {
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
	_, indentLen := LeadingWhitespace(lines[insertAt])
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

// templateTerminationGracePeriodSeconds templates terminationGracePeriodSeconds.
// Always injects this field (even when not in kustomize output) so users can configure
// pod shutdown timeout in values.yaml. Uses hasKey to allow 0 seconds (immediate shutdown).
func templateTerminationGracePeriodSeconds(yamlContent string) string {
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
	_, indentLen := LeadingWhitespace(lines[insertAt-1])
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
			result = injectDeploymentAnnotations(result, childIndent)
		}

		result = append(result, line)
		state.addedAnnotationsToDeployment = true
		state.currentBlock = blockNone
	}

	return result
}

// handlePodAnnotations handles injection of custom Pod template annotations.
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
			result = addPodAnnotations(result, childIndent)
		}

		result = append(result, line)
		state.addedPodAnnotations = true
		state.currentBlock = blockNone
	}

	if state.position == positionPodMetadata && !state.addedPodAnnotations && trimmed == common.YamlKeyLabels {
		result = result[:len(result)-1]
		result = append(result, indent+"{{- if .Values.manager.pod.annotations }}")
		result = append(result, indent+"annotations:")
		result = addPodAnnotations(result, indent+"  ")
		result = append(result, indent+"{{- end }}")
		result = append(result, indent+common.YamlKeyLabels)
		state.addedPodAnnotations = true
	}

	return result
}

// shouldInjectDeploymentAnnotations checks if we should inject Deployment annotations.
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

// shouldInjectPodAnnotations checks if we should inject Pod annotations.
func shouldInjectPodAnnotations(state *customFieldsState, trimmed string, indentLen int) bool {
	return (state.position == positionPodMetadata || state.position == positionAfterDeploymentMetadata) &&
		state.currentBlock == blockPodAnnotations &&
		!state.addedPodAnnotations &&
		indentLen <= state.currentBlockIndent &&
		trimmed != "" &&
		trimmed != common.YamlKeyAnnotations &&
		!strings.HasPrefix(trimmed, common.YamlKeyAnnotations+" {")
}

// handleDeploymentLabels handles injection of custom Deployment labels.
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

// appendHelmMapBlock appends Helm template blocks for rendering YAML maps with optional key filtering.
// When existingKeys is empty, uses simple {{- if }} conditional.
// When existingKeys is provided, uses nested {{- with }} blocks with omit() to filter duplicate keys.
// When existingKeys is provided, adds an extra {{- with omit() }} layer for key filtering.

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

	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
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

// MakeWebhookVolumesConditional makes webhook volumes conditional on certManager.enable.
func MakeWebhookVolumesConditional(yamlContent string) string {
	// Make webhook volumes conditional on certManager.enable
	if strings.Contains(yamlContent, "webhook-certs") && strings.Contains(yamlContent, "secretName: webhook-server-cert") {
		// Match only spaces/tabs for indent to avoid consuming the newline
		volumePattern := regexp.MustCompile(`([ \t]+)-\s*name:\s*webhook-certs[\s\S]*?secretName:\s*webhook-server-cert`)
		yamlContent = volumePattern.ReplaceAllStringFunc(yamlContent, MakeYamlContent)
	}

	return yamlContent
}

// MakeWebhookVolumeMountsConditional makes webhook volumeMounts conditional on certManager.enable.
func MakeWebhookVolumeMountsConditional(yamlContent string) string {
	// Make webhook volumeMounts conditional on certManager.enable
	webhookCertsPath := "/tmp/k8s-webhook-server/serving-certs"
	if strings.Contains(yamlContent, "webhook-certs") && strings.Contains(yamlContent, webhookCertsPath) {
		// Match only spaces/tabs for indent to avoid consuming the newline
		mountPattern := regexp.MustCompile(
			`([ \t]+)-\s*mountPath:\s*/tmp/k8s-webhook-server/serving-certs[\s\S]*?readOnly:\s*true`)
		yamlContent = mountPattern.ReplaceAllStringFunc(yamlContent, MakeYamlContent)
	}

	return yamlContent
}

// MakeMetricsVolumesConditional makes metrics volumes conditional on certManager.enable AND metrics.enable.
func MakeMetricsVolumesConditional(yamlContent string) string {
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

// MakeMetricsVolumeMountsConditional makes metrics volumeMounts conditional on certManager.enable AND metrics.enable.
func MakeMetricsVolumeMountsConditional(yamlContent string) string {
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
