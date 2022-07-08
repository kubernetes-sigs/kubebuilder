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
package main

import (
	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/external"
)

// metadataCmd handles all the logic for the `metadata` subcommand of the sample external plugin.
// In Kubebuilder's Phase 2 Plugins the `metadata` subcommand is an optional subcommand for
// external plugins to support. The `metadata` subcommand allows for an external plugin
// to provide Kubebuilder with a description of the plugin and examples for each of the
// `init`, `create api`, `create webhook`, and `edit` subcommands. This allows Kubebuilder
// to provide users a native Kubebuilder plugin look and feel for an external plugin.
func metadataCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "flags",
		Universe:   pr.Universe,
	}

	// Here is an example of parsing multiple flags from a Kubebuilder external plugin request
	flagsToParse := pflag.NewFlagSet("flagsFlags", pflag.ContinueOnError)
	flagsToParse.Bool("init", false, "sets the init flag to true")
	flagsToParse.Bool("api", false, "sets the api flag to true")
	flagsToParse.Bool("webhook", false, "sets the webhook flag to true")

	flagsToParse.Parse(pr.Args)

	initFlag, _ := flagsToParse.GetBool("init")
	apiFlag, _ := flagsToParse.GetBool("api")
	webhookFlag, _ := flagsToParse.GetBool("webhook")

	// The Phase 2 Plugins implementation will only ever pass a single boolean flag
	// argument in the JSON request `args` field. The flag will be `--init` if it is
	// attempting to get the flags for the `init` subcommand, `--api` for `create api`,
	// `--webhook` for `create webhook`, and `--edit` for `edit`
	if initFlag {
		// Populate the JSON response `metadata` field with a description
		// and examples for the `init` subcommand
		pluginResponse.Metadata = plugin.SubcommandMetadata{
			Description: "The `init` subcommand of the sampleexternalplugin is meant to initialize a project via Kubebuilder. It scaffolds a single file: `initFile.txt`",
			Examples: `
			Scaffold with the defaults:
			$ kubebuilder init --plugins sampleexternalplugin/v1

			Scaffold with a specific domain:
			$ kubebuilder init --plugins sampleexternalplugin/v1 --domain sample.domain.com
			`,
		}
	} else if apiFlag {
		pluginResponse.Metadata = plugin.SubcommandMetadata{
			Description: "The `create api` subcommand of the sampleexternalplugin is meant to create an api for a project via Kubebuilder. It scaffolds a single file: `apiFile.txt`",
			Examples: `
			Scaffold with the defaults:
			$ kubebuilder create api --plugins sampleexternalplugin/v1

			Scaffold with a specific number in the apiFile.txt file:
			$ kubebuilder create api --plugins sampleexternalplugin/v1 --number 2
			`,
		}
	} else if webhookFlag {
		pluginResponse.Metadata = plugin.SubcommandMetadata{
			Description: "The `create webhook` subcommand of the sampleexternalplugin is meant to create a webhook for a project via Kubebuilder. It scaffolds a single file: `webhookFile.txt`",
			Examples: `
			Scaffold with the defaults:
			$ kubebuilder create webhook --plugins sampleexternalplugin/v1

			Scaffold with the text "HOOKED!" in the webhookFile.txt file:
			$ kubebuilder create webhook --plugins sampleexternalplugin/v1 --hooked
			`,
		}
	} else {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			"unrecognized flag",
		}
	}

	return pluginResponse
}
