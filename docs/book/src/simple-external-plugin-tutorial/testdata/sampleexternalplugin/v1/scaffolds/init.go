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
	"fmt"

	"github.com/spf13/pflag"

	"v1/internal/test/plugins/prometheus"

	_ "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

// InitFlags defines flags for the init subcommand.
var InitFlags = []external.Flag{
	{
		Name:    "prometheus-namespace",
		Type:    "string",
		Default: "monitoring-system",
		Usage:   "Namespace where Prometheus instance will be deployed",
	},
}

// InitMeta provides help text for the init subcommand.
var InitMeta = plugin.SubcommandMetadata{
	Description: "Scaffold Prometheus instance during project initialization",
	Examples:    "kubebuilder init --plugins go/v4,sampleexternalplugin/v1 --domain example.com --prometheus-namespace monitoring",
}

// InitCmd handles the optional "init" subcommand.
//
// EXTERNAL PLUGIN FLOW:
// 1. User runs: kubebuilder init --plugins go/v4,sampleexternalplugin/v1 --domain example.com
// 2. Kubebuilder processes go/v4 plugin first (creates PROJECT file, basic structure)
// 3. Kubebuilder calls this external plugin via JSON over stdin
// 4. Plugin reads PluginRequest, generates files, returns PluginResponse via stdout
// 5. Kubebuilder writes files from response.Universe to disk
//
// NOTE: During init, plugins run in chain order. The go/v4 plugin runs first and populates
// the config map with project metadata. This plugin runs after and can read that config.
func InitCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "init",
		Universe:   pr.Universe,
	}

	// Parse flags using pflag (same library Kubebuilder uses internally)
	// External plugins declare their flags via the "flags" subcommand, then parse them here
	// IMPORTANT: Use ParseErrorsWhitelist to ignore flags from other plugins in the chain
	// The pr.Args may contain flags for multiple plugins (go/v4, this plugin, etc.)
	flagSet := pflag.NewFlagSet("init", pflag.ContinueOnError)
	flagSet.ParseErrorsWhitelist.UnknownFlags = true // Ignore flags from other plugins
	prometheusNamespace := flagSet.String("prometheus-namespace", "monitoring-system", "")

	if err := flagSet.Parse(pr.Args); err != nil {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			fmt.Sprintf("failed to parse flags: %v", err),
		}
		return pluginResponse
	}

	// Validate flag values before using them
	// This demonstrates proper input validation in external plugins
	if err := validateNamespace(*prometheusNamespace); err != nil {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{err.Error()}
		return pluginResponse
	}

	// Read project name from config
	// During init, Kubebuilder passes the config map even before writing PROJECT file
	// The go/v4 plugin (which runs first in the chain) populates this config
	var projectName string
	if pr.Config != nil {
		if name, ok := pr.Config["projectName"].(string); ok && name != "" {
			projectName = name
		}
	}

	// If projectName is still empty, the go/v4 plugin didn't set it (shouldn't happen)
	if projectName == "" {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			"project name not found in config; ensure go/v4 plugin runs before this plugin in the chain",
		}
		return pluginResponse
	}

	// Generate manifests using template functions
	// External plugins don't use Kubebuilder's machinery.Template interface
	// Instead, they generate content as strings and return via Universe map
	prometheusInstance := prometheus.NewPrometheusInstance(
		prometheus.WithProjectName(projectName),
		prometheus.WithNamespace(*prometheusNamespace),
	)
	prometheusKustomization := prometheus.NewPrometheusKustomization(*prometheusNamespace)
	kustomizationPatch := prometheus.NewDefaultKustomizationPatch()

	// Populate the Universe map with file paths and contents
	// Kubebuilder reads this map and writes each file to disk
	// Key: relative file path from project root
	// Value: complete file content as string
	pluginResponse.Universe[prometheusInstance.Path] = prometheusInstance.Content
	pluginResponse.Universe[prometheusKustomization.Path] = prometheusKustomization.Content
	pluginResponse.Universe[kustomizationPatch.Path] = kustomizationPatch.Content

	return pluginResponse
}
