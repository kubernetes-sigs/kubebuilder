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

package charttemplates

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &HelmHelpers{}

// HelmHelpers scaffolds the _helpers.tpl file for Helm charts
type HelmHelpers struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults sets the default template configuration
func (f *HelmHelpers) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", "templates", "_helpers.tpl")
	}

	f.TemplateBody = helmHelpersTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const helmHelpersTemplate = `{{` + "`" + `{{- define "chart.name" -}}` + "`" + `}}
{{` + "`" + `{{- if .Chart }}` + "`" + `}}
  {{` + "`" + `{{- if .Chart.Name }}` + "`" + `}}
    {{` + "`" + `{{- .Chart.Name | trunc 63 | trimSuffix "-" }}` + "`" + `}}
  {{` + "`" + `{{- else if .Values.nameOverride }}` + "`" + `}}
    {{` + "`" + `{{ .Values.nameOverride | trunc 63 | trimSuffix "-" }}` + "`" + `}}
  {{` + "`" + `{{- else }}` + "`" + `}}
    {{ .ProjectName }}
  {{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- else }}` + "`" + `}}
  {{ .ProjectName }}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{/*
Common labels for the chart.
*/}}
{{` + "`" + `{{- define "chart.labels" -}}` + "`" + `}}
{{` + "`" + `{{- if .Chart.AppVersion -}}` + "`" + `}}
app.kubernetes.io/version: {{` + "`" + `{{ .Chart.AppVersion | quote }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- if .Chart.Version }}` + "`" + `}}
helm.sh/chart: {{` + "`" + `{{ .Chart.Version | quote }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
app.kubernetes.io/name: {{` + "`" + `{{ include "chart.name" . }}` + "`" + `}}
app.kubernetes.io/instance: {{` + "`" + `{{ .Release.Name }}` + "`" + `}}
app.kubernetes.io/managed-by: {{` + "`" + `{{ .Release.Service }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{/*
Selector labels for the chart.
*/}}
{{` + "`" + `{{- define "chart.selectorLabels" -}}` + "`" + `}}
app.kubernetes.io/name: {{` + "`" + `{{ include "chart.name" . }}` + "`" + `}}
app.kubernetes.io/instance: {{` + "`" + `{{ .Release.Name }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{/*
Helper to check if mutating webhooks exist in the services.
*/}}
{{` + "`" + `{{- define "chart.hasMutatingWebhooks" -}}` + "`" + `}}
{{` + "`" + `{{- $hasMutating := false }}` + "`" + `}}
{{` + "`" + `{{- range . }}` + "`" + `}}
  {{` + "`" + `{{- if eq .type "mutating" }}` + "`" + `}}
    {{` + "`" + `$hasMutating = true }}{{- end }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{ $hasMutating }}}}{{- end }}` + "`" + `}}

{{/*
Helper to check if validating webhooks exist in the services.
*/}}
{{` + "`" + `{{- define "chart.hasValidatingWebhooks" -}}` + "`" + `}}
{{` + "`" + `{{- $hasValidating := false }}` + "`" + `}}
{{` + "`" + `{{- range . }}` + "`" + `}}
  {{` + "`" + `{{- if eq .type "validating" }}` + "`" + `}}
    {{` + "`" + `$hasValidating = true }}{{- end }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{ $hasValidating }}}}{{- end }}` + "`" + `}}
`
