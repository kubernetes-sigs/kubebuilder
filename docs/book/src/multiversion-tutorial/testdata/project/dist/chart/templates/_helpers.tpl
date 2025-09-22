{{/*
Chart name based on project name.
Truncated to 63 characters for Kubernetes compatibility.
*/}}
{{- define "chart.name" -}}
{{- if .Chart }}
  {{- if .Chart.Name }}
    {{- .Chart.Name | trunc 63 | trimSuffix "-" }}
  {{- else }}
    project
  {{- end }}
{{- else }}
  project
{{- end }}
{{- end }}

{{/*
Full name of the chart (with release name prefix).
Combines release name with chart name.
Truncated to 63 characters for Kubernetes compatibility.
*/}}
{{- define "chart.fullname" -}}
{{- $name := include "chart.name" . }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Namespace for generated references.
Always uses the Helm release namespace.
*/}}
{{- define "chart.namespaceName" -}}
{{ .Release.Namespace }}
{{- end }}



{{/*
Service name with proper truncation for Kubernetes 63-character limit.
Takes a context with .suffix for the service type (e.g., "webhook-service").
If fullname + suffix exceeds 63 chars, truncates fullname to 45 chars.
*/}}
{{- define "chart.serviceName" -}}
{{- $fullname := include "chart.fullname" .context -}}
{{- if gt (len $fullname) 45 -}}
{{- printf "%s-%s" (trunc 45 $fullname | trimSuffix "-") .suffix | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" $fullname .suffix | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end }}

{{/*
Common labels for Helm charts.
Includes app version, chart version, app name, instance, and managed-by labels.
*/}}
{{- define "chart.labels" -}}
{{- if .Chart.AppVersion -}}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- if .Chart.Version }}
helm.sh/chart: {{ .Chart.Version | quote }}
{{- end }}
app.kubernetes.io/name: {{ include "chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels for matching pods and services.
Only includes name and instance for consistent selection.
*/}}
{{- define "chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
