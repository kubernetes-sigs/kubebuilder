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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &ExampleMetricsManifest{}

// Kustomization scaffolds a file that defines the kustomization scheme for the metrics folder
type ExampleMetricsManifest struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements file.Template
func (f *RuntimeManifest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("monitoring", "metrics", "example_metrics.go")
	}

	f.TemplateBody = ExampleMetricsTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// nolint: lll
const ExampleMetricsTemplate = `
package metrics

import "github.com/<example>/<example-operator>/monitoring/metrics/util"

var ExampleMetrics = []util.Metric{
	{
		Name: "example_count_total",
		Help: "write here the metric description",
		// A Counter is typically used to count requests served, tasks completed, errors occurred, etc.
		Type: util.Counter,
	},
	{
		Name: "example_status",
		Help: "write here the metric description",
		// A Gauge is typically used for measured values, but also "counts" that can go up and down.
		Type: util.Gauge,
	},
}

// IncrementNumber and SetStatus should update both the metric and the
// deployment resource. for more information about metric types: 
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#pkg-types

// Inc increments the counter by 1.
// You will need to call this function in your operator controller.
func IncrementMetric() {
	util.GetCounterMetric("example_count_total").Inc()
}

// Set sets the Gauge to an arbitrary value.
func SetStatus(status float64) {
	util.GetGaugeMetric("example_status").Set(status)
}
`
