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
	// Force if true allows overwriting the scaffolded file
	Force bool
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

	f.TemplateBody = f.generateHelpersTemplate()

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

// generateHelpersTemplate creates the _helpers.tpl content with project-specific template names
func (f *HelmHelpers) generateHelpersTemplate() string {
	// Use project name as prefix (e.g., "project-v4-with-plugins")
	// This creates templates like "project-v4-with-plugins.name" instead of generic "chart.name"
	// preventing collisions when chart is used as a Helm dependency
	prefix := f.ProjectName

	return fmt.Sprintf(helmHelpersTemplate, prefix, prefix, prefix, prefix, prefix)
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
{{` + "`" + `{{- .Release.Namespace }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}

{{` + "`" + `{{/*
Resource name with proper truncation for Kubernetes 63-character limit.
Takes a dict with:
  - .suffix: Resource name suffix (e.g., "metrics", "webhook")
  - .context: Template context (root context with .Values, .Release, etc.)
Dynamically calculates safe truncation to ensure total name length <= 63 chars.
*/}}` + "`" + `}}
{{` + "`" + `{{- define "%s.resourceName" -}}` + "`" + `}}
{{` + "`" + `{{- $fullname := include "%s.fullname" .context }}` + "`" + `}}
{{` + "`" + `{{- $suffix := .suffix }}` + "`" + `}}
{{` + "`" + `{{- $maxLen := sub 62 (len $suffix) | int }}` + "`" + `}}
{{` + "`" + `{{- if gt (len $fullname) $maxLen }}` + "`" + `}}
{{` + "`" + `{{- printf "%%s-%%s" (trunc $maxLen $fullname | trimSuffix "-") $suffix ` +
	`| trunc 63 | trimSuffix "-" }}` + "`" + `}}
{{` + "`" + `{{- else }}` + "`" + `}}
{{` + "`" + `{{- printf "%%s-%%s" $fullname $suffix | trunc 63 | trimSuffix "-" }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
{{` + "`" + `{{- end }}` + "`" + `}}
`
