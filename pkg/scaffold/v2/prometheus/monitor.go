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

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

// PrometheusMetricsService scaffolds an issuer CR and a certificate CR
type PrometheusServiceMonitor struct {
	input.Input
}

// GetInput implements input.File
func (p *PrometheusServiceMonitor) GetInput() (input.Input, error) {
	if p.Path == "" {
		p.Path = filepath.Join("config", "prometheus", "monitor.yaml")
	}
	p.TemplateBody = monitorTemplate
	return p.Input, nil
}

const monitorTemplate = `
# Prometheus Monitor Service (Metrics)
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    control-plane: controller-manager
  name: controller-manager-metrics-monitor
  namespace: system
spec:
  endpoints:
    - path: /metrics
      port: https
  selector:
    control-plane: controller-manager
`
