/*
Copyright 2020 The Kubernetes Authors.

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

// Package testdata contains input and expected output for running config-gen.
//
// To add a new test create a new directory with the test name, a config.yaml with
// the input, and an expected.yaml with the expected output.
//
// The project directory contains a sample project used as input.  New sample projects
// may be added as new directories and referenced from the config.yaml.
//
// To update the testdata automatically modify ../configgen_test.go by uncommenting
// the corresponding line.
package testdata

import (
	_ "sigs.k8s.io/controller-runtime"
	_ "sigs.k8s.io/controller-runtime/pkg/scheme"
)
