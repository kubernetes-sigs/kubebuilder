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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"v1/scaffolds"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/external"
)

// Run will run the actual steps of the plugin
func Run() {
	// Phase 2 Plugins makes requests to an external plugin by
	// writing to the STDIN buffer. This means that an external plugin
	// call will NOT include any arguments other than the program name
	// itself. In order to get the request JSON from Kubebuilder
	// we will need to read the input from STDIN
	reader := bufio.NewReader(os.Stdin)

	input, err := io.ReadAll(reader)
	if err != nil {
		returnError(fmt.Errorf("encountered error reading STDIN: %w", err))
	}

	// Parse the JSON input from STDIN to a PluginRequest object.
	// Since the Phase 2 Plugin implementation was written in Go
	// there is already a Go API in place to represent these values.
	// Phase 2 Plugins can be written in any language, but you may
	// need to create some classes/interfaces to parse the JSON used
	// in the Phase 2 Plugins communication. More information on the
	// Phase 2 Plugin JSON schema can be found in the Phase 2 Plugins docs
	pluginRequest := &external.PluginRequest{}

	err = json.Unmarshal(input, pluginRequest)
	if err != nil {
		returnError(fmt.Errorf("encountered error unmarshaling STDIN: %w", err))
	}

	var response external.PluginResponse

	// Run logic depending on the command that is requested by Kubebuilder
	switch pluginRequest.Command {
	// the `init` subcommand is often used when initializing a new project
	case "init":
		response = scaffolds.InitCmd(pluginRequest)
	// the `create api` subcommand is often used after initializing a project
	// with the `init` subcommand to create a controller and CRDs for a
	// provided group, version, and kind
	case "create api":
		response = scaffolds.ApiCmd(pluginRequest)
	// the `create webhook` subcommand is often used after initializing a project
	// with the `init` subcommand to create a webhook for a provided
	// group, version, and kind
	case "create webhook":
		response = scaffolds.WebhookCmd(pluginRequest)
	// the `flags` subcommand is used to customize the flags that
	// the Kubebuilder cli will bind for use with this plugin
	case "flags":
		response = flagsCmd(pluginRequest)
	// the `metadata` subcommand is used to customize the
	// plugin metadata (help message and examples) that are
	// shown to Kubebuilder CLI users.
	case "metadata":
		response = metadataCmd(pluginRequest)
	// Any errors should still be returned as part of the plugin's
	// JSON response. There is an `error` boolean field to signal to
	// Kubebuilder that the external plugin encountered an error.
	// There is also an `errorMsgs` string array field to provide all
	// error messages to Kubebuilder.
	default:
		response = external.PluginResponse{
			Error: true,
			ErrorMsgs: []string{
				"unknown subcommand:" + pluginRequest.Command,
			},
		}
	}

	// The Phase 2 Plugins implementation will read the response
	// from a Phase 2 Plugin via STDOUT. For Kubebuilder to properly
	// read our response we need to create a valid JSON string and
	// write it to the STDOUT buffer.
	output, err := json.Marshal(response)
	if err != nil {
		returnError(fmt.Errorf("encountered error marshaling output: %w | OUTPUT: %s", err, output))
	}

	fmt.Printf("%s", output)
}
