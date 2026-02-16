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

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

// metadataCmd handles the "metadata" subcommand.
//
// PURPOSE: Provide help text and examples for each subcommand.
// This information appears in `kubebuilder <subcommand> --help` output.
//
// FLOW:
// 1. User runs: kubebuilder init --help
// 2. Kubebuilder calls each plugin with: {"command":"metadata","args":["--init"]}
// 3. Plugin returns description and examples for init
// 4. Kubebuilder displays combined help from all plugins
func metadataCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "flags",
		Universe:   pr.Universe,
	}

	// Parse which subcommand's metadata is being requested
	flagsToParse := pflag.NewFlagSet("flagsFlags", pflag.ContinueOnError)
	flagsToParse.Bool("init", false, "sets the init flag to true")
	flagsToParse.Bool("edit", false, "sets the edit flag to true")

	if err := flagsToParse.Parse(pr.Args); err != nil {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			fmt.Sprintf("failed to parse flags: %s", err.Error()),
		}
		return pluginResponse
	}

	initFlag, _ := flagsToParse.GetBool("init")
	editFlag, _ := flagsToParse.GetBool("edit")

	// Return metadata for the requested subcommand
	if initFlag {
		pluginResponse.Metadata = scaffolds.InitMeta
	} else if editFlag {
		pluginResponse.Metadata = scaffolds.EditMeta
	} else {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			"no subcommand flag provided; expected --init or --edit",
		}
	}

	return pluginResponse
}
