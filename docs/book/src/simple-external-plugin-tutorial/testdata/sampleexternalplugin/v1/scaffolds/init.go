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
package scaffolds

import (
	"v1/scaffolds/internal/templates"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

var InitFlags = []external.Flag{
	{
		Name:    "domain",
		Type:    "string",
		Default: "example.domain.com",
		Usage:   "sets the domain added in the scaffolded initFile.txt",
	},
}

var InitMeta = plugin.SubcommandMetadata{
	Description: "The `init` subcommand of the sampleexternalplugin is meant to initialize a project via Kubebuilder. It scaffolds a single file: `initFile.txt`",
	Examples: `
	Scaffold with the defaults:
	$ kubebuilder init --plugins sampleexternalplugin/v1

	Scaffold with a specific domain:
	$ kubebuilder init --plugins sampleexternalplugin/v1 --domain sample.domain.com
	`,
}

// InitCmd handles all the logic for the `init` subcommand of this sample external plugin
func InitCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "init",
		Universe:   pr.Universe,
	}

	// Here is an example of parsing a flag from a Kubebuilder external plugin request
	flags := pflag.NewFlagSet("initFlags", pflag.ContinueOnError)
	flags.String("domain", "example.domain.com", "sets the domain added in the scaffolded initFile.txt")
	flags.Parse(pr.Args)
	domain, _ := flags.GetString("domain")

	initFile := templates.NewInitFile(templates.WithDomain(domain))

	// Phase 2 Plugins uses the concept of a "universe" to represent the filesystem for a plugin.
	// This universe is a key:value mapping of filename:contents. Here we are adding the file
	// "initFile.txt" to the universe with some content. When this is returned Kubebuilder will
	// take all values within the "universe" and write them to the user's filesystem.
	pluginResponse.Universe[initFile.Name] = initFile.Contents

	return pluginResponse
}
