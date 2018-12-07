/*
Copyright 2018 The Kubernetes Authors.

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
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// QueueLength is a prometheus metric which counts the current reconcile
	// queue length per controller
	QueueLength = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "controller_runtime_reconcile_queue_length",
		Help: "Length of reconcile queue per controller",
	}, []string{"controller"})

	// ReconcileErrors is a prometheus counter metrics which holds the total
	// number of errors from the Reconciler
	ReconcileErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "controller_runtime_reconcile_errors_total",
		Help: "Total number of reconcile errors per controller",
	}, []string{"controller"})

	// ReconcileTime is a prometheus metric which keeps track of the duration
	// of reconciles
	ReconcileTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "controller_runtime_reconcile_time_seconds",
		Help: "Length of time per reconcile per controller",
	}, []string{"controller"})
)

func init() {
	metrics.Registry.MustRegister(
		QueueLength,
		ReconcileErrors,
		ReconcileTime,
		// expose process metrics like CPU, Memory, file descriptor usage etc.
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		// expose Go runtime metrics like GC stats, memory stats etc.
		prometheus.NewGoCollector(),
	)
}
