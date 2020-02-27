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

	"sigs.k8s.io/kubebuilder/cmd/version"
	"sigs.k8s.io/kubebuilder/pkg/cli"
	pluginv1 "sigs.k8s.io/kubebuilder/pkg/plugin/v1"
	pluginv2 "sigs.k8s.io/kubebuilder/pkg/plugin/v2"
)

func main() {
	c, err := cli.New(
		cli.WithPlugins(
			&pluginv1.Plugin{},
			&pluginv2.Plugin{},
		),
		cli.WithDefaultPlugins(
			&pluginv1.Plugin{},
			&pluginv2.Plugin{},
		),
		cli.WithExtraCommands(
			newEditCmd(),
			newUpdateCmd(),
			version.NewCmd(),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}
