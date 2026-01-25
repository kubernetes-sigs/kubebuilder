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

// Package main implements a sample external plugin for Kubebuilder.
// Adds Prometheus monitoring to Kubernetes operators.
//
// External plugins communicate via JSON over stdin/stdout:
//  1. Kubebuilder writes PluginRequest to stdin
//  2. Plugin reads request, executes logic
//  3. Plugin writes PluginResponse to stdout
//  4. Kubebuilder writes files from response.Universe to disk
//
// This plugin demonstrates:
//   - Using Kubebuilder's pkg/config API (don't reimplement PROJECT parsing)
//   - Using Kubebuilder's pkg/plugin/external types (PluginRequest/Response)
//   - Implementing only needed subcommands (this shows all for reference)
//   - Flag parsing using pflag (same library Kubebuilder uses)
//
// See https://book.kubebuilder.io/plugins/extending/external-plugins
package main

import (
	"v1/cmd"
)

func main() {
	cmd.Run()
}
