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
	"strings"

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

func (f *HelmHelpers) generateHelpersTemplate() string {
	escape := func(s string) string {
		return "{{`" + s + "`}}"
	}

	return strings.Join([]string{
		escape("{{/*"),
		"Expand the name of the chart.",
		escape("*/}}"),
		escape("{{- define \"") + f.ProjectName + escape(".name\" -}}"),
		escape("{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix \"-\" }}"),
		escape("{{- end }}"),
		"",
		escape("{{/*"),
		"Create a default fully qualified app name.",
		"We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).",
		"If release name contains chart name it will be used as a full name.",
		escape("*/}}"),
		escape("{{- define \"") + f.ProjectName + escape(".fullname\" -}}"),
		escape("{{- if .Values.fullnameOverride }}"),
		escape("{{- .Values.fullnameOverride | trunc 63 | trimSuffix \"-\" }}"),
		escape("{{- else }}"),
		escape("{{- $name := default .Chart.Name .Values.nameOverride }}"),
		escape("{{- if contains $name .Release.Name }}"),
		escape("{{- .Release.Name | trunc 63 | trimSuffix \"-\" }}"),
		escape("{{- else }}"),
		escape("{{- printf \"%s-%s\" .Release.Name $name | trunc 63 | trimSuffix \"-\" }}"),
		escape("{{- end }}"),
		escape("{{- end }}"),
		escape("{{- end }}"),
		"",
		escape("{{/*"),
		"Namespace for generated references.",
		"Always uses the Helm release namespace.",
		escape("*/}}"),
		escape("{{- define \"") + f.ProjectName + escape(".namespaceName\" -}}"),
		escape("{{- .Release.Namespace }}"),
		escape("{{- end }}"),
		"",
		escape("{{/*"),
		"Resource name with proper truncation for Kubernetes 63-character limit.",
		"Takes a dict with .suffix (resource name suffix) and .context (template context).",
		"Dynamically calculates safe truncation length based on suffix to ensure total <= 63 chars.",
		"Generic helper that works for any resource type (Service, Role, Certificate, etc.).",
		escape("*/}}"),
		escape("{{- define \"") + f.ProjectName + escape(".resourceName\" -}}"),
		escape("{{- $fullname := include \"") + f.ProjectName + escape(".fullname\" .context }}"),
		escape("{{- $suffix := .suffix }}"),
		escape("{{- $maxLen := sub 62 (len $suffix) | int }}"),
		escape("{{- if gt (len $fullname) $maxLen }}"),
		escape("{{- printf \"%s-%s\" (trunc $maxLen $fullname | trimSuffix \"-\") $suffix | trunc 63 | trimSuffix \"-\" }}"),
		escape("{{- else }}"),
		escape("{{- printf \"%s-%s\" $fullname $suffix | trunc 63 | trimSuffix \"-\" }}"),
		escape("{{- end }}"),
		escape("{{- end }}"),
		"",
		escape("{{/*"),
		"Common labels for Helm charts.",
		"Includes app version, chart version, app name, instance, and managed-by labels.",
		escape("*/}}"),
		escape("{{- define \"") + f.ProjectName + escape(".labels\" -}}"),
		escape("{{- if .Chart.AppVersion -}}"),
		escape("app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}"),
		escape("{{- end }}"),
		escape("{{- if .Chart.Version }}"),
		escape("helm.sh/chart: {{ .Chart.Version | quote }}"),
		escape("{{- end }}"),
		escape("app.kubernetes.io/name: {{ include \"") + f.ProjectName + escape(".name\" . }}"),
		escape("app.kubernetes.io/instance: {{ .Release.Name }}"),
		escape("app.kubernetes.io/managed-by: {{ .Release.Service }}"),
		escape("{{- end }}"),
		"",
		escape("{{/*"),
		"Selector labels for matching pods and services.",
		"Only includes name and instance for consistent selection.",
		escape("*/}}"),
		escape("{{- define \"") + f.ProjectName + escape(".selectorLabels\" -}}"),
		escape("app.kubernetes.io/name: {{ include \"") + f.ProjectName + escape(".name\" . }}"),
		escape("app.kubernetes.io/instance: {{ .Release.Name }}"),
		escape("{{- end }}"),
	}, "\n") + "\n" // Add EOF new line
}
