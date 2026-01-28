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

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

// Run executes the external plugin logic.
//
// EXTERNAL PLUGIN ARCHITECTURE:
// External plugins are standalone executables that communicate with Kubebuilder via JSON over stdin/stdout.
// This allows plugins to be written in any language, though this sample uses Go for consistency.
//
// PROTOCOL:
// 1. INPUT:  Kubebuilder writes PluginRequest (JSON) to plugin's stdin
// 2. PROCESS: Plugin executes the requested subcommand
// 3. OUTPUT: Plugin writes PluginResponse (JSON) to stdout
//
// IMPORTANT: stdout is RESERVED for JSON responses. Debug output must go to stderr or log files.
func Run() {
	// Read PluginRequest JSON from stdin
	// Kubebuilder writes a single JSON object containing:
	// - command: which subcommand to execute (init, edit, flags, metadata)
	// - args: command-line arguments for the plugin
	// - config: PROJECT file contents as map (if file exists)
	// - universe: existing files that may need modification
	reader := bufio.NewReader(os.Stdin)
	input, err := io.ReadAll(reader)
	if err != nil {
		returnError(fmt.Errorf("encountered error reading STDIN: %w", err))
	}

	// Parse request using Kubebuilder's external.PluginRequest type
	// This type is defined in pkg/plugin/external and ensures compatibility
	pluginRequest := &external.PluginRequest{}

	err = json.Unmarshal(input, pluginRequest)
	if err != nil {
		returnError(fmt.Errorf("encountered error unmarshaling STDIN: %w", err))
	}

	var response external.PluginResponse

	// Route to appropriate handler based on subcommand
	// External plugins can implement any subset of these subcommands
	switch pluginRequest.Command {
	case "init":
		// Called during: kubebuilder init --plugins sampleexternalplugin/v1
		// Purpose: Add files/features during project initialization
		// Runs AFTER base plugin (go/v4) sets up project structure
		response = scaffolds.InitCmd(pluginRequest)

	case "edit":
		// Called during: kubebuilder edit --plugins sampleexternalplugin/v1
		// Purpose: Add files/features to existing project
		// Most common use case for external plugins (optional enhancements)
		response = scaffolds.EditCmd(pluginRequest)

	case "flags":
		// Called by: kubebuilder internally before running subcommands
		// Purpose: Inform Kubebuilder which flags this plugin accepts
		// Enables early validation and better --help output
		response = flagsCmd(pluginRequest)

	case "metadata":
		// Called by: kubebuilder internally for --help output
		// Purpose: Provide description and examples for each subcommand
		// Makes external plugins feel native to Kubebuilder
		response = metadataCmd(pluginRequest)

	default:
		// Unknown subcommand - return error response
		// Errors must be returned as JSON, not exit codes
		response = external.PluginResponse{
			Error: true,
			ErrorMsgs: []string{
				"unknown subcommand: " + pluginRequest.Command,
			},
		}
	}

	// Write PluginResponse JSON to stdout
	// IMPORTANT: This must be the only stdout output (no debug prints!)
	output, err := json.Marshal(response)
	if err != nil {
		returnError(fmt.Errorf("encountered error marshaling output: %w | OUTPUT: %s", err, output))
	}

	fmt.Printf("%s", output)
}
