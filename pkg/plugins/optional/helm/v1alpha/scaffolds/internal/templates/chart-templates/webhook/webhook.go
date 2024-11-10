/*
Copyright 2024 The Kubernetes Authors.

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

package webhook

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &WebhookTemplate{}

// WebhookTemplate scaffolds both MutatingWebhookConfiguration and ValidatingWebhookConfiguration for the Helm chart
type WebhookTemplate struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults sets default configuration for the webhook template
func (f *WebhookTemplate) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", "templates", "webhook", "webhooks.yaml")
	}

	f.TemplateBody = webhookTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const webhookTemplate = `{{` + "`" + `{{- if .Values.webhook.enable }}` + "`" + `}}

apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: {{ .ProjectName }}-mutating-webhook-configuration
  namespace: {{ "{{ .Release.Namespace }}" }}
  annotations:
    {{` + "`" + `{{- if .Values.certmanager.enable }}` + "`" + `}}
    cert-manager.io/inject-ca-from: "{{` + "`" + `{{ $.Release.Namespace }}` + "`" + `}}/serving-cert"
    {{` + "`" + `{{- end }}` + "`" + `}}
  labels:
    {{ "{{- include \"chart.labels\" . | nindent 4 }}" }}
webhooks:
  {{` + "`" + `{{- range .Values.webhook.services }}` + "`" + `}}
  {{` + "`" + `{{- if eq .type "mutating" }}` + "`" + `}}
  - name: {{` + "`" + `{{ .name }}` + "`" + `}}
    clientConfig:
      service:
        name: {{ .ProjectName }}-webhook-service
        namespace: {{` + "`" + `{{ $.Release.Namespace }}` + "`" + `}}
        path: {{` + "`" + `{{ .path }}` + "`" + `}}
    failurePolicy: {{` + "`" + `{{ .failurePolicy }}` + "`" + `}}
    sideEffects: {{` + "`" + `{{ .sideEffects }}` + "`" + `}}
    admissionReviewVersions:
      {{` + "`" + `{{- range .admissionReviewVersions }}` + "`" + `}}
      - {{` + "`" + `{{ . }}` + "`" + `}}
      {{` + "`" + `{{- end }}` + "`" + `}}
    rules:
      {{` + "`" + `{{- range .rules }}` + "`" + `}}
      - operations:
          {{` + "`" + `{{- range .operations }}` + "`" + `}}
          - {{` + "`" + `{{ . }}` + "`" + `}}
          {{` + "`" + `{{- end }}` + "`" + `}}
        apiGroups:
          {{` + "`" + `{{- range .apiGroups }}` + "`" + `}}
          - {{` + "`" + `{{ . }}` + "`" + `}}
          {{` + "`" + `{{- end }}` + "`" + `}}
        apiVersions:
          {{` + "`" + `{{- range .apiVersions }}` + "`" + `}}
          - {{` + "`" + `{{ . }}` + "`" + `}}
          {{` + "`" + `{{- end }}` + "`" + `}}
        resources:
          {{` + "`" + `{{- range .resources }}` + "`" + `}}
          - {{` + "`" + `{{ . }}` + "`" + `}}
          {{` + "`" + `{{- end }}` + "`" + `}}
      {{` + "`" + `{{- end }}` + "`" + `}}
  {{` + "`" + `{{- end }}` + "`" + `}}
  {{` + "`" + `{{- end }}` + "`" + `}}
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: {{ .ProjectName }}-validating-webhook-configuration
  namespace: {{ "{{ .Release.Namespace }}" }}
  annotations:
    {{` + "`" + `{{- if .Values.certmanager.enable }}` + "`" + `}}
    cert-manager.io/inject-ca-from: "{{` + "`" + `{{ $.Release.Namespace }}` + "`" + `}}/serving-cert"
    {{` + "`" + `{{- end }}` + "`" + `}}
webhooks:
  {{` + "`" + `{{- range .Values.webhook.services }}` + "`" + `}}
  {{` + "`" + `{{- if eq .type "validating" }}` + "`" + `}}
  - name: {{` + "`" + `{{ .name }}` + "`" + `}}
    clientConfig:
      service:
        name: {{ .ProjectName }}-webhook-service
        namespace: {{` + "`" + `{{ $.Release.Namespace }}` + "`" + `}}
        path: {{` + "`" + `{{ .path }}` + "`" + `}}
    failurePolicy: {{` + "`" + `{{ .failurePolicy }}` + "`" + `}}
    sideEffects: {{` + "`" + `{{ .sideEffects }}` + "`" + `}}
    admissionReviewVersions:
      {{` + "`" + `{{- range .admissionReviewVersions }}` + "`" + `}}
      - {{` + "`" + `{{ . }}` + "`" + `}}
      {{` + "`" + `{{- end }}` + "`" + `}}
    rules:
      {{` + "`" + `{{- range .rules }}` + "`" + `}}
      - operations:
          {{` + "`" + `{{- range .operations }}` + "`" + `}}
          - {{` + "`" + `{{ . }}` + "`" + `}}
          {{` + "`" + `{{- end }}` + "`" + `}}
        apiGroups:
          {{` + "`" + `{{- range .apiGroups }}` + "`" + `}}
          - {{` + "`" + `{{ . }}` + "`" + `}}
          {{` + "`" + `{{- end }}` + "`" + `}}
        apiVersions:
          {{` + "`" + `{{- range .apiVersions }}` + "`" + `}}
          - {{` + "`" + `{{ . }}` + "`" + `}}
          {{` + "`" + `{{- end }}` + "`" + `}}
        resources:
          {{` + "`" + `{{- range .resources }}` + "`" + `}}
          - {{` + "`" + `{{ . }}` + "`" + `}}
          {{` + "`" + `{{- end }}` + "`" + `}}
      {{` + "`" + `{{- end }}` + "`" + `}}
  {{` + "`" + `{{- end }}` + "`" + `}}
  {{` + "`" + `{{- end }}` + "`" + `}}
---
{{` + "`" + `{{- end }}` + "`" + `}}
`
