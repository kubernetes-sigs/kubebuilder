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

package templates

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &CustomMetricsConfigManifest{}

// Kustomization scaffolds a file that defines the kustomization scheme for the prometheus folder
type CustomMetricsConfigManifest struct {
	machinery.TemplateMixin
	ConfigPath string
}

// SetTemplateDefaults implements file.Template
func (f *CustomMetricsConfigManifest) SetTemplateDefaults() error {
	f.Path = f.ConfigPath

	f.TemplateBody = customMetricsConfigTemplate

	f.IfExistsAction = machinery.SkipFile

	return nil
}

// nolint: lll
const customMetricsConfigTemplate = `---
customMetrics:
#  - metric: # Raw custom metric (required)
#    type:   # Metric type: counter/gauge/histogram (required)
#    expr:   # Prom_ql for the metric (optional)
#    unit:   # Unit of measurement, examples: s,none,bytes,percent,etc. (optional)
#
#
# Example:
# ---
# customMetrics:
#   - metric: foo_bar
#     unit: none
#     type: histogram
#   	expr: histogram_quantile(0.90, sum by(instance, le) (rate(foo_bar{job=\"$job\", namespace=\"$namespace\"}[5m])))
`
