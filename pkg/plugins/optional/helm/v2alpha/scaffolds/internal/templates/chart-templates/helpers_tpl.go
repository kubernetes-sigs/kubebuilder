/*
Copyright 2025 The Kubernetes Authors.

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

	// OutputDir specifies the output directory for the chart
	OutputDir string
}

// SetTemplateDefaults sets the default template configuration
func (f *HelmHelpers) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = "dist"
		}
		f.Path = filepath.Join(outputDir, "chart", "templates", "_helpers.tpl")
	}

	f.TemplateBody = helmHelpersTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

const helmHelpersTemplate = `{{` + "`" + `{{/*
Chart name based on project name.
Truncated to 63 characters for Kubernetes compatibility.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "chart.name" -}}` + "`" + `}}
{{` + "`" + `{{- if .Chart }}` + "`" + `}}
  {{` + "`" + `{{- if .Chart.Name }}` + "`" + `}}
    {{` + "`" + `{{- .Chart.Name | trunc 63 | trimSuffix "-" }}` + "`" + `}}
  {{` + "`" + `{{- else }}` + "`" + `}}
    {{ .ProjectName }}
  {{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- else }}` + "`" + `}}
  {{ .ProjectName }}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{` + "`" + `{{/*
Full name of the chart (with release name prefix).
Combines release name with chart name.
Truncated to 63 characters for Kubernetes compatibility.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "chart.fullname" -}}` + "`" + `}}
{{` + "`" + `{{- $name := include "chart.name" . }}` + "`" + `}}
{{` + "`" + `{{- if contains $name .Release.Name }}` + "`" + `}}
{{` + "`" + `{{- .Release.Name | trunc 63 | trimSuffix "-" }}` + "`" + `}}
{{` + "`" + `{{- else }}` + "`" + `}}
{{` + "`" + `{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{` + "`" + `{{/*
Namespace for generated references.
Always uses the Helm release namespace.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "chart.namespaceName" -}}` + "`" + `}}
{{` + "`" + `{{ .Release.Namespace }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}



{{` + "`" + `{{/*
Service name with proper truncation for Kubernetes 63-character limit.
Takes a context with .suffix for the service type (e.g., "webhook-service").
If fullname + suffix exceeds 63 chars, truncates fullname to 45 chars.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "chart.serviceName" -}}` + "`" + `}}
{{` + "`" + `{{- $fullname := include "chart.fullname" .context -}}` + "`" + `}}
{{` + "`" + `{{- if gt (len $fullname) 45 -}}` + "`" + `}}
{{` + "`" + `{{- printf "%s-%s" (trunc 45 $fullname | trimSuffix "-") .suffix ` +
	`| trunc 63 | trimSuffix "-" -}}` + "`" + `}}
{{` + "`" + `{{- else -}}` + "`" + `}}
{{` + "`" + `{{- printf "%s-%s" $fullname .suffix | trunc 63 | trimSuffix "-" -}}` + "`" + `}}
{{` + "`" + `{{- end -}}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{` + "`" + `{{/*
Common labels for Helm charts.
Includes app version, chart version, app name, instance, and managed-by labels.
*/}}` + "`" + `}}
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

{{` + "`" + `{{/*
Selector labels for matching pods and services.
Only includes name and instance for consistent selection.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "chart.selectorLabels" -}}` + "`" + `}}
app.kubernetes.io/name: {{` + "`" + `{{ include "chart.name" . }}` + "`" + `}}
app.kubernetes.io/instance: {{` + "`" + `{{ .Release.Name }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
`
