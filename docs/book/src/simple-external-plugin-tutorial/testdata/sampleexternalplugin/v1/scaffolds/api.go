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
	"v1/scaffolds/internal/templates/api"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

var ApiFlags = []external.Flag{
	{
		Name:    "number",
		Default: "1",
		Type:    "int",
		Usage:   "set a number to be added to the scaffolded apiFile.txt",
	},
}

var ApiMeta = plugin.SubcommandMetadata{
	Description: "The `create api` subcommand of the sampleexternalplugin is meant to create an api for a project via Kubebuilder. It scaffolds a single file: `apiFile.txt`",
	Examples: `
	Scaffold with the defaults:
	$ kubebuilder create api --plugins sampleexternalplugin/v1

	Scaffold with a specific number in the apiFile.txt file:
	$ kubebuilder create api --plugins sampleexternalplugin/v1 --number 2
	`,
}

// ApiCmd handles all the logic for the `create api` subcommand of this sample external plugin
func ApiCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "create api",
		Universe:   pr.Universe,
	}

	// Here is an example of parsing a flag from a Kubebuilder external plugin request
	flags := pflag.NewFlagSet("apiFlags", pflag.ContinueOnError)
	flags.Int("number", 1, "set a number to be added in the scaffolded apiFile.txt")
	flags.Parse(pr.Args)
	number, _ := flags.GetInt("number")

	apiFile := api.NewApiFile(api.WithNumber(number))

	// Phase 2 Plugins uses the concept of a "universe" to represent the filesystem for a plugin.
	// This universe is a key:value mapping of filename:contents. Here we are adding the file
	// "apiFile.txt" to the universe with some content. When this is returned Kubebuilder will
	// take all values within the "universe" and write them to the user's filesystem.
	pluginResponse.Universe[apiFile.Name] = apiFile.Contents

	return pluginResponse
}
