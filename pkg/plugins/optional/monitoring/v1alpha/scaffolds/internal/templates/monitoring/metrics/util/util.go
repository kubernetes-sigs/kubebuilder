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

package util

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &MetricsUtilManifest{}

// MetricsUtilManifest scaffolds a file that defines the prometheus metrics utils
type MetricsUtilManifest struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements file template
func (f *MetricsUtilManifest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("monitoring", "metrics", "util", "util.go")
	}

	f.TemplateBody = MetricsUtilTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// nolint: lll
const MetricsUtilTemplate = `
package util

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type MetricType string
// For more available Prometheus metric types
// see: https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#pkg-types
const (
	Counter     MetricType = "counter"
	Gauge       MetricType = "gauge"
	Histogram   MetricType = "Histogram"
	Summary     MetricType = "Summary"
)

type Metric struct {
	Name string
	Help string
	Type MetricType
}

var collectors map[string]prometheus.Collector

func RegisterMetrics(allMetrics [][]Metric) {
	collectors = map[string]prometheus.Collector{}

	for _, m := range flatMetrics(allMetrics) {
		v := createMetric(m)
		metrics.Registry.MustRegister(v)
		collectors[m.Name] = v
	}
}

// TODO: add here a comment about the functions to explain what they do
func ListMetrics(allMetrics [][]Metric) []Metric {
	return flatMetrics(allMetrics)
}

func GetCounterMetric(metricName string) prometheus.Counter {
	return castCounter(metricName, getMetric(metricName))
}

func GetGaugeMetric(metricName string) prometheus.Gauge {
	return castGauge(metricName, getMetric(metricName))
}

func GetHistogramMetric(metricName string) prometheus.Histogram {
	return castHistogram(metricName, getMetric(metricName))
}

func GetSummaryMetric(metricName string) prometheus.Summary {
	return castSummary(metricName, getMetric(metricName))
}

func flatMetrics(allMetrics [][]Metric) []Metric {
	var result []Metric
	for _, metricFile := range allMetrics {
		result = append(result, metricFile...)
	}
	return result
}

func createMetric(metric Metric) prometheus.Collector {
	opts := prometheus.Opts{
		Name: metric.Name,
		Help: metric.Help,
	}

	histogramOpts := prometheus.HistogramOpts{
		Name: metric.Name,
		Help: metric.Help,
	}

	summaryOpts := prometheus.SummaryOpts{
		Name: metric.Name,
		Help: metric.Help,
	}

	// To create Type instances, use New<Type>.
	switch metric.Type {
	case Counter:
		return prometheus.NewCounter(prometheus.CounterOpts(opts))
	case Gauge:
		return prometheus.NewGauge(prometheus.GaugeOpts(opts))
	case Histogram:
		return prometheus.NewHistogram(histogramOpts)
	case Summary:
		return prometheus.NewSummary(summaryOpts)
	}

	panic(unknownMetricTypeError(metric.Name, string(metric.Type)))
}

func getMetric(metricName string) prometheus.Collector {
	metric, found := collectors[metricName]
	if !found {
		panic(unknownMetricNameError(metricName))
	}
	return metric
}

func castCounter(metricName string, metric prometheus.Collector) prometheus.Counter {
	v, ok := metric.(prometheus.Counter)
	if !ok {
		panic(unknownMetricTypeError(metricName, "Counter"))
	}
	return v
}

func castGauge(metricName string, metric prometheus.Collector) prometheus.Gauge {
	v, ok := metric.(prometheus.Gauge)
	if !ok {
		panic(unknownMetricTypeError(metricName, "Gauge"))
	}
	return v
}

func castHistogram(metricName string, metric prometheus.Collector) prometheus.Histogram {
	v, ok := metric.(prometheus.Histogram)
	if !ok {
		panic(unknownMetricTypeError(metricName, "Histogram"))
	}
	return v
}

func castSummary(metricName string, metric prometheus.Collector) prometheus.Summary {
	v, ok := metric.(prometheus.Summary)
	if !ok {
		panic(unknownMetricTypeError(metricName, "Summary"))
	}
	return v
}

func unknownMetricNameError(metricName string) error {
	return fmt.Errorf("unknown metric name %s", metricName)
}

func unknownMetricTypeError(metricName string, requestedType string) error {
	return fmt.Errorf("%s is not requested %s metric type", metricName, requestedType)
}
`
