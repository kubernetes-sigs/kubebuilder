/*
Copyright 2022 The Kubernetes Authors.

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

import "fmt"

// PrometheusInstance represents a Prometheus instance manifest
type PrometheusInstance struct {
	Path        string
	Content     string
	projectName string
}

// PrometheusOptions allows configuration of the Prometheus instance
type PrometheusOptions func(*PrometheusInstance)

// WithProjectName sets the project name for the Prometheus instance
func WithProjectName(projectName string) PrometheusOptions {
	return func(p *PrometheusInstance) {
		p.projectName = projectName
	}
}

// NewPrometheusInstance creates a new Prometheus instance manifest
func NewPrometheusInstance(opts ...PrometheusOptions) *PrometheusInstance {
	p := &PrometheusInstance{
		Path:        "config/prometheus/prometheus.yaml",
		projectName: "project",
	}

	for _, opt := range opts {
		opt(p)
	}

	// Generate content with project name for labels
	p.Content = fmt.Sprintf(prometheusTemplate, p.projectName)

	return p
}

const prometheusTemplate = `# Prometheus Instance
# This creates a Prometheus instance that will scrape metrics from your operator.
# Requires the Prometheus Operator to be installed in your cluster.
# See: https://github.com/prometheus-operator/prometheus-operator
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: kustomize
spec:
  replicas: 1
  scrapeInterval: 30s
  serviceAccountName: controller-manager
  serviceMonitorSelector:
    matchLabels:
      control-plane: controller-manager
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
`
