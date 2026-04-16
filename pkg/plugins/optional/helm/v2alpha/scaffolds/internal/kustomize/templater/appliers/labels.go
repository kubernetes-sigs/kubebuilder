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
