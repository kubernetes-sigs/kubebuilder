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
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ValueInjector handles injecting user-added values into Helm templates
type ValueInjector struct {
	userValues *UserAddedValues
	chartName  string
}

// NewValueInjector creates a new value injector
func NewValueInjector(userValues *UserAddedValues, chartName string) *ValueInjector {
	return &ValueInjector{
		userValues: userValues,
		chartName:  chartName,
	}
}

// InjectCustomValues injects user-added custom values into the YAML template
func (v *ValueInjector) InjectCustomValues(yamlContent string, resource *unstructured.Unstructured) string {
	kind := resource.GetKind()

	// Apply injections based on resource kind
	if kind == kindDeployment {
		yamlContent = v.injectDeploymentValues(yamlContent)
	}

	// Apply injections that work for any resource with metadata
	yamlContent = v.injectMetadataLabels(yamlContent, resource)
	yamlContent = v.injectMetadataAnnotations(yamlContent, resource)

	// Inject pod template values for resources that have pod templates
	if kind == kindDeployment || kind == "StatefulSet" || kind == "DaemonSet" || kind == "Job" || kind == "CronJob" {
		yamlContent = v.injectPodTemplateValues(yamlContent)
	}

	return yamlContent
}

// injectDeploymentValues injects custom values specific to Deployment resources
func (v *ValueInjector) injectDeploymentValues(yamlContent string) string {
	// Inject environment variables if any
	if len(v.userValues.EnvVars) > 0 {
		yamlContent = v.injectEnvVars(yamlContent)
	}

	// Inject custom manager fields
	for fieldName := range v.userValues.CustomManagerFields {
		// Skip special fields that are handled separately
		if fieldName == "labels" || fieldName == "annotations" ||
			fieldName == "podLabels" || fieldName == "podAnnotations" ||
			fieldName == "env" {
			continue
		}

		// Inject based on field name patterns
		switch fieldName {
		case "ports":
			yamlContent = v.injectCustomPorts(yamlContent)
		case "volumes":
			yamlContent = v.injectCustomVolumes(yamlContent)
		case "volumeMounts":
			yamlContent = v.injectCustomVolumeMounts(yamlContent)
		case "initContainers":
			yamlContent = v.injectInitContainers(yamlContent)
		case "serviceAccountName":
			yamlContent = v.injectServiceAccountName(yamlContent)
		case "hostNetwork":
			yamlContent = v.injectHostNetwork(yamlContent)
		case "dnsPolicy":
			yamlContent = v.injectDNSPolicy(yamlContent)
		case "priorityClassName":
			yamlContent = v.injectPriorityClassName(yamlContent)
		default:
			// For unknown fields, try to inject them generically
			yamlContent = v.injectGenericField(yamlContent)
		}
	}

	return yamlContent
}

// injectMetadataLabels injects custom labels into resource metadata
func (v *ValueInjector) injectMetadataLabels(yamlContent string, resource *unstructured.Unstructured) string {
	if len(v.userValues.ManagerLabels) == 0 {
		return yamlContent
	}

	kind := resource.GetKind()
	// Only inject into Deployment resources to avoid over-propagation
	if kind != kindDeployment {
		return yamlContent
	}

	// Find the metadata.labels section
	// Look for "labels:" followed by indented label entries
	lines := strings.Split(yamlContent, "\n")
	var result []string
	foundMetadataLabels := false
	labelsIndent := ""

	for i, line := range lines {
		result = append(result, line)

		// Look for "labels:" in metadata section (not in spec.template)
		if strings.Contains(line, "labels:") && !foundMetadataLabels {
			// Check if this is in metadata by looking at previous lines
			isInMetadata := false
			for j := i - 1; j >= 0 && j >= i-10; j-- {
				if strings.TrimSpace(lines[j]) == yamlKeyMetadata {
					isInMetadata = true
					break
				}
				if strings.TrimSpace(lines[j]) == yamlKeySpec || strings.TrimSpace(lines[j]) == "template:" {
					break
				}
			}

			if !isInMetadata {
				continue
			}

			foundMetadataLabels = true
			// Extract indentation from the labels line
			before, _, _ := strings.Cut(line, "labels:")
			labelsIndent = before + "  "

			// Add custom labels after existing labels
			for key := range v.userValues.ManagerLabels {
				valuePath := fmt.Sprintf(".Values.manager.labels.%s", escapeYAMLKey(key))
				result = append(result, fmt.Sprintf("%s%s: {{ %s }}", labelsIndent, key, valuePath))
			}
		}
	}

	return strings.Join(result, "\n")
}

// injectMetadataAnnotations injects custom annotations into resource metadata
func (v *ValueInjector) injectMetadataAnnotations(yamlContent string, resource *unstructured.Unstructured) string {
	if len(v.userValues.ManagerAnnotations) == 0 {
		return yamlContent
	}

	kind := resource.GetKind()
	// Only inject into Deployment resources to avoid over-propagation
	if kind != kindDeployment {
		return yamlContent
	}

	// Check if annotations section exists
	if !strings.Contains(yamlContent, "annotations:") {
		// Need to add annotations section after labels
		return v.addAnnotationsSection(yamlContent)
	}

	// Find and inject into the metadata.annotations section
	lines := strings.Split(yamlContent, "\n")
	var result []string
	foundMetadataAnnotations := false
	annotationsIndent := ""

	for i, line := range lines {
		result = append(result, line)

		// Look for "annotations:" in metadata section
		if strings.Contains(line, "annotations:") && !foundMetadataAnnotations {
			// Check if this is in metadata by looking at previous lines
			isInMetadata := false
			for j := i - 1; j >= 0 && j >= i-10; j-- {
				if strings.TrimSpace(lines[j]) == yamlKeyMetadata {
					isInMetadata = true
					break
				}
				if strings.TrimSpace(lines[j]) == yamlKeySpec || strings.TrimSpace(lines[j]) == yamlKeyTemplate {
					break
				}
			}

			if !isInMetadata {
				continue
			}

			foundMetadataAnnotations = true
			// Extract indentation from the annotations line
			before, _, _ := strings.Cut(line, "annotations:")
			annotationsIndent = before + "  "

			// Add custom annotations after existing annotations
			for key := range v.userValues.ManagerAnnotations {
				valuePath := fmt.Sprintf(".Values.manager.annotations.%s", escapeYAMLKey(key))
				result = append(result, fmt.Sprintf("%s%s: {{ %s }}", annotationsIndent, key, valuePath))
			}
		}
	}

	return strings.Join(result, "\n")
}

// addAnnotationsSection adds an annotations section if it doesn't exist
func (v *ValueInjector) addAnnotationsSection(yamlContent string) string {
	// Find metadata section and add annotations after labels
	lines := strings.Split(yamlContent, "\n")
	var result []string
	foundMetadataLabels := false
	labelsIndent := ""

	for i, line := range lines {
		result = append(result, line)

		// Look for "labels:" in metadata section
		if strings.Contains(line, "labels:") && !foundMetadataLabels {
			// Check if this is in metadata
			isInMetadata := false
			for j := i - 1; j >= 0 && j >= i-10; j-- {
				if strings.TrimSpace(lines[j]) == yamlKeyMetadata {
					isInMetadata = true
					break
				}
			}

			if !isInMetadata {
				continue
			}

			foundMetadataLabels = true
			// Extract indentation
			before, _, _ := strings.Cut(line, "labels:")
			labelsIndent = before

			// Skip existing label entries
			j := i + 1
			for j < len(lines) && strings.HasPrefix(lines[j], labelsIndent+"  ") {
				result = append(result, lines[j])
				j++
			}

			// Add annotations section
			result = append(result, labelsIndent+"annotations:")
			for key := range v.userValues.ManagerAnnotations {
				valuePath := fmt.Sprintf(".Values.manager.annotations.%s", escapeYAMLKey(key))
				result = append(result, fmt.Sprintf("%s  %s: {{ %s }}", labelsIndent, key, valuePath))
			}
		}
	}

	return strings.Join(result, "\n")
}

// injectPodTemplateValues injects custom values into pod templates
func (v *ValueInjector) injectPodTemplateValues(yamlContent string) string {
	// Inject pod labels if any
	if len(v.userValues.ManagerPodLabels) > 0 {
		yamlContent = v.injectPodLabels(yamlContent)
	}

	// Inject pod annotations if any
	if len(v.userValues.ManagerPodAnnotations) > 0 {
		yamlContent = v.injectPodAnnotations(yamlContent)
	}

	return yamlContent
}

// injectPodLabels injects custom labels into pod template
// injectPodTemplateField injects custom labels or annotations into pod template metadata
func (v *ValueInjector) injectPodTemplateField(
	yamlContent, fieldName, valuesPath string,
	fieldMap map[string]string,
) string {
	lines := strings.Split(yamlContent, "\n")
	var result []string
	foundField := false
	fieldIndent := ""
	inTemplate := false

	for i, line := range lines {
		result = append(result, line)

		// Track if we're in the template section
		if strings.Contains(line, "template:") {
			inTemplate = true
		}

		// Look for the field within template.metadata
		if inTemplate && strings.Contains(line, fieldName+":") && !foundField {
			// Check if this is within template.metadata
			isInPodMetadata := false
			for j := i - 1; j >= 0 && j >= i-5; j-- {
				trimmed := strings.TrimSpace(lines[j])
				if trimmed == "metadata:" {
					// Check if there's a template: before this metadata
					for k := j - 1; k >= 0 && k >= j-5; k-- {
						if strings.Contains(lines[k], "template:") {
							isInPodMetadata = true
							break
						}
					}
					break
				}
			}

			if !isInPodMetadata {
				continue
			}

			foundField = true
			// Extract indentation
			before, _, _ := strings.Cut(line, fieldName+":")
			fieldIndent = before + "  "

			// Add custom field entries
			for key := range fieldMap {
				valuePath := fmt.Sprintf("%s.%s", valuesPath, escapeYAMLKey(key))
				result = append(result, fmt.Sprintf("%s%s: {{ %s }}", fieldIndent, key, valuePath))
			}
		}
	}

	return strings.Join(result, "\n")
}

func (v *ValueInjector) injectPodLabels(yamlContent string) string {
	return v.injectPodTemplateField(
		yamlContent, "labels", ".Values.manager.podLabels", v.userValues.ManagerPodLabels,
	)
}

// injectPodAnnotations injects custom annotations into pod template
func (v *ValueInjector) injectPodAnnotations(yamlContent string) string {
	return v.injectPodTemplateField(
		yamlContent, "annotations", ".Values.manager.podAnnotations", v.userValues.ManagerPodAnnotations,
	)
}

// injectEnvVars injects environment variables from manager.env
func (v *ValueInjector) injectEnvVars(yamlContent string) string {
	// The env vars are already handled by the existing helm_templater.go
	// which adds {{- if .Values.manager.env }} template
	// This function is kept for consistency but doesn't need to do anything
	// since the standard scaffolding already handles manager.env
	return yamlContent
}

// injectCustomPorts injects custom container ports
func (v *ValueInjector) injectCustomPorts(yamlContent string) string {
	// Find the ports section in the container
	portsPattern := regexp.MustCompile(`(?m)^(\s+)ports:\s*$\n`)
	matches := portsPattern.FindStringIndex(yamlContent)

	if matches == nil {
		return yamlContent
	}

	// Add conditional to include custom ports
	portsEndIdx := matches[1]
	customPortsTemplate := `{{- if .Values.manager.ports }}
{{- toYaml .Values.manager.ports | nindent 8 }}
{{- end }}
`

	return yamlContent[:portsEndIdx] + customPortsTemplate + yamlContent[portsEndIdx:]
}

// injectCustomVolumes injects custom volumes at pod level
func (v *ValueInjector) injectCustomVolumes(yamlContent string) string {
	// Find the existing volumes section in pod spec and add custom volumes template
	lines := strings.Split(yamlContent, "\n")
	volumesLineIdx := -1
	lastVolumeLineIdx := -1
	volumesIndent := ""
	inVolumesSection := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for "volumes:" line - this is always at pod spec level after serviceAccountName
		if trimmed == "volumes:" && volumesLineIdx == -1 {
			volumesLineIdx = i
			inVolumesSection = true
			// Get the base indentation for volume entries
			before, _, _ := strings.Cut(line, "volumes:")
			volumesIndent = before
			continue
		}

		// Track lines within the volumes section
		if inVolumesSection {
			if trimmed == "" {
				continue // Skip empty lines
			}

			lineIndent := len(line) - len(strings.TrimLeft(line, " "))
			baseIndent := len(volumesIndent)

			// YAML list items start with "- " at the same level as the parent key
			// Content under the list item is further indented
			// So we check if line is at baseIndent (list item) or deeper (list item content)
			if lineIndent >= baseIndent {
				lastVolumeLineIdx = i
			} else {
				// We've exited the volumes section (found a line at less indentation)
				inVolumesSection = false
			}
		}
	}

	// If we found volumes, add custom volumes template after the last volume entry
	if volumesLineIdx >= 0 && lastVolumeLineIdx >= 0 {
		// Build the custom volumes template with proper indentation
		// Volume list items are indented 4 spaces from "volumes:" line
		volumeEntryIndent := volumesIndent + "    "
		customVolumesTemplate := []string{
			fmt.Sprintf("%s{{- with .Values.manager.volumes }}", volumeEntryIndent),
			fmt.Sprintf("%s{{- toYaml . | nindent %d }}", volumeEntryIndent, len(volumeEntryIndent)),
			fmt.Sprintf("%s{{- end }}", volumeEntryIndent),
		}

		// Insert after the last volume line
		newLines := append([]string{}, lines[:lastVolumeLineIdx+1]...)
		newLines = append(newLines, customVolumesTemplate...)
		newLines = append(newLines, lines[lastVolumeLineIdx+1:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

// injectCustomVolumeMounts injects custom volume mounts at container level
func (v *ValueInjector) injectCustomVolumeMounts(yamlContent string) string {
	// Find the existing volumeMounts section in the manager container
	lines := strings.Split(yamlContent, "\n")
	foundVolumeMounts := false
	volumeMountsLineIdx := -1
	lastVolumeMountIdx := -1
	volumeMountsIndent := ""
	inManagerContainer := false
	inVolumeMounts := false

	for i, line := range lines {
		// Track if we're in the manager container
		if strings.Contains(line, "name: manager") || strings.Contains(line, `name: {{ include`) {
			inManagerContainer = true
		} else if inManagerContainer && strings.TrimSpace(line) != "" {
			// Check if we've left the container
			lineIndent := len(line) - len(strings.TrimLeft(line, " "))
			if lineIndent <= 6 && (strings.Contains(line, "name:") || strings.Contains(line, "-")) {
				// We've moved to another container
				inManagerContainer = false
			}
		}

		// Look for "volumeMounts:" in manager container
		if inManagerContainer && strings.Contains(line, "volumeMounts:") && volumeMountsLineIdx == -1 {
			volumeMountsLineIdx = i
			inVolumeMounts = true
			before, _, _ := strings.Cut(line, "volumeMounts:")
			volumeMountsIndent = before
			foundVolumeMounts = true
			continue
		}

		// Track lines within volumeMounts section
		if inVolumeMounts {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue // Skip empty lines
			}

			lineIndent := len(line) - len(strings.TrimLeft(line, " "))
			baseIndent := len(volumeMountsIndent)

			// List items and their content are at baseIndent or deeper
			if lineIndent >= baseIndent {
				lastVolumeMountIdx = i
			} else {
				// We've left the volumeMounts section
				inVolumeMounts = false
			}
		}
	}

	// Insert custom volumeMounts template after last volumeMount
	if foundVolumeMounts && lastVolumeMountIdx >= 0 {
		// Volume mount list items are indented 2 spaces from "volumeMounts:" line (YAML list syntax)
		volumeMountEntryIndent := volumeMountsIndent + "  "
		customVolumeMountsTemplate := []string{
			fmt.Sprintf("%s{{- with .Values.manager.volumeMounts }}", volumeMountEntryIndent),
			fmt.Sprintf("%s{{- toYaml . | nindent %d }}", volumeMountEntryIndent, len(volumeMountEntryIndent)),
			fmt.Sprintf("%s{{- end }}", volumeMountEntryIndent),
		}

		newLines := append([]string{}, lines[:lastVolumeMountIdx+1]...)
		newLines = append(newLines, customVolumeMountsTemplate...)
		newLines = append(newLines, lines[lastVolumeMountIdx+1:]...)
		return strings.Join(newLines, "\n")
	}

	return yamlContent
}

// injectInitContainers injects init containers
func (v *ValueInjector) injectInitContainers(yamlContent string) string {
	// Find spec.template.spec section and add initContainers
	specPattern := regexp.MustCompile(`(?m)^(\s+)spec:\s*$\n\s+containers:\s*$`)
	matches := specPattern.FindStringSubmatchIndex(yamlContent)

	if matches == nil {
		return yamlContent
	}

	// Insert initContainers before containers
	insertIdx := matches[0]
	indent := yamlContent[matches[2]:matches[3]]

	initContainersTemplate := fmt.Sprintf(`%sinitContainers:
{{- if .Values.manager.initContainers }}
{{- toYaml .Values.manager.initContainers | nindent 8 }}
{{- end }}
`, indent)

	return yamlContent[:insertIdx] + initContainersTemplate + yamlContent[insertIdx:]
}

// injectServiceAccountName injects custom service account name
func (v *ValueInjector) injectServiceAccountName(yamlContent string) string {
	// Replace existing serviceAccountName with conditional template
	serviceAccountPattern := regexp.MustCompile(`(?m)^(\s+)serviceAccountName:.*$`)

	serviceAccountTemplate := `{{- if .Values.manager.serviceAccountName }}
      serviceAccountName: {{ .Values.manager.serviceAccountName }}
{{- else }}
			serviceAccountName: {{ include "` + v.chartName + `.resourceName" (dict "suffix" "controller-manager" "context" $) }}
{{- end }}`

	return serviceAccountPattern.ReplaceAllString(yamlContent, serviceAccountTemplate)
}

// injectHostNetwork injects hostNetwork setting
func (v *ValueInjector) injectHostNetwork(yamlContent string) string {
	// Add hostNetwork conditionally
	specPattern := regexp.MustCompile(`(?m)^(\s+)spec:\s*$\n`)
	matches := specPattern.FindStringSubmatchIndex(yamlContent)

	if matches == nil {
		return yamlContent
	}

	indent := yamlContent[matches[2]:matches[3]]
	insertIdx := matches[1]

	hostNetworkTemplate := fmt.Sprintf(`
%shostNetwork: {{ .Values.manager.hostNetwork | default false }}`, indent)

	return yamlContent[:insertIdx] + hostNetworkTemplate + yamlContent[insertIdx:]
}

// injectDNSPolicy injects DNS policy
func (v *ValueInjector) injectDNSPolicy(yamlContent string) string {
	// Add dnsPolicy conditionally
	specPattern := regexp.MustCompile(`(?m)^(\s+)securityContext:\s*$`)
	matches := specPattern.FindStringIndex(yamlContent)

	if matches == nil {
		return yamlContent
	}

	insertIdx := matches[0]

	dnsPolicyTemplate := `{{- if .Values.manager.dnsPolicy }}
      dnsPolicy: {{ .Values.manager.dnsPolicy }}
{{- end }}
`

	return yamlContent[:insertIdx] + dnsPolicyTemplate + yamlContent[insertIdx:]
}

// injectPriorityClassName injects priority class name
func (v *ValueInjector) injectPriorityClassName(yamlContent string) string {
	// Add priorityClassName conditionally
	securityContextPattern := regexp.MustCompile(`(?m)^(\s+)securityContext:\s*$`)
	matches := securityContextPattern.FindStringIndex(yamlContent)

	if matches == nil {
		return yamlContent
	}

	insertIdx := matches[0]

	priorityClassTemplate := `{{- if .Values.manager.priorityClassName }}
      priorityClassName: {{ .Values.manager.priorityClassName }}
{{- end }}
`

	return yamlContent[:insertIdx] + priorityClassTemplate + yamlContent[insertIdx:]
}

// injectGenericField attempts to inject a generic field
func (v *ValueInjector) injectGenericField(yamlContent string) string {
	// This is a fallback for fields we don't have specific handling for
	// We'll add them as comments that users can uncomment and modify
	return yamlContent
}

// escapeYAMLKey escapes a YAML key for use in a Helm template
func escapeYAMLKey(key string) string {
	// Replace special characters that might cause issues in YAML paths
	key = strings.ReplaceAll(key, ".", "_")
	key = strings.ReplaceAll(key, "/", "_")
	return key
}
