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
	"slices"
	"strings"
)

// TemplateWebhookConfiguration exposes admission webhook settings through Helm values.
// Fields already present in the Kustomize output are left untouched so that webhook patch
// markers remain authoritative for projects that have configured them explicitly.
func TemplateWebhookConfiguration(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	webhooksIndex := slices.Index(lines, "webhooks:")
	if webhooksIndex == -1 || webhooksIndex+1 >= len(lines) || !isWebhookItem(lines[webhooksIndex+1]) {
		return yamlContent
	}

	result := append([]string{}, lines[:webhooksIndex+1]...)
	for itemStart := webhooksIndex + 1; itemStart < len(lines) && isWebhookItem(lines[itemStart]); {
		itemEnd := itemStart + 1
		for itemEnd < len(lines) && (strings.HasPrefix(lines[itemEnd], " ") || lines[itemEnd] == "") {
			itemEnd++
		}

		result = append(result, templateWebhookItem(lines[itemStart:itemEnd])...)
		if itemEnd >= len(lines) || !isWebhookItem(lines[itemEnd]) {
			result = append(result, lines[itemEnd:]...)
			break
		}
		itemStart = itemEnd
	}

	return strings.Join(result, "\n")
}

func isWebhookItem(line string) bool {
	return strings.HasPrefix(line, "- ")
}

func templateWebhookItem(lines []string) []string {
	rulesIndex := slices.Index(lines, "  rules:")
	if rulesIndex == -1 {
		return lines
	}

	var additions []string
	if !hasWebhookField(lines, "namespaceSelector") {
		additions = append(additions,
			"  {{- with .Values.webhook.namespaceSelector }}",
			"  namespaceSelector:",
			"    {{- toYaml . | nindent 4 }}",
			"  {{- end }}",
		)
	}
	if !hasWebhookField(lines, "objectSelector") {
		additions = append(additions,
			"  {{- with .Values.webhook.objectSelector }}",
			"  objectSelector:",
			"    {{- toYaml . | nindent 4 }}",
			"  {{- end }}",
		)
	}
	if !hasWebhookField(lines, "matchConditions") {
		additions = append(additions,
			"  {{- with .Values.webhook.matchConditions }}",
			"  matchConditions:",
			"    {{- toYaml . | nindent 4 }}",
			"  {{- end }}",
		)
	}
	if !hasWebhookField(lines, "timeoutSeconds") {
		additions = append(additions,
			"  {{ with .Values.webhook.timeoutSeconds }}",
			"  timeoutSeconds: {{ . }}",
			"  {{ end }}",
		)
	}

	result := make([]string, 0, len(lines)+len(additions))
	result = append(result, lines[:rulesIndex]...)
	result = append(result, additions...)
	result = append(result, lines[rulesIndex:]...)
	return result
}

func hasWebhookField(lines []string, field string) bool {
	prefix := "  " + field + ":"
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}
