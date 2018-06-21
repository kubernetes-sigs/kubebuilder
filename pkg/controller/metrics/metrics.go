/*
Copyright 2017 The Kubernetes Authors.

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
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ReconcileQueueLength = "QueueLength"
	ReconcileError       = "Error"
	ReconcileTime        = "Time"
	ControllerName       = "Name"
)

var (
	reconcileQueueLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "generic_controller_reconcile_listening_queue_length",
			Help: "Length of listeningQueue for reconcile",
		},
		[]string{"controller"},
	)

	reconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "generic_controller_reconcile_errors_total",
			Help: "Number of errors for reconcile",
		},
		[]string{"controller"},
	)

	reconcileTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "generic_controller_reconcile_time_second",
			Help: "Reconcile time for each running concile loop",
		},
		[]string{"controller"},
	)

	metrics = []metricsCollector{
		reconcileQueueLength,
		reconcileErrors,
		reconcileTime,
	}

	registerMetrics sync.Once
)

type metricsCollector interface {
	prometheus.Collector
	Reset()
}

// Metrics contains runtime metrics about the controller
type Metrics struct {
	// UncompletedReconcileTs is a sorted slice of start timestamps from the currently running reconcile loops
	// This can be used to calculate stats such as - shortest running reconcile, longest running reconcile, mean time
	UncompletedReconcileTs []int64

	// QueueLength is the number of unprocessed messages in the queue
	QueueLength int

	// MeanCompletionTime gives the average reconcile time over the past 10m
	// TODO: Implement this
	MeanReconcileTime int

	// ReconcileRate gives the average reconcile rate over the past 10m
	// TODO: Implement this
	ReconcileRate int

	// QueueLength is the average queue length over the past 10m
	// TODO: Implement this
	MeanQueueLength int
}

func Register() {
	registerMetrics.Do(func() {
		for _, m := range metrics {
			prometheus.MustRegister(m)
		}
	})
}

// Reset all metrics
func Reset() {
	for _, m := range metrics {
		m.Reset()
	}
}

func RecordMetrics(name string, m *Metrics) *Metrics {
	if m != nil {
		t := time.Now()
		observeReconcileTime(name, t, m)
		observeQueueLength(name, m)
		m.MeanReconcileTime = int(calcualteMeanReconcileTime(t, m))
	}
	return m
}

func observeReconcileTime(name string, t time.Time, m *Metrics) {
	for _, ts := range m.UncompletedReconcileTs {
		reconcileTime.WithLabelValues(name).Observe(float64(t.Unix() - ts))
	}
}

func observeQueueLength(name string, m *Metrics) {
	reconcileQueueLength.WithLabelValues(name).Set(float64(m.QueueLength))
}

func ObserveReconcileErrors(name string, err error) {
	if err != nil {
		reconcileErrors.WithLabelValues(name).Inc()
	}
}

func calcualteMeanReconcileTime(t time.Time, m *Metrics) float64 {
	var total, count float64
	for _, ts := range m.UncompletedReconcileTs {
		if ts > t.Unix()-600 {
			total = total + float64(t.Unix()-ts)
			count++
		}
	}
	return total / count
}

func getReconcileTime(t time.Time, ts []int64) float64 {
	if len(ts) == 0 {
		return 0
	}
	if len(ts) == 1 {
		return float64(t.Unix() - ts[0])
	}
	return float64(ts[len(ts)-1] - ts[0])
}
