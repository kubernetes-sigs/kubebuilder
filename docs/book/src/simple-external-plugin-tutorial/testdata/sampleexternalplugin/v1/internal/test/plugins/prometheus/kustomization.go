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

// PrometheusKustomization represents the kustomization.yaml for Prometheus resources
type PrometheusKustomization struct {
	Path    string
	Content string
}

// NewPrometheusKustomization creates a new kustomization.yaml for Prometheus resources
func NewPrometheusKustomization() *PrometheusKustomization {
	return &PrometheusKustomization{
		Path:    "config/prometheus/kustomization.yaml",
		Content: prometheusKustomizationTemplate,
	}
}

const prometheusKustomizationTemplate = `resources:
  - prometheus.yaml
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

const defaultKustomizationPatchTemplate = `# [PROMETHEUS] To enable Prometheus monitoring, add the following to config/default/kustomization.yaml:
#
# In the resources section, add:
# - ../prometheus
#
# This will include the Prometheus instance in your deployment.
# Make sure you have the Prometheus Operator installed in your cluster.
#
# For more information, see: https://github.com/prometheus-operator/prometheus-operator
`
