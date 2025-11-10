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
	"errors"
	"fmt"
	"os"

	"github.com/spf13/afero"

	"v1/internal/test/plugins/prometheus"

	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	_ "sigs.k8s.io/kubebuilder/v4/pkg/config/v3" // Register v3 config
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

var EditFlags = []external.Flag{}

var EditMeta = plugin.SubcommandMetadata{
	Description: "The `edit` subcommand of the sampleexternalplugin adds Prometheus instance configuration for monitoring your operator",
	Examples: `
	Add Prometheus monitoring to your project:
	$ kubebuilder edit --plugins sampleexternalplugin/v1
	`,
}

// EditCmd handles all the logic for the `edit` subcommand of this sample external plugin
func EditCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "edit",
		Universe:   pr.Universe,
	}

	// Load PROJECT config to get domain and other metadata
	projectConfig, err := loadProjectConfig()
	if err != nil {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			fmt.Sprintf("failed to load PROJECT config: %s", err.Error()),
		}
		return pluginResponse
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

// ProjectConfig represents the minimal PROJECT file structure we need
type ProjectConfig struct {
	ProjectName string
}

// loadProjectConfig reads the PROJECT file using the kubebuilder config API.
func loadProjectConfig() (*ProjectConfig, error) {
	store := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := store.Load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("PROJECT file does not exist; please run 'init' first")
		}
		return nil, fmt.Errorf("failed to load PROJECT file: %w", err)
	}

	cfg := store.Config()
	if cfg == nil {
		return nil, fmt.Errorf("PROJECT file is empty or invalid")
	}

	projectName := cfg.GetProjectName()
	if projectName == "" {
		projectName = "project"
	}

	return &ProjectConfig{
		ProjectName: projectName,
	}, nil
}
