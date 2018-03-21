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
