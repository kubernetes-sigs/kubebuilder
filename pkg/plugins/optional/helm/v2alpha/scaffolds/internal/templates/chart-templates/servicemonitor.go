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

const defaultOutputDir = "dist"

var _ machinery.Template = &ServiceMonitor{}

// ServiceMonitor scaffolds a ServiceMonitor for Prometheus monitoring in the Helm chart
type ServiceMonitor struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// ServiceName is the full name of the metrics service, derived from Kustomize
	ServiceName string

	// OutputDir specifies the output directory for the chart
	OutputDir string
}

// SetTemplateDefaults implements machinery.Template
func (f *ServiceMonitor) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = defaultOutputDir
		}
		f.Path = filepath.Join(outputDir, "chart", "templates", "monitoring", "servicemonitor.yaml")
	}

	// Replace {{ .Chart.Name }} placeholders with actual project name
	chartName := f.ProjectName
	f.TemplateBody = fmt.Sprintf(serviceMonitorTemplate, chartName, chartName, chartName, chartName)

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const serviceMonitorTemplate = `{{` + "`" + `{{- if .Values.prometheus.enable }}` + "`" + `}}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
    control-plane: controller-manager
  name: ` +
	`{{ "{{ include \"%s.resourceName\" " }}` +
	`{{ "(dict \"suffix\" \"controller-manager-metrics-monitor\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  endpoints:
    - path: /metrics
      port: https
      scheme: https
      bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
      tlsConfig:
        {{ "{{- if .Values.certManager.enable }}" }}
        serverName: ` +
	`{{ "{{ include \"%s.resourceName\" " }}` +
	`{{ "(dict \"suffix\" \"controller-manager-metrics-service\" \"context\" $) }}" }}.` +
	`{{ "{{ .Release.Namespace }}" }}.svc
        # Apply secure TLS configuration with cert-manager
        insecureSkipVerify: false
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
        # Development/Test mode (insecure configuration)
        insecureSkipVerify: true
        {{ "{{- end }}" }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
      control-plane: controller-manager
{{` + "`" + `{{- end }}` + "`" + `}}
`
