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

package prometheus

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Monitor{}

// Monitor scaffolds the ServiceMonitor for Prometheus in the Helm chart
type Monitor struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults sets the default template configuration
func (f *Monitor) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("dist", "chart", "templates", "prometheus", "monitor.yaml")
	}

	f.TemplateBody = monitorTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const monitorTemplate = `# To integrate with Prometheus.
{{ "{{- if .Values.prometheus.enable }}" }}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    {{ "{{- include \"chart.labels\" . | nindent 4 }}" }}
    control-plane: controller-manager
  name: {{ .ProjectName }}-controller-manager-metrics-monitor
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  endpoints:
    - path: /metrics
      port: https
      scheme: https
      bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
      tlsConfig:
        {{ "{{- if .Values.certmanager.enable }}" }}
        serverName: {{ .ProjectName }}-controller-manager-metrics-service.{{ "{{ .Release.Namespace }}" }}.svc
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
      control-plane: controller-manager
{{ "{{- end }}" }}
`
