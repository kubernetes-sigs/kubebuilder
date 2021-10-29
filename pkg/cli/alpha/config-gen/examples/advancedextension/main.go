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
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type API struct {
	Replicas int `json:"replicas,omitempty" yaml:"replicas,omitempty"`
}

// Advanced function for transforming kubebuilder output by looking at the resources and modifying them.
func main() {
	api := &API{}
	c := framework.SimpleProcessor{
		Config: api,
		Filter: kio.FilterFunc(func(r []*yaml.RNode) ([]*yaml.RNode, error) {
			matches, err := (&framework.Selector{
				Kinds: []string{"Deployment"},
				Names: []string{"controller-manager"},
			}).Filter(r)
			if err != nil {
				return nil, err
			}

			// set the replicas on all matching resources
			for i := range matches {
				matches[i].PipeE(
					// grab the spec.replicas field
					yaml.Lookup("spec", "replicas"),
					// set the value
					yaml.Set(yaml.NewScalarRNode(fmt.Sprintf("%d", api.Replicas))))
			}
			return matches, nil
		}),
	}

	cmd := command.Build(&c, command.StandaloneEnabled, false)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
