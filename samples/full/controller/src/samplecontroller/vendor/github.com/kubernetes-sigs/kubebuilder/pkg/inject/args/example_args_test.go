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

package args_test

import (
	"flag"
	"github.com/kubernetes-sigs/kubebuilder/pkg/config"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/args"
)

func Example() {
	flag.Parse()
	config := config.GetConfigOrDie()

	// Create base arguments for initializing controllers
	var _ = args.CreateInjectArgs(config)
}

func ExampleCreateInjectArgs() {
	flag.Parse()
	config := config.GetConfigOrDie()

	// Create base arguments for initializing controllers
	var _ = args.CreateInjectArgs(config)
}

func ExampleInjectArgs_CreateRecorder() {
	flag.Parse()
	config := config.GetConfigOrDie()

	iargs := args.CreateInjectArgs(config)
	var _ = iargs.CreateRecorder("ControllerName")
}
