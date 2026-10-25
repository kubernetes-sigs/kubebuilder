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

var defaultContainerPattern = regexp.MustCompile(
	`(?m)^\s*` + regexp.QuoteMeta(common.DefaultContainerAnnotation) + `:\s+(\S+)`,
)

// GetDefaultContainerName extracts the container name from kubectl.kubernetes.io/default-container annotation.
// This allows the Helm plugin to work with any container name, not just "manager".
// If the annotation is not found, it falls back to "manager" for backward compatibility.
func GetDefaultContainerName(yamlContent string) string {
	matches := defaultContainerPattern.FindStringSubmatch(yamlContent)
	if len(matches) > 1 {
		return matches[1]
	}
	return common.DefaultManagerContainerName
}

// LeadingWhitespace extracts the leading whitespace from a line.
// Returns the whitespace string and its length in characters.
func LeadingWhitespace(line string) (string, int) {
	trimmed := strings.TrimLeft(line, " \t")
	indentLen := len(line) - len(trimmed)
	return line[:indentLen], indentLen
}

// blockScalarRegex matches a line whose value starts a multi-line block scalar, such as "key: |",
// "key: >-" or "- |2". The first group is set when the scalar is a list item.
var blockScalarRegex = regexp.MustCompile(`(?:(^\s*-)|:)\s+[|>][-+0-9]*\s*$`)

// blockScalarLines reports, for each line, whether it is text inside a multi-line block scalar value:
// a line after "key: |" that is blank or indented deeper than the key. Such lines are not YAML keys.
func blockScalarLines(lines []string) []bool {
	inside := make([]bool, len(lines))
	parentIndent := -1
	for i, line := range lines {
		_, indent := LeadingWhitespace(line)
		if parentIndent >= 0 && (strings.TrimSpace(line) == "" || indent > parentIndent) {
			inside[i] = true
			continue
		}
		parentIndent = -1
		if match := blockScalarRegex.FindStringSubmatch(line); match != nil {
			if match[1] != "" {
				// "- |": the text sits deeper than the dash.
				parentIndent = indent
			} else {
				// "key: |" or "- key: |": the text sits deeper than the key.
				parentIndent = len(line) - len(strings.TrimLeft(line, " -"))
			}
		}
	}
	return inside
}

// IsManagerDeployment reports whether resource is the controller-manager Deployment.
// Annotation is not checked — any extra Deployment may carry it, causing false positives.
func IsManagerDeployment(resource *unstructured.Unstructured) bool {
	if resource.GetLabels()["control-plane"] == "controller-manager" {
		return true
	}
	if names := ExtractContainerNames(resource); names["manager"] {
		return true
	}
	return strings.Contains(resource.GetName(), "controller-manager")
}

// wrapWebhookCertificateBlock wraps a webhook certificate block with its feature conditions.
// Shifts by 2 spaces to align with the child indent used by appendToListFromValues.
func wrapWebhookCertificateBlock(match string) string {
	return wrapBlock(match, "{{- if and .Values.certManager.enabled .Values.webhook.enabled }}")
}

// wrapBlock wraps a YAML block match with the given Helm conditional string.
func wrapBlock(match, condition string) string {
	lines := strings.Split(match, "\n")
	indent, _ := LeadingWhitespace(lines[0])
	childIndent := indent + "  "
	var result strings.Builder
	fmt.Fprintf(&result, "%s%s\n", childIndent, condition)
	for _, line := range lines {
		result.WriteString("  ")
		result.WriteString(line)
		result.WriteByte('\n')
	}
	fmt.Fprintf(&result, "%s{{- end }}", childIndent)
	return result.String()
}

const (
	k8sObjectSpecField     = "spec"
	k8sObjectTemplateField = "template"
	k8sContainersFieldName = "containers"
)

var (
	podTemplateContainersPath = []string{
		k8sObjectSpecField, k8sObjectTemplateField, k8sObjectSpecField, k8sContainersFieldName,
	}
	podTemplateInitContainersPath = []string{
		k8sObjectSpecField, k8sObjectTemplateField, k8sObjectSpecField, "initContainers",
	}
)

// FindManagerContainerRange returns the 0-based inclusive line range [start, end]
// of the manager container in yamlContent.
// Returns (-1, -1) when not found; callers use this to restrict substitutions to the manager only.
func FindManagerContainerRange(yamlContent string) (int, int) {
	name := GetDefaultContainerName(yamlContent)
	lines := strings.Split(yamlContent, "\n")

	listLine, listIndent := findListField(lines, k8sContainersFieldName+":")
	if listLine < 0 {
		return -1, -1
	}

	nameField := "name: " + name
	itemStart := -1
	itemChildIndent := -1
	found := false

	for i := listLine + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		_, indent := LeadingWhitespace(lines[i])

		if indent > listIndent {
			if itemStart >= 0 && indent == itemChildIndent && trimmed == nameField {
				found = true
			}
			continue
		}

		if found {
			return itemStart, i - 1
		}
		if indent < listIndent || !strings.HasPrefix(trimmed, "- ") {
			break
		}
		itemStart = i
		itemChildIndent = indent + 2
		if trimmed == "- "+nameField {
			found = true
		}
	}
	if found {
		return itemStart, len(lines) - 1
	}
	return -1, -1
}

func findListField(lines []string, field string) (int, int) {
	for i, line := range lines {
		if strings.TrimSpace(line) == field {
			_, indent := LeadingWhitespace(line)
			return i, indent
		}
	}
	return -1, -1
}

// ExtractContainerNames returns all container and initContainer names from a Deployment.
func ExtractContainerNames(resource *unstructured.Unstructured) map[string]bool {
	names := map[string]bool{}
	for _, fieldPath := range [][]string{
		podTemplateContainersPath,
		podTemplateInitContainersPath,
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
