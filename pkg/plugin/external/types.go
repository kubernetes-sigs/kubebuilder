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

package external

import "sigs.k8s.io/kubebuilder/v3/pkg/plugin"

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

	// Metadata contains the plugin specific help text that the plugin returns to Kubebuilder when it receives
	// `--help` flag from Kubebuilder.
	Metadata plugin.SubcommandMetadata `json:"metadata"`

	// Universe in the PluginResponse represents the updated file contents that was written by the plugin.
	Universe map[string]string `json:"universe"`

	// Error is a boolean type that indicates whether there were any errors due to plugin failures.
	Error bool `json:"error,omitempty"`

	// ErrorMsgs contains the specific error messages of the plugin failures.
	ErrorMsgs []string `json:"errorMsgs,omitempty"`

	// Flags contains the plugin specific flags that the plugin returns to Kubebuilder when it receives
	// a request for a list of supported flags from Kubebuilder
	Flags []Flag `json:"flags,omitempty"`
}

// Flag is meant to represent a CLI flag that is used by Kubebuilder to define flags that are parsed
// for use with an external plugin
type Flag struct {
	// Name is the name that should be used when creating the flag.
	// i.e a name of "domain" would become the CLI flag "--domain"
	Name string

	// Type is the type of flag that should be created. The types that
	// Kubebuilder supports are: string, bool, int, and float.
	// any value other than the supported will be defaulted to be a string
	Type string

	// Default is the default value that should be used for a flag.
	// Kubebuilder will attempt to convert this value to the defined
	// type for this flag.
	Default string

	// Usage is a description of the flag and when/why/what it is used for.
	Usage string
}
