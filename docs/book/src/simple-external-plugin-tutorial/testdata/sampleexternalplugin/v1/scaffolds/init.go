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
	"v1/internal/test/plugins/prometheus"

	_ "sigs.k8s.io/kubebuilder/v4/pkg/config/v3" // Register v3 config
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

var InitFlags = []external.Flag{}

var InitMeta = plugin.SubcommandMetadata{
	Description: "The `init` subcommand of the sampleexternalplugin adds Prometheus instance configuration during project initialization",
	Examples: `
	Initialize a new project with Prometheus monitoring:
	$ kubebuilder init --plugins sampleexternalplugin/v1
	`,
}

// InitCmd handles all the logic for the `init` subcommand of this sample external plugin
func InitCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "init",
		Universe:   pr.Universe,
	}

	// For init command, we'll use default values since PROJECT file may not exist yet
	projectConfig := &ProjectConfig{
		ProjectName: "project",
	}

	// Create Prometheus instance manifest
	prometheusInstance := prometheus.NewPrometheusInstance(
		prometheus.WithProjectName(projectConfig.ProjectName),
	)

	// Create Kustomization for Prometheus resources
	prometheusKustomization := prometheus.NewPrometheusKustomization()

	// Create instructions for adding Prometheus to default kustomization
	kustomizationPatch := prometheus.NewDefaultKustomizationPatch()

	// Add files to universe
	pluginResponse.Universe[prometheusInstance.Path] = prometheusInstance.Content
	pluginResponse.Universe[prometheusKustomization.Path] = prometheusKustomization.Content
	pluginResponse.Universe[kustomizationPatch.Path] = kustomizationPatch.Content

	return pluginResponse
}
