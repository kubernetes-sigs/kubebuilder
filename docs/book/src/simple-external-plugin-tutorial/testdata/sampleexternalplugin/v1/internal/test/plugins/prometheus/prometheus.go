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
	Path      string
	Content   string
	domain    string
	namespace string
}

// PrometheusOptions allows configuration of the Prometheus instance
type PrometheusOptions func(*PrometheusInstance)

// WithDomain sets the domain for the Prometheus instance
func WithDomain(domain string) PrometheusOptions {
	return func(p *PrometheusInstance) {
		p.domain = domain
	}
}

// WithProjectName sets the project name for the Prometheus instance
func WithProjectName(projectName string) PrometheusOptions {
	return func(p *PrometheusInstance) {
		// Project name can be used for labels or naming
		// For now, we'll use it in a future iteration if needed
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
		Path:      "config/prometheus/prometheus.yaml",
		domain:    "example.com",
		namespace: "system",
	}

	for _, opt := range opts {
		opt(p)
	}

	// Generate content with domain and namespace
	p.Content = fmt.Sprintf(prometheusTemplate,
		p.namespace, p.domain, // ServiceAccount
		p.domain,              // ClusterRole
		p.domain,              // ClusterRoleBinding
		p.namespace,           // ClusterRoleBinding subject
		p.namespace, p.domain, // Prometheus
	)

	return p
}

const prometheusTemplate = `# Prometheus ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: kustomize

---
# Prometheus ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: kustomize
rules:
- apiGroups: [""]
  resources: ["nodes", "nodes/metrics", "services", "endpoints", "pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get"]
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]

---
# Prometheus ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: kustomize
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus
subjects:
- kind: ServiceAccount
  name: prometheus
  namespace: %s

---
# Prometheus Instance
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: kustomize
spec:
  logLevel: debug
  ruleSelector: {}
  scrapeInterval: 1m
  scrapeTimeout: 30s
  securityContext:
    runAsNonRoot: true
    runAsUser: 65534
    seccompProfile:
      type: RuntimeDefault
  serviceAccountName: prometheus
  serviceDiscoveryRole: EndpointSlice
  serviceMonitorSelector: {}
`
