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
	// Selector holds the matchLabels lines that select the metrics Service, as the chart renders
	// its labels. When empty, the ServiceMonitor selects the scaffolded labels.
	Selector []string

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
	matchLabels := matchLabelsLines(f.Selector, []string{
		`app.kubernetes.io/name: {{ include "` + chartName + `.name" . }}`,
		`control-plane: controller-manager`,
	})
	f.TemplateBody = fmt.Sprintf(serviceMonitorTemplate, chartName, matchLabels)

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
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%[1]s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
    control-plane: {{ "{{ include \"%[1]s.valueOrDefault\" " }}` +
	`{{ "(list .Values.prometheus.labels \"control-plane\" \"controller-manager\") }}" }}
    {{ "{{- with .Values.prometheus.labels }}" }}
` + `    {{ "{{- with omit . \"app.kubernetes.io/managed-by\" \"app.kubernetes.io/name\" \"helm.sh/chart\" \"app.kubernetes.io/instance\" \"control-plane\" }}" }}` + "\n" + //nolint:lll
	`    {{ "{{- include \"%[1]s.stringMap\" . | nindent 4 }}" }}
    {{ "{{- end }}" }}
    {{ "{{- end }}" }}
  {{ "{{- with .Values.prometheus.annotations }}" }}
  annotations:
    {{ "{{- include \"%[1]s.stringMap\" . | nindent 4 }}" }}
  {{ "{{- end }}" }}
  name: ` +
	`{{ "{{ include \"%[1]s.resourceName\" " }}` +
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
	`{{ "{{ include \"%[1]s.resourceName\" " }}` +
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
%[2]s{{` + "`" + `{{- end }}` + "`" + `}}
`
