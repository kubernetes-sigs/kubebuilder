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
package cmd

import (
	"v1/scaffolds"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

// flagsCmd handles all the logic for the `flags` subcommand of the sample external plugin.
// In Kubebuilder's Phase 2 Plugins the `flags` subcommand is an optional subcommand for
// external plugins to support. The `flags` subcommand allows for an external plugin
// to provide Kubebuilder with a list of flags that the `init`, `create api`, `create webhook`,
// and `edit` subcommands allow. This allows Kubebuilder to give an external plugin the ability
// to feel like a native Kubebuilder plugin to a Kubebuilder user by only binding the supported
// flags and failing early if an unknown flag is provided.
func flagsCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "flags",
		Universe:   pr.Universe,
		Flags:      []external.Flag{},
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
		// Add a flag to the JSON response `flags` field that Kubebuilder reads
		// to ensure it binds to the flags given in the response.
		pluginResponse.Flags = scaffolds.InitFlags
	} else if apiFlag {
		pluginResponse.Flags = scaffolds.ApiFlags
	} else if webhookFlag {
		pluginResponse.Flags = scaffolds.WebhookFlags
	} else {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			"unrecognized flag",
		}
	}

	return pluginResponse
}
