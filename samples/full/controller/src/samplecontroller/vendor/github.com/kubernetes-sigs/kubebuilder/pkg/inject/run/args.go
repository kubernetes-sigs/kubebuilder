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

package run

// RunArguments configures options for running controllers
type RunArguments struct {
	// ControllerParallelism is the number of concurrent ReconcileFn routines to run for each GenericController
	ControllerParallelism int

	// Stop will shutdown the GenericController when it is closed
	Stop <-chan struct{}
}

// CreateRunArguments returns new run arguments for controllers and admission hooks
func CreateRunArguments() RunArguments {
	return RunArguments{
		Stop: make(chan struct{}),
		ControllerParallelism: 1,
	}
}
