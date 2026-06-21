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

// GetDefaultContainerName extracts the container name from kubectl.kubernetes.io/default-container annotation.
// This allows the Helm plugin to work with any container name, not just "manager".
// If the annotation is not found, it falls back to "manager" for backward compatibility.
func GetDefaultContainerName(yamlContent string) string {
	pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(common.DefaultContainerAnnotation) + `:\s+(\S+)`)
	matches := pattern.FindStringSubmatch(yamlContent)
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

// IsManagerDeployment checks if a Deployment is the controller manager.
// It returns true if either the deployment name contains "controller-manager"
// OR the deployment has the label "control-plane: controller-manager".
func IsManagerDeployment(resource *unstructured.Unstructured) bool {
	name := resource.GetName()
	labels := resource.GetLabels()
	return strings.Contains(name, "controller-manager") ||
		(labels != nil && labels["control-plane"] == "controller-manager")
}

// MakeYamlContent wraps YAML content with conditional cert-manager wrappers.
// This function is used as a callback for regexp.ReplaceAllStringFunc.
// It shifts the block by 2 additional spaces so that items align with the
// child indent used by appendToListFromValues for extraVolumes/extraVolumeMounts.
func MakeYamlContent(match string) string {
	lines := strings.Split(match, "\n")
	if len(lines) > 0 {
		var indent strings.Builder
		if len(lines[0]) > 0 && lines[0][0] == ' ' {
			// Count leading spaces
			for _, char := range lines[0] {
				if char == ' ' {
					indent.WriteString(" ")
				} else {
					break
				}
			}
		}

		childIndent := indent.String() + "  "

		// Reconstruct the block with conditional wrapper at child indent
		var result strings.Builder
		fmt.Fprintf(&result, "%s{{- if .Values.certManager.enabled }}\n", childIndent)
		for _, line := range lines {
			result.WriteString("  " + line + "\n")
		}
		fmt.Fprintf(&result, "%s{{- end }}", childIndent)
		return result.String()
	}
	return match
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

// ExtractContainerNames returns the set of container and initContainer names declared in a
// Deployment (or any Pod-template-bearing resource).
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
