/*
Copyright 2020 The Kubernetes Authors.

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

// Monitor scaffolds a file that defines the prometheus service monitor
type Monitor struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Monitor) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "prometheus", "monitor.yaml")
	}

	f.TemplateBody = serviceMonitorTemplate

	return nil
}

//nolint:lll
const serviceMonitorTemplate = `# Prometheus Monitor Service (Metrics)
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: {{ .ProjectName }}
    app.kubernetes.io/managed-by: kustomize
  name: controller-manager-metrics-monitor
  namespace: system
spec:
  endpoints:
    - path: /metrics
      port: https # Ensure this is the name of the port that exposes HTTPS metrics
      scheme: https
      bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
      tlsConfig:
        # TODO(user): The option insecureSkipVerify: true is not recommended for production since it disables
        # certificate verification, exposing the system to potential man-in-the-middle attacks.
        # For production environments, it is recommended to use cert-manager for automatic TLS certificate management.
        # To apply this configuration, enable cert-manager and use the patch located at config/prometheus/servicemonitor_tls_patch.yaml,
        # which securely references the certificate from the 'metrics-server-cert' secret.
        insecureSkipVerify: true
  selector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: {{ .ProjectName }}
`
