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
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

var _ machinery.Template = &ServiceMonitor{}

// ServiceMonitor scaffolds a ServiceMonitor for Prometheus monitoring in the Helm chart
type ServiceMonitor struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// ServiceName is the full name of the metrics service, derived from Kustomize
	ServiceName string

	// OutputDir specifies the output directory for the chart
	OutputDir string
	// Force if true allows overwriting the scaffolded file
	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *ServiceMonitor) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = common.DefaultOutputDir
		}
		f.Path = filepath.Join(outputDir, "chart", "templates", "prometheus", "controller-manager-metrics-monitor.yaml")
	}

	chartName := f.ProjectName
	f.TemplateBody = fmt.Sprintf(serviceMonitorTemplate, chartName, chartName, chartName, chartName)

	// User-owned file: preserve local edits on regeneration unless --force is set.
	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

const serviceMonitorTemplate = `{{` + "`" +
	`{{- if and .Values.prometheus.enabled .Values.metrics.enabled }}` + "`" + `}}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    {{ "{{- $labels := .Values.prometheus.labels | default dict }}" }}
    {{ "{{- if not (hasKey $labels \"app.kubernetes.io/managed-by\") }}" }}
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    {{ "{{- end }}" }}
    {{ "{{- if not (hasKey $labels \"app.kubernetes.io/name\") }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    {{ "{{- end }}" }}
    {{ "{{- if not (hasKey $labels \"helm.sh/chart\") }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    {{ "{{- end }}" }}
    {{ "{{- if not (hasKey $labels \"app.kubernetes.io/instance\") }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
    {{ "{{- end }}" }}
    {{ "{{- if not (hasKey $labels \"control-plane\") }}" }}
    control-plane: controller-manager
    {{ "{{- end }}" }}
    {{ "{{- with .Values.prometheus.labels }}" }}
    {{ "{{- toYaml . | nindent 4 }}" }}
    {{ "{{- end }}" }}
  {{ "{{- with .Values.prometheus.annotations }}" }}
  annotations:
    {{ "{{- toYaml . | nindent 4 }}" }}
  {{ "{{- end }}" }}
  name: ` +
	`{{ "{{ include \"%s.resourceName\" " }}` +
	`{{ "(dict \"suffix\" \"controller-manager-metrics-monitor\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  endpoints:
  - {{ "{{- if .Values.metrics.secure }}" }}
    bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    {{ "{{- end }}" }}
    path: /metrics
    port: {{ "{{ if .Values.metrics.secure }}https{{ else }}http{{ end }}" }}
    scheme: {{ "{{ if .Values.metrics.secure }}https{{ else }}http{{ end }}" }}
    {{ "{{- if .Values.metrics.secure }}" }}
    tlsConfig:
      serverName: ` +
	`{{ "{{ include \"%s.resourceName\" " }}` +
	`{{ "(dict \"suffix\" \"controller-manager-metrics-service\" \"context\" $) }}" }}.` +
	`{{ "{{ .Release.Namespace }}" }}.svc
      {{ "{{- if .Values.certManager.enabled }}" }}
      ca:
        secret:
          name: metrics-server-cert
          key: ca.crt
      cert:
        secret:
          name: metrics-server-cert
          key: tls.crt
      keySecret:
        name: metrics-server-cert
        key: tls.key
      {{ "{{- else }}" }}
      insecureSkipVerify: true
      {{ "{{- end }}" }}
    {{ "{{- end }}" }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
      control-plane: controller-manager
{{` + "`" + `{{- end }}` + "`" + `}}
`
