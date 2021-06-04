/*
Copyright 2021 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"os"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/parser"
)

type API struct {
	Replicas int `json:"replicas,omitempty" yaml:"replicas,omitempty"`
}

// Simple function for transforming kubebuilder output by patching the replicas field.
// This is a very basic example that applies patches statically based on the input
func main() {
	c := framework.TemplateProcessor{
		TemplateData: &API{},
		PatchTemplates: []framework.PatchTemplate{&framework.ResourcePatchTemplate{
			Selector: &framework.Selector{
				Kinds: []string{"Deployment"},
				Names: []string{"controller-manager"},
			},
			Templates: parser.TemplateStrings(`
spec:
  replicas: {{.Replicas}}
`),
		}},
	}

	cmd := command.Build(&c, command.StandaloneEnabled, false)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
