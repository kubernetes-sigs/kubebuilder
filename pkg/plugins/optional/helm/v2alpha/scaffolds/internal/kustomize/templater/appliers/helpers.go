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

// ManagerField locates one field of the manager container in a Deployment's YAML text.
type ManagerField struct {
	// Line is the index of the line declaring the field.
	Line int
	// Indent is the field's own indent, which for a folded field is the position past the dash.
	Indent string
	// Folded reports that the field is written onto the container's sequence dash, as "- env:".
	Folded bool
	// Inline reports that the value sits on the declaring line, as "env: []" or "env: null".
	Inline bool
}

// FindManagerField returns the manager container's field of the given name.
//
// Identity here is structural, never textual. A line declares the field only when it sits at the
// container's own child indentation, or is folded onto the container's sequence dash as the
// container's first field. A line that merely trims to "<name>:" is a value - an argument, a
// command word, a line inside a block scalar - and replacing it would rewrite something the user
// wrote. Only indentation tells the two apart, so a matcher must not trim the line before deciding.
func FindManagerField(yamlContent, name string) (ManagerField, bool) {
	rangeStart, rangeEnd := FindManagerContainerRange(yamlContent)
	if rangeStart < 0 {
		return ManagerField{}, false
	}

	lines := strings.Split(yamlContent, "\n")
	dashIndent, dashIndentLen := LeadingWhitespace(lines[rangeStart])
	childIndent := dashIndent + "  "
	declaration := name + ":"

	for i := rangeStart; i <= rangeEnd && i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		// The container's first field may share the line with the sequence dash. That is the only
		// line where a field is allowed to sit at the dash's indent rather than the child indent.
		if i == rangeStart {
			body, isItem := strings.CutPrefix(trimmed, "- ")
			if !isItem {
				continue
			}
			if rest, ok := strings.CutPrefix(body, declaration); ok {
				return ManagerField{
					Line:   i,
					Indent: childIndent,
					Folded: true,
					Inline: strings.TrimSpace(rest) != "",
				}, true
			}
			continue
		}

		lineIndent, lineIndentLen := LeadingWhitespace(lines[i])
		if lineIndentLen != dashIndentLen+2 {
			continue
		}
		if rest, ok := strings.CutPrefix(trimmed, declaration); ok {
			return ManagerField{Line: i, Indent: lineIndent, Inline: strings.TrimSpace(rest) != ""}, true
		}
	}

	return ManagerField{}, false
}

// hasManagerEnvBlock reports whether the manager container already carries the generated env block.
//
// The marker is a Helm action, so it is matched as a complete line at the container's own child
// indentation - never as a substring of the document. The same text can legitimately appear as user
// data: an annotation, a value, an argument, a line inside a block scalar. EscapeExistingTemplateSyntax
// preserves that verbatim inside a larger line, so a substring test reads it as "already generated"
// and leaves the container's literal env in place. Escaping is right to preserve the input; telling a
// generated action apart from an escaped literal is this function's job.
func hasManagerEnvBlock(yamlContent string) bool {
	start, end := FindManagerContainerRange(yamlContent)
	if start < 0 {
		return false
	}

	lines := strings.Split(yamlContent, "\n")
	dashIndent, _ := LeadingWhitespace(lines[start])
	childIndent := dashIndent + "  "

	for i := start; i <= end && i < len(lines); i++ {
		indent, _ := LeadingWhitespace(lines[i])
		if indent != childIndent {
			continue
		}
		// At the container's child indent, valid YAML is "key: value" or "- item". A line that is
		// nothing but an action - or the escaped literal an earlier pass made of one - can only be
		// something this plugin generated, never user data.
		switch strings.TrimSpace(lines[i]) {
		case envBlockMarker, envBlockMarkerEscaped:
			return true
		}
	}

	return false
}

// ReplaceManagerField swaps a manager field - its declaration and whatever value follows it - for
// block. A field folded onto the container's sequence dash leaves the dash behind as a bare
// sequence entry, which is valid YAML whose mapping follows indented; dropping it would merge the
// container into its predecessor.
func ReplaceManagerField(yamlContent string, field ManagerField, block []string) string {
	lines := strings.Split(yamlContent, "\n")

	end := field.Line + 1
	if !field.Inline {
		end = blockValueEnd(lines, end, len(field.Indent))
	}

	out := append([]string{}, lines[:field.Line]...)
	if field.Folded {
		dashIndent, _ := LeadingWhitespace(lines[field.Line])
		out = append(out, dashIndent+"-")
	}
	out = append(out, block...)
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n")
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

	listLine, listIndent := findPodSpecContainers(lines)
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
		// A sequence item is "- field: value" or a bare "-" whose mapping follows indented. The
		// bare form appears once env is templated in place of a folded `- env:` declaration; without
		// it the manager container stops being found and every later applier loses its scope.
		if indent < listIndent || (trimmed != "-" && !strings.HasPrefix(trimmed, "- ")) {
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

// mappingKey is one entry of the mapping path a line is nested under.
type mappingKey struct {
	name   string
	indent int
}

// findPodSpecContainers returns the line index and indent of the pod template's containers list -
// spec.template.spec.containers, and nothing else that happens to be spelled the same.
//
// Identity is the mapping path, never the first line that reads like "containers:". A Deployment
// carries that text as ordinary data: a last-applied-configuration annotation holds an entire
// object, an argument or a probe command may be written as a block scalar, and a field nested
// deeper in the pod spec may reuse the name. Taking any of those for the pod spec aims every
// applier at whatever follows it, which leaves the manager's fields literal and rewrites something
// the user wrote instead.
//
// The document is walked as text rather than parsed, because by the time this runs the earlier
// appliers have woven Helm actions through it and it is no longer valid YAML.
func findPodSpecContainers(lines []string) (int, int) {
	path := make([]mappingKey, 0, len(podTemplateContainersPath))
	blockScalarIndent := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Neither declares a mapping key, and both can carry a colon that would otherwise be read
		// as one. A Helm action is indented for the reader rather than for the structure, so its
		// indent says nothing about where it sits; a comment says nothing at all.
		if trimmed == "" || strings.HasPrefix(trimmed, "{{") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		_, indent := LeadingWhitespace(line)

		// A block scalar's content is a value however much it reads like YAML, so it is skipped
		// whole: it ends at the first line no deeper than the field that owns it.
		if blockScalarIndent >= 0 {
			if indent > blockScalarIndent {
				continue
			}
			blockScalarIndent = -1
		}

		// A key folded onto a sequence dash sits past the dash, and that is where its indent is.
		if body, isItem := strings.CutPrefix(trimmed, "- "); isItem {
			indent += len(trimmed) - len(body)
			trimmed = body
		}

		// Leaving a mapping ends every path entry it contained.
		for len(path) > 0 && indent <= path[len(path)-1].indent {
			path = path[:len(path)-1]
		}

		name, value, isMapping := strings.Cut(trimmed, ":")

		// "key: |" and a bare "- |" item both open a block scalar. Every modifier - |-, >-, |2 -
		// follows the indicator, so the first character is what decides.
		header := trimmed
		if isMapping {
			header = strings.TrimSpace(value)
		}
		if strings.HasPrefix(header, "|") || strings.HasPrefix(header, ">") {
			blockScalarIndent = indent
		}
		if !isMapping {
			continue
		}

		// The whole path, so the key is matched as a direct child of the field that must contain
		// it: a containers: nested one level deeper leaves a path of the wrong length and no match.
		path = append(path, mappingKey{name: name, indent: indent})
		if slices.EqualFunc(path, podTemplateContainersPath, mappingKeyIs) {
			return i, indent
		}
	}

	return -1, -1
}

func mappingKeyIs(key mappingKey, name string) bool { return key.name == name }

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
