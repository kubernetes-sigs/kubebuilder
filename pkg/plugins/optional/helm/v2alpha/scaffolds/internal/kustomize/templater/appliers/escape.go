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
	"unicode"

	"sigs.k8s.io/yaml"
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

// MarshalResource marshals a resource with sigs.k8s.io/yaml. That encoder writes U+0085 (NEL) as is,
// and YAML reads it back as a line break, so a string holding it is written double-quoted with \N.
func MarshalResource(object map[string]any) (string, error) {
	var quoted []string
	copied, ok := replaceNEL(object, &quoted).(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected resource type %T", object)
	}
	out, err := yaml.Marshal(copied)
	if err != nil {
		return "", fmt.Errorf("marshaling the resource: %w", err)
	}
	content := string(out)
	for i, value := range quoted {
		content = strings.Replace(content, nelPlaceholder(i), doubleQuoted(value), 1)
	}
	return content, nil
}

func nelPlaceholder(i int) string {
	return fmt.Sprintf("kubebuilder-nel-placeholder-%d-end", i)
}

// replaceNEL returns a copy of value where each string holding U+0085 is replaced by a placeholder.
func replaceNEL(value any, quoted *[]string) any {
	switch v := value.(type) {
	case string:
		if !strings.ContainsRune(v, '\u0085') {
			return v
		}
		*quoted = append(*quoted, v)
		return nelPlaceholder(len(*quoted) - 1)
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = replaceNEL(item, quoted)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = replaceNEL(item, quoted)
		}
		return out
	}
	return value
}

// doubleQuoted returns s as a one-line YAML double-quoted string.
func doubleQuoted(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\u0085':
			b.WriteString(`\N`)
		case '\u2028':
			b.WriteString(`\L`)
		case '\u2029':
			b.WriteString(`\P`)
		default:
			if unicode.IsControl(r) {
				fmt.Fprintf(&b, `\u%04X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
