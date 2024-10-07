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
	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

var WebhookFlags = []external.Flag{
	{
		Name:    "hooked",
		Type:    "bool",
		Default: "false",
		Usage:   "add the word `hooked` to the end of the scaffolded webhookFile.txt",
	},
}

var WebhookMeta = plugin.SubcommandMetadata{
	Description: "The `create webhook` subcommand of the sampleexternalplugin is meant to create a webhook for a project via Kubebuilder. It scaffolds a single file: `webhookFile.txt`",
	Examples: `
	Scaffold with the defaults:
	$ kubebuilder create webhook --plugins sampleexternalplugin/v1

	Scaffold with the text "HOOKED!" in the webhookFile.txt file:
	$ kubebuilder create webhook --plugins sampleexternalplugin/v1 --hooked
	`,
}

// WebhookCmd handles all the logic for the `create webhook` subcommand of this sample external plugin
func WebhookCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "create webhook",
		Universe:   pr.Universe,
	}

	// Here is an example of parsing a flag from a Kubebuilder external plugin request
	flags := pflag.NewFlagSet("apiFlags", pflag.ContinueOnError)
	flags.Bool("hooked", false, "add the word `hooked` to the end of the scaffolded webhookFile.txt")
	flags.Parse(pr.Args)
	hooked, _ := flags.GetBool("hooked")

	msg := "A simple text file created with the `create webhook` subcommand"
	if hooked {
		msg += "\nHOOKED!"
	}

	// Phase 2 Plugins uses the concept of a "universe" to represent the filesystem for a plugin.
	// This universe is a key:value mapping of filename:contents. Here we are adding the file
	// "webhookFile.txt" to the universe with some content. When this is returned Kubebuilder will
	// take all values within the "universe" and write them to the user's filesystem.
	pluginResponse.Universe["webhookFile.txt"] = msg

	return pluginResponse
}
