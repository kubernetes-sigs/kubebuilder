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

var _ machinery.Template = &Template{}

// Template scaffolds both MutatingWebhookConfiguration and ValidatingWebhookConfiguration for the Helm chart
type Template struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	MutatingWebhooks   []DataWebhook
	ValidatingWebhooks []DataWebhook
}

// SetTemplateDefaults sets default configuration for the webhook template
func (f *Template) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", "templates", "webhook", "webhooks.yaml")
	}

	f.TemplateBody = webhookTemplate
	f.IfExistsAction = machinery.OverwriteFile
	return nil
}

// DataWebhook helps generate manifests based on the data gathered from the kustomize files
type DataWebhook struct {
	ServiceName             string
	Name                    string
	Path                    string
	Type                    string
	FailurePolicy           string
	SideEffects             string
	AdmissionReviewVersions []string
	Rules                   []DataWebhookRule
}

// DataWebhookRule helps generate manifests based on the data gathered from the kustomize files
type DataWebhookRule struct {
	Operations  []string
	APIGroups   []string
	APIVersions []string
	Resources   []string
}

const webhookTemplate = `{{` + "`" + `{{- if .Values.webhook.enable }}` + "`" + `}}

{{- if .MutatingWebhooks }}
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
  {{- range .MutatingWebhooks }}
  - name: {{ .Name }}
    clientConfig:
      service:
        name: {{ .ServiceName }}
        namespace: {{ "{{ .Release.Namespace }}" }}
        path: {{ .Path }}
    failurePolicy: {{ .FailurePolicy }}
    sideEffects: {{ .SideEffects }}
    admissionReviewVersions:
      {{- range .AdmissionReviewVersions }}
      - {{ . }}
      {{- end }}
    rules:
      {{- range .Rules }}
      - operations:
          {{- range .Operations }}
          - {{ . }}
          {{- end }}
        apiGroups:
          {{- range .APIGroups }}
          - {{ . }}
          {{- end }}
        apiVersions:
          {{- range .APIVersions }}
          - {{ . }}
          {{- end }}
        resources:
          {{- range .Resources }}
          - {{ . }}
          {{- end }}
      {{- end -}}
  {{- end }}
{{- end }}
{{- if and .MutatingWebhooks .ValidatingWebhooks }}
---
{{- end }}
{{- if .ValidatingWebhooks }}
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: {{ .ProjectName }}-validating-webhook-configuration
  namespace: {{ "{{ .Release.Namespace }}" }}
  annotations:
    {{` + "`" + `{{- if .Values.certmanager.enable }}` + "`" + `}}
    cert-manager.io/inject-ca-from: "{{` + "`" + `{{ $.Release.Namespace }}` + "`" + `}}/serving-cert"
    {{` + "`" + `{{- end }}` + "`" + `}}
  labels:
    {{ "{{- include \"chart.labels\" . | nindent 4 }}" }}
webhooks:
  {{- range .ValidatingWebhooks }}
  - name: {{ .Name }}
    clientConfig:
      service:
        name: {{ .ServiceName }}
        namespace: {{ "{{ .Release.Namespace }}" }}
        path: {{ .Path }}
    failurePolicy: {{ .FailurePolicy }}
    sideEffects: {{ .SideEffects }}
    admissionReviewVersions:
      {{- range .AdmissionReviewVersions }}
      - {{ . }}
      {{- end }}
    rules:
      {{- range .Rules }}
      - operations:
          {{- range .Operations }}
          - {{ . }}
          {{- end }}
        apiGroups:
          {{- range .APIGroups }}
          - {{ . }}
          {{- end }}
        apiVersions:
          {{- range .APIVersions }}
          - {{ . }}
          {{- end }}
        resources:
          {{- range .Resources }}
          - {{ . }}
          {{- end }}
      {{- end }}
  {{- end }}
{{- end }}
{{` + "`" + `{{- end }}` + "`" + `}}
`
