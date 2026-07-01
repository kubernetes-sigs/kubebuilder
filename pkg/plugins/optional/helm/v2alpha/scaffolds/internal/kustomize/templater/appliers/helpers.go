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

// MakeYamlContent wraps a YAML block with a cert-manager conditional.
// Shifts by 2 spaces to align with the child indent used by appendToListFromValues.
func MakeYamlContent(match string) string {
	return wrapBlock(match, "{{- if .Values.certManager.enabled }}")
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
)

var (
	podTemplateContainersPath = []string{
		k8sObjectSpecField, k8sObjectTemplateField, k8sObjectSpecField, "containers",
	}
	podTemplateInitContainersPath = []string{
		k8sObjectSpecField, k8sObjectTemplateField, k8sObjectSpecField, "initContainers",
	}
)

// FindManagerContainerRange returns the 0-based inclusive line range of the manager container in yamlContent.
// Returns (-1, -1) when not found; callers use this to restrict substitutions to the manager only.
//
// The function handles yaml.Marshal output where fields are sorted alphabetically,
// meaning the name field may appear anywhere in the container block (not just first).
func FindManagerContainerRange(yamlContent string) (int, int) {
	name := GetDefaultContainerName(yamlContent)
	lines := strings.Split(yamlContent, "\n")

	listLine, listIndent := findListField(lines, "containers:")
	if listLine < 0 {
		return -1, -1
	}

	start := findNamedBlockStart(lines, listLine, listIndent, name)
	if start < 0 {
		return -1, -1
	}
	return start, findListItemEnd(lines, start, listIndent)
}

// findListField returns the line index and indent of a YAML field.
func findListField(lines []string, field string) (int, int) {
	for i, line := range lines {
		if strings.TrimSpace(line) == field {
			_, indent := LeadingWhitespace(line)
			return i, indent
		}
	}
	return -1, -1
}

// findNamedBlockStart locates the "- " line that starts the YAML list
// item containing "name: <name>". Scans forward from listLine so that
// Helm directive insertions at low indentation are safely ignored.
func findNamedBlockStart(lines []string, listLine, listIndent int, name string) int {
	blockStart := -1
	fieldIndent := -1

	for i := listLine + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		_, indent := LeadingWhitespace(lines[i])

		if trimmed == "" {
			continue
		}
		if indent < listIndent {
			break
		}

		if indent == listIndent && strings.HasPrefix(trimmed, "- ") {
			blockStart = i
			fieldIndent = -1
		}

		if blockStart < 0 {
			continue
		}

		if fieldIndent < 0 && indent > listIndent && !strings.HasPrefix(trimmed, "- ") {
			fieldIndent = indent
		}

		isFirstField := indent == listIndent && trimmed == "- name: "+name
		isContinuationField := fieldIndent > 0 && indent == fieldIndent && trimmed == "name: "+name

		if isFirstField || isContinuationField {
			return blockStart
		}
	}

	return -1
}

// findListItemEnd returns the inclusive end line index of a YAML list
// item that starts at blockStart within a list at listIndent.
func findListItemEnd(lines []string, blockStart, listIndent int) int {
	for i := blockStart + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		_, indent := LeadingWhitespace(lines[i])
		if indent <= listIndent && strings.HasPrefix(trimmed, "- ") {
			return i - 1
		}
	}
	return len(lines) - 1
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
