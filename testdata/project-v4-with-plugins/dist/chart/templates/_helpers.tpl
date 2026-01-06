{{/*
Expand the name of the chart.
*/}}
{{- define "project-v4-with-plugins.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "project-v4-with-plugins.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Namespace for generated references.
Always uses the Helm release namespace.
*/}}
{{- define "project-v4-with-plugins.namespaceName" -}}
{{ .Release.Namespace }}
{{- end }}



{{/*
Resource name with proper truncation for Kubernetes 63-character limit.
Takes a dict with .suffix (resource name suffix) and .context (template context).
Dynamically calculates safe truncation length based on suffix to ensure total <= 63 chars.
Generic helper that works for any resource type (Service, Role, Certificate, etc.).
*/}}
{{- define "project-v4-with-plugins.resourceName" -}}
{{- $fullname := include "project-v4-with-plugins.fullname" .context -}}
{{- $suffix := .suffix -}}
{{- $maxLen := sub 62 (len $suffix) | int -}}
{{- if gt (len $fullname) $maxLen -}}
{{- printf "%s-%s" (trunc $maxLen $fullname | trimSuffix "-") $suffix | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" $fullname $suffix | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end }}

{{/*
Common labels for Helm charts.
Includes app version, chart version, app name, instance, and managed-by labels.
*/}}
{{- define "project-v4-with-plugins.labels" -}}
{{- if .Chart.AppVersion -}}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- if .Chart.Version }}
helm.sh/chart: {{ .Chart.Version | quote }}
{{- end }}
app.kubernetes.io/name: {{ include "project-v4-with-plugins.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels for matching pods and services.
Only includes name and instance for consistent selection.
*/}}
{{- define "project-v4-with-plugins.selectorLabels" -}}
app.kubernetes.io/name: {{ include "project-v4-with-plugins.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
