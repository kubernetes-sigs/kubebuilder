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
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "Histogram"
	Summary   MetricType = "Summary"
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
