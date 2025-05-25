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

	switch pr.Command {
	case "init":
		pluginResponse.Flags = scaffolds.InitFlags
	case "create api":
		pluginResponse.Flags = scaffolds.ApiFlags
	case "create webhook":
		pluginResponse.Flags = scaffolds.WebhookFlags
	default:
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			"unrecognized command: " + pr.Command,
		}
	}

	return pluginResponse
}
