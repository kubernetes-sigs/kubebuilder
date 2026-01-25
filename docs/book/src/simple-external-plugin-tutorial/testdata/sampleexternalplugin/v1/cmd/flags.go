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
	"fmt"
	"v1/scaffolds"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

// flagsCmd handles the "flags" subcommand.
//
// PURPOSE: Inform Kubebuilder which flags this plugin accepts for each subcommand.
// This enables Kubebuilder to:
// 1. Validate flags early (before calling the plugin)
// 2. Show plugin flags in --help output
// 3. Prevent flag conflicts between plugins in a chain
//
// FLOW:
// 1. Kubebuilder calls: echo '{"command":"flags","args":["--init"]}' | plugin
// 2. Plugin returns: {"flags":[{name:"prometheus-namespace", type:"string", ...}]}
// 3. Kubebuilder binds those flags for the init command
func flagsCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "flags",
		Universe:   pr.Universe,
		Flags:      []external.Flag{},
	}

	// Determine which subcommand's flags are being requested
	var subcommand string
	for _, arg := range pr.Args {
		if arg == "--init" {
			subcommand = "init"
			break
		} else if arg == "--edit" {
			subcommand = "edit"
			break
		}
	}

	switch subcommand {
	case "init":
		pluginResponse.Flags = scaffolds.InitFlags
	case "edit":
		pluginResponse.Flags = scaffolds.EditFlags
	default:
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			fmt.Sprintf("unrecognized subcommand flag in args (received %d args)", len(pr.Args)),
		}
	}

	return pluginResponse
}
