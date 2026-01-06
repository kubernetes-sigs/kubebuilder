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
	"fmt"
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

	// Use project name as template prefix to avoid collisions when chart is used as dependency
	// This follows the pattern used by Bitnami, cert-manager, and other production Helm charts
	f.TemplateBody = f.generateHelpersTemplate()

	f.IfExistsAction = machinery.SkipFile

	return nil
}

// generateHelpersTemplate creates the _helpers.tpl content with project-specific template names
func (f *HelmHelpers) generateHelpersTemplate() string {
	// Use project name as prefix (e.g., "project-v4-with-plugins")
	// This creates templates like "project-v4-with-plugins.name" instead of generic "chart.name"
	// preventing collisions when chart is used as a Helm dependency
	prefix := f.ProjectName

	return fmt.Sprintf(helmHelpersTemplate, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix)
}

const helmHelpersTemplate = `{{` + "`" + `{{/*
Expand the name of the chart.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "%s.name" -}}` + "`" + `}}
{{` + "`" + `{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{` + "`" + `{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "%s.fullname" -}}` + "`" + `}}
{{` + "`" + `{{- if .Values.fullnameOverride }}` + "`" + `}}
{{` + "`" + `{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}` + "`" + `}}
{{` + "`" + `{{- else }}` + "`" + `}}
{{` + "`" + `{{- $name := default .Chart.Name .Values.nameOverride }}` + "`" + `}}
{{` + "`" + `{{- if contains $name .Release.Name }}` + "`" + `}}
{{` + "`" + `{{- .Release.Name | trunc 63 | trimSuffix "-" }}` + "`" + `}}
{{` + "`" + `{{- else }}` + "`" + `}}
{{` + "`" + `{{- printf "%%s-%%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{` + "`" + `{{/*
Namespace for generated references.
Always uses the Helm release namespace.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "%s.namespaceName" -}}` + "`" + `}}
{{` + "`" + `{{ .Release.Namespace }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}



{{` + "`" + `{{/*
Resource name with proper truncation for Kubernetes 63-character limit.
Takes a dict with .suffix (resource name suffix) and .context (template context).
Dynamically calculates safe truncation length based on suffix to ensure total <= 63 chars.
Generic helper that works for any resource type (Service, Role, Certificate, etc.).
*/}}` + "`" + `}}
{{` + "`" + `{{- define "%s.resourceName" -}}` + "`" + `}}
{{` + "`" + `{{- $fullname := include "%s.fullname" .context -}}` + "`" + `}}
{{` + "`" + `{{- $suffix := .suffix -}}` + "`" + `}}
{{` + "`" + `{{- $maxLen := sub 62 (len $suffix) | int -}}` + "`" + `}}
{{` + "`" + `{{- if gt (len $fullname) $maxLen -}}` + "`" + `}}
{{` + "`" + `{{- printf "%%s-%%s" (trunc $maxLen $fullname | trimSuffix "-") $suffix ` +
	`| trunc 63 | trimSuffix "-" -}}` + "`" + `}}
{{` + "`" + `{{- else -}}` + "`" + `}}
{{` + "`" + `{{- printf "%%s-%%s" $fullname $suffix | trunc 63 | trimSuffix "-" -}}` + "`" + `}}
{{` + "`" + `{{- end -}}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{` + "`" + `{{/*
Common labels for Helm charts.
Includes app version, chart version, app name, instance, and managed-by labels.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "%s.labels" -}}` + "`" + `}}
{{` + "`" + `{{- if .Chart.AppVersion -}}` + "`" + `}}
app.kubernetes.io/version: {{` + "`" + `{{ .Chart.AppVersion | quote }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- if .Chart.Version }}` + "`" + `}}
helm.sh/chart: {{` + "`" + `{{ .Chart.Version | quote }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
app.kubernetes.io/name: {{` + "`" + `{{ include "%s.name" . }}` + "`" + `}}
app.kubernetes.io/instance: {{` + "`" + `{{ .Release.Name }}` + "`" + `}}
app.kubernetes.io/managed-by: {{` + "`" + `{{ .Release.Service }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{` + "`" + `{{/*
Selector labels for matching pods and services.
Only includes name and instance for consistent selection.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "%s.selectorLabels" -}}` + "`" + `}}
app.kubernetes.io/name: {{` + "`" + `{{ include "%s.name" . }}` + "`" + `}}
app.kubernetes.io/instance: {{` + "`" + `{{ .Release.Name }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
`
