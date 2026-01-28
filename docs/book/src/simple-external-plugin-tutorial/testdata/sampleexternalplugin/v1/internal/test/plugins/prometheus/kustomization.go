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

// PrometheusKustomization represents the kustomization.yaml for Prometheus resources
type PrometheusKustomization struct {
	Path    string
	Content string
}

// NewPrometheusKustomization creates a new kustomization.yaml for Prometheus resources
func NewPrometheusKustomization(namespace string) *PrometheusKustomization {
	content := fmt.Sprintf(prometheusKustomizationTemplate, namespace)
	return &PrometheusKustomization{
		Path:    "config/prometheus/kustomization.yaml",
		Content: content,
	}
}

const prometheusKustomizationTemplate = `# Kustomization for Prometheus instance
# This kustomization includes the Prometheus instance that works with
# the ServiceMonitor already created by Kubebuilder in monitor.yaml
resources:
  - prometheus.yaml

namespace: %s
`

// DefaultKustomizationPatch represents instructions for adding Prometheus to the default kustomization.yaml
type DefaultKustomizationPatch struct {
	Path    string
	Content string
}

// NewDefaultKustomizationPatch creates instructions for adding Prometheus to config/default/kustomization.yaml
func NewDefaultKustomizationPatch() *DefaultKustomizationPatch {
	return &DefaultKustomizationPatch{
		Path:    "config/default/kustomization_prometheus_patch.yaml",
		Content: defaultKustomizationPatchTemplate,
	}
}

const defaultKustomizationPatchTemplate = `# Prometheus Setup Instructions
# ==============================
# 
# This plugin added a Prometheus instance to config/prometheus/prometheus.yaml
# 
# To enable Prometheus monitoring:
# 
# 1. Install Prometheus Operator:
#    kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml
# 
# 2. Uncomment the prometheus patch in config/default/kustomization.yaml
#    to include the Prometheus instance resources
# 
# 3. Deploy: make deploy
# 
# The Prometheus instance will automatically discover the ServiceMonitor
# that Kubebuilder scaffolded in config/prometheus/monitor.yaml
`
