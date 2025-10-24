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

// ServiceMonitor represents a Prometheus ServiceMonitor manifest
type ServiceMonitor struct {
	Path    string
	Content string
}

// ServiceMonitorOptions allows configuration of the ServiceMonitor
type ServiceMonitorOptions func(*ServiceMonitor)

// WithDomain sets the domain for the ServiceMonitor
func WithDomain(domain string) ServiceMonitorOptions {
	return func(sm *ServiceMonitor) {
		sm.Content = fmt.Sprintf(serviceMonitorTemplate, domain, domain)
	}
}

// WithProjectName sets the project name for the ServiceMonitor
func WithProjectName(projectName string) ServiceMonitorOptions {
	return func(sm *ServiceMonitor) {
		// Project name can be used for labels or naming
		// For now, we'll use it in a future iteration if needed
	}
}

// NewServiceMonitor creates a new ServiceMonitor manifest
func NewServiceMonitor(opts ...ServiceMonitorOptions) *ServiceMonitor {
	sm := &ServiceMonitor{
		Path: "config/prometheus/monitor.yaml",
	}

	for _, opt := range opts {
		opt(sm)
	}

	// Set default content if not set by options
	if sm.Content == "" {
		sm.Content = fmt.Sprintf(serviceMonitorTemplate, "example.com", "example.com")
	}

	return sm
}

const serviceMonitorTemplate = `# Prometheus Monitor Service (Metrics)
apiVersion: v1
kind: Service
metadata:
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: kustomize
  name: controller-manager-metrics-service
  namespace: system
spec:
  ports:
    - name: https
      port: 8443
      protocol: TCP
      targetPort: 8443
  selector:
    control-plane: controller-manager

---
# Prometheus ServiceMonitor
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: kustomize
  name: controller-manager-metrics-monitor
  namespace: system
spec:
  endpoints:
    - path: /metrics
      port: https
      scheme: https
      bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
      tlsConfig:
        insecureSkipVerify: true
  selector:
    matchLabels:
      control-plane: controller-manager
`
