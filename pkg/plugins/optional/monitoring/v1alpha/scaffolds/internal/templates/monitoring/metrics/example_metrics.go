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

package metrics

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &ExampleMetricsManifest{}

// ExampleMetricsManifest scaffolds a file that defines the example prometheus metrics
type ExampleMetricsManifest struct {
	machinery.TemplateMixin
	machinery.RepositoryMixin
}

// SetTemplateDefaults implements file template
func (f *ExampleMetricsManifest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("monitoring", "metrics", "example_metrics.go")
	}

	f.TemplateBody = OperatorMetricsTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// nolint: lll
const OperatorMetricsTemplate = `
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

package metrics

import "{{ .Repo }}/monitoring/metrics/util"

var OperatorMetrics = []util.Metric{
// When adding new metrics, Please follow the naming conventions best practices
// TODO: Add the link to the Observability Best Practices - Metrics Naming
	{
		Name: "metric_name",
		Help: "metric description",
		// A Counter is typically used to count requests served, tasks completed, errors occurred, etc.
		Type: util.Counter,
	},
}

// For information about Prometheus metric types and help functions see:
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#pkg-types


// The util.go includes the Prometheus API main functions for setting the metrics value.
// Add here functions to update the metrics values in this file.
// IMPORTANT:
// In your core operator code only call these functions, with the required parameters.
// All the logic should be handheld here.
func IncrementMetric() {
	util.GetCounterMetric("metric_name").Inc()
}
`
