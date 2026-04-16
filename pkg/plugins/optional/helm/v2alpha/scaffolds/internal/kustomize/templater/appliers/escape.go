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
)

var (
	// templatePattern matches {{ }} syntax, including those wrapped across lines by yaml.Marshal
	templatePattern = regexp.MustCompile(`(?s)\{\{(.*?)\}\}`)
	// newlineCollapsePattern matches newline followed by whitespace for collapsing yaml line wrapping
	newlineCollapsePattern = regexp.MustCompile(`\n[ \t]+`)
)

// EscapeExistingTemplateSyntax escapes Go template syntax ({{ }}) in YAML to prevent
// Helm from parsing them. Converts existing templates to literal strings that Helm outputs as-is.
//
// Why this is needed:
// Resources from install.yaml may contain {{ }} in string fields that are NOT Helm templates.
// Without escaping, Helm will try to evaluate them and fail. For example:
//
//	CRD default: "Branch: {{ .Spec.Branch }}"  ->  ERROR: .Spec undefined
//
// How it works:
// Wraps non-Helm templates in string literals so Helm outputs them unchanged:
//
//	{{ .Field }}  ->  {{ "{{ .Field }}" }}
//
// When Helm renders this, it outputs the literal string: {{ .Field }}
//
// Smart detection:
// Only escapes templates that DON'T start with Helm keywords:
//   - .Release, .Values, .Chart (Helm built-ins)
//   - include, if, with, range, toYaml (Helm functions)
//
// This means our Helm templates work normally while existing templates are preserved.
func EscapeExistingTemplateSyntax(yamlContent string) string {
	yamlContent = templatePattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
		// Extract content between {{ and }}
		content := strings.TrimPrefix(match, "{{")
		content = strings.TrimSuffix(content, "}}")
		trimmedContent := strings.TrimSpace(content)

		// Check if this is a Helm template (starts with Helm keyword)
		helmPatterns := []string{
			"include ", "- include ",
			".Release.", "- .Release.",
			".Values.", "- .Values.",
			".Chart.", "- .Chart.",
			"toYaml ", "- toYaml ",
			"if ", "- if ",
			"end", "- end",
			"end ", "- end ",
			"with ", "- with ",
			"range ", "- range ",
			"else", "- else",
		}

		// If it's a Helm template, keep it as-is
		for _, pattern := range helmPatterns {
			if strings.HasPrefix(trimmedContent, pattern) {
				return match
			}
		}

		// Otherwise, escape it to preserve as literal text
		// Collapse any newline+indent that sigs.k8s.io/yaml may have introduced via line-wrapping.
		collapsed := newlineCollapsePattern.ReplaceAllString(content, " ")

		// Before re-escaping for Go template string literals, unescape any YAML double-quoted
		// scalar escape sequences. yaml.Marshal emits \" for a literal " inside a double-quoted
		// YAML scalar; without this step the subsequent " to \" replacement double-escapes them to
		// \\" which breaks Helm's Go template parser: \\ becomes one backslash, then the next "
		// closes the string prematurely, leaving tokens like "asset-id" outside where "-" is a
		// bad character (U+002D).
		unescaped := strings.ReplaceAll(collapsed, `\"`, `"`)
		escapedContent := strings.ReplaceAll(unescaped, `"`, `\"`)

		// Wrap in Helm string literal: {{ "{{...}}" }}
		return `{{ "{{` + escapedContent + `}}" }}`
	})

	return yamlContent
}

// CollapseBlankLineAfterIf removes unwanted blank lines after {{- if ... }} and before {{- end }}.
// This ensures clean formatting in generated templates by:
//   - Removing blank line immediately after {{- if ... }}
//   - Removing blank line immediately before {{- end }}
func CollapseBlankLineAfterIf(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	if len(lines) == 0 {
		return yamlContent
	}
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// If current line is an if, and next line is blank, skip the blank
		if strings.Contains(line, "{{- if ") {
			out = append(out, line)
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
				i++ // skip one blank line after if
			}
			continue
		}
		// If current line is blank, and next line is an end, skip the blank
		if strings.TrimSpace(line) == "" && i+1 < len(lines) && strings.Contains(lines[i+1], "{{- end }}") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
