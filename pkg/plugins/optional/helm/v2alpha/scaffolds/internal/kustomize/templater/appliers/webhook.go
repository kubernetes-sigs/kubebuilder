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

import "strings"

// TemplateWebhookConfiguration exposes admission webhook settings through Helm values.
// Fields already present in the Kustomize output are left untouched so that webhook patch
// markers remain authoritative for projects that have configured them explicitly.
func TemplateWebhookConfiguration(yamlContent string) string {
	const webhookRule = "  rules:\n"
	if !strings.Contains(yamlContent, "webhooks:\n") || !strings.Contains(yamlContent, webhookRule) {
		return yamlContent
	}

	var additions strings.Builder
	if !strings.Contains(yamlContent, "  namespaceSelector:") {
		additions.WriteString("  {{- with .Values.webhook.namespaceSelector }}\n")
		additions.WriteString("  namespaceSelector:\n")
		additions.WriteString("    {{- toYaml . | nindent 4 }}\n")
		additions.WriteString("  {{- end }}\n")
	}
	if !strings.Contains(yamlContent, "  objectSelector:") {
		additions.WriteString("  {{- with .Values.webhook.objectSelector }}\n")
		additions.WriteString("  objectSelector:\n")
		additions.WriteString("    {{- toYaml . | nindent 4 }}\n")
		additions.WriteString("  {{- end }}\n")
	}
	if !strings.Contains(yamlContent, "  matchConditions:") {
		additions.WriteString("  {{- with .Values.webhook.matchConditions }}\n")
		additions.WriteString("  matchConditions:\n")
		additions.WriteString("    {{- toYaml . | nindent 4 }}\n")
		additions.WriteString("  {{- end }}\n")
	}
	if !strings.Contains(yamlContent, "  timeoutSeconds:") {
		additions.WriteString("  timeoutSeconds: {{ .Values.webhook.timeoutSeconds }}\n")
	}

	return strings.ReplaceAll(yamlContent, webhookRule, additions.String()+webhookRule)
}
