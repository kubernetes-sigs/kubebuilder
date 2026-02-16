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
	namespace   string
}

// PrometheusOptions allows configuration of the Prometheus instance
type PrometheusOptions func(*PrometheusInstance)

// WithProjectName sets the project name for the Prometheus instance
func WithProjectName(projectName string) PrometheusOptions {
	return func(p *PrometheusInstance) {
		p.projectName = projectName
	}
}

// WithNamespace sets the namespace for the Prometheus instance
func WithNamespace(namespace string) PrometheusOptions {
	return func(p *PrometheusInstance) {
		p.namespace = namespace
	}
}

// NewPrometheusInstance creates a new Prometheus instance manifest
func NewPrometheusInstance(opts ...PrometheusOptions) *PrometheusInstance {
	p := &PrometheusInstance{
		Path:        "config/prometheus/prometheus.yaml",
		projectName: "project",
		namespace:   "monitoring-system",
	}

	for _, opt := range opts {
		opt(p)
	}

	// Generate content with project name and namespace
	p.Content = fmt.Sprintf(prometheusTemplate, p.projectName, p.projectName, p.namespace, p.projectName)

	return p
}

const prometheusTemplate = `# Prometheus Instance for %s
# This resource defines a Prometheus instance that scrapes metrics from your operator.
# Requires Prometheus Operator: https://github.com/prometheus-operator/prometheus-operator
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: %s-prometheus
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/component: prometheus
    app.kubernetes.io/managed-by: kustomize
spec:
  replicas: 1
  scrapeInterval: 30s
  retention: 24h
  
  # Service account for Prometheus to use when scraping
  serviceAccountName: prometheus
  
  # Selector for ServiceMonitors to scrape
  # Matches the ServiceMonitor created by Kubebuilder in config/prometheus/monitor.yaml
  serviceMonitorSelector:
    matchLabels:
      control-plane: controller-manager
  
  # Security best practices
  securityContext:
    runAsNonRoot: true
    runAsUser: 65534
    fsGroup: 65534
    seccompProfile:
      type: RuntimeDefault
  
  # Resource limits
  resources:
    requests:
      memory: "400Mi"
      cpu: "100m"
    limits:
      memory: "800Mi"
      cpu: "200m"
`
