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

package main

import (
	"log"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/version"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-tools/pkg/scaffold"
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/project"
)

func newVendorUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "updates vendor dependencies.",
		Long:  `updates vendor dependencies.`,
		Example: `Update the vendor dependencies:
kubebuilder update vendor
`,
		Run: func(cmd *cobra.Command, args []string) {
			dieIfNoProject()
			version := version.GetVersion()
			err := (&scaffold.Scaffold{}).Execute(input.Options{},
				&project.GopkgToml{KubebuilderVersion: version.KubeBuilderVersion})
			if err != nil {
				log.Fatalf("error updating vendor dependecies %v", err)
			}
		},
	}
}
