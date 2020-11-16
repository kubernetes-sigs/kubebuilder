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

	"sigs.k8s.io/kubebuilder/v2/pkg/cli"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	pluginv2 "sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v2"
	pluginv3 "sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3"
)

func main() {
	c, err := cli.New(
		cli.WithCommandName("kubebuilder"),
		cli.WithVersion(versionString()),
		cli.WithDefaultProjectVersion(config.Version3Alpha),
		cli.WithPlugins(
			&pluginv2.Plugin{},
			&pluginv3.Plugin{},
		),
		cli.WithDefaultPlugins(config.Version2,
			&pluginv2.Plugin{},
		),
		cli.WithDefaultPlugins(config.Version3Alpha,
			&pluginv2.Plugin{},
		),
		cli.WithCompletion,
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}
