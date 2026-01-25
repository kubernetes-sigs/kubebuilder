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

// EditFlags defines flags for the edit subcommand.
var EditFlags = []external.Flag{
	{
		Name:    "prometheus-namespace",
		Type:    "string",
		Default: "monitoring-system",
		Usage:   "Namespace where Prometheus instance will be deployed",
	},
}

// EditMeta provides help text for the edit subcommand.
var EditMeta = plugin.SubcommandMetadata{
	Description: "Add Prometheus instance to an existing project",
	Examples:    "kubebuilder edit --plugins sampleexternalplugin/v1 --prometheus-namespace monitoring",
}

// EditCmd handles the "edit" subcommand.
//
// EXTERNAL PLUGIN FLOW FOR EDIT:
// 1. User runs: kubebuilder edit --plugins sampleexternalplugin/v1
// 2. Kubebuilder reads existing PROJECT file into config map
// 3. Kubebuilder calls this external plugin via JSON over stdin
// 4. Plugin reads PluginRequest (with config from PROJECT), generates files
// 5. Kubebuilder writes files from response.Universe to disk
//
// NOTE: For edit, the PROJECT file MUST exist since we're modifying an existing project.
// The config map is populated from the PROJECT file, not from command-line flags.
func EditCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "edit",
		Universe:   pr.Universe,
	}

	// Parse plugin-specific flags
	// IMPORTANT: Use ParseErrorsWhitelist to ignore unknown flags from other plugins
	flagSet := pflag.NewFlagSet("edit", pflag.ContinueOnError)
	flagSet.ParseErrorsAllowlist.UnknownFlags = true // Ignore flags from other plugins
	prometheusNamespace := flagSet.String("prometheus-namespace", "monitoring-system", "")

	if err := flagSet.Parse(pr.Args); err != nil {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			fmt.Sprintf("failed to parse flags: %v", err),
		}
		return pluginResponse
	}

	// Validate flag values
	if err := validateNamespace(*prometheusNamespace); err != nil {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{err.Error()}
		return pluginResponse
	}

	// Read project name from PROJECT file (via config map)
	// For edit, this MUST exist since we're in an existing project
	var projectName string
	if pr.Config != nil {
		if name, ok := pr.Config["projectName"].(string); ok && name != "" {
			projectName = name
		}
	}

	// Fail if we can't read the project name - something is wrong with PROJECT file
	if projectName == "" {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			"failed to read project name from PROJECT file; ensure you're in a valid Kubebuilder project directory",
		}
		return pluginResponse
	}

	// Generate the same manifests as init (idempotent operation)
	prometheusInstance := prometheus.NewPrometheusInstance(
		prometheus.WithProjectName(projectName),
		prometheus.WithNamespace(*prometheusNamespace),
	)
	prometheusKustomization := prometheus.NewPrometheusKustomization(*prometheusNamespace)
	kustomizationPatch := prometheus.NewDefaultKustomizationPatch()

	// Return files via Universe map
	pluginResponse.Universe[prometheusInstance.Path] = prometheusInstance.Content
	pluginResponse.Universe[prometheusKustomization.Path] = prometheusKustomization.Content
	pluginResponse.Universe[kustomizationPatch.Path] = kustomizationPatch.Content

	return pluginResponse
}
