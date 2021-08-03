/*
Copyright 2021 The Kubernetes Authors.

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

package external

// PluginRequest contains all information kubebuilder received from the CLI
// and plugins executed before it.
type PluginRequest struct {
	// APIVersion defines the versioned schema of PluginRequest that is being sent from Kubebuilder.
	// Initially, this will be marked as alpha (v1alpha1).
	APIVersion string `json:"apiVersion"`

	// Args holds the plugin specific arguments that are received from the CLI
	// which are to be passed down to the external plugin.
	Args []string `json:"args"`

	// Command contains the command to be executed by the plugin such as init, create api, etc.
	Command string `json:"command"`

	// Universe represents the modified file contents that gets updated over a series of plugin runs
	// across the plugin chain. Initially, it starts out as empty.
	Universe map[string]string `json:"universe"`
}

// PluginResponse is returned to kubebuilder by the plugin and contains all files
// written by the plugin following a certain command.
type PluginResponse struct {
	// APIVersion defines the versioned schema of the PluginResponse that is back sent back to Kubebuilder.
	// Initially, this will be marked as alpha (v1alpha1)
	APIVersion string `json:"apiVersion"`

	// Command holds the command that gets executed by the plugin such as init, create api, etc.
	Command string `json:"command"`

	// Help contains the plugin specific help text that the plugin returns to Kubebuilder when it receives
	// `--help` flag from Kubebuilder.
	Help string `json:"help"`

	// Universe in the PluginResponse represents the updated file contents that was written by the plugin.
	Universe map[string]string `json:"universe"`

	// Error is a boolean type that indicates whether there were any errors due to plugin failures.
	Error bool `json:"error,omitempty"`

	// ErrorMsgs contains the specific error messages of the plugin failures.
	ErrorMsgs []string `json:"errorMsgs,omitempty"`
}
