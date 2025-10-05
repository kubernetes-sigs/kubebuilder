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
	"os"
	"path/filepath"

	"v1/scaffolds/internal/templates/prometheus"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
	yamlv3 "sigs.k8s.io/yaml"
)

var EditFlags = []external.Flag{}

var EditMeta = plugin.SubcommandMetadata{
	Description: "The `edit` subcommand of the sampleexternalplugin adds Prometheus ServiceMonitor configuration for monitoring your operator",
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

	// Create ServiceMonitor manifest
	serviceMonitor := prometheus.NewServiceMonitor(
		prometheus.WithDomain(projectConfig.Domain),
		prometheus.WithProjectName(projectConfig.ProjectName),
	)

	// Create Kustomization for Prometheus resources
	kustomization := prometheus.NewPrometheusKustomization()

	// Create Kustomization patch for default
	kustomizationPatch := prometheus.NewDefaultKustomizationPatch()

	// Add files to universe
	pluginResponse.Universe[serviceMonitor.Path] = serviceMonitor.Content
	pluginResponse.Universe[kustomization.Path] = kustomization.Content
	pluginResponse.Universe[kustomizationPatch.Path] = kustomizationPatch.Content

	return pluginResponse
}

// ProjectConfig represents the minimal PROJECT file structure we need
type ProjectConfig struct {
	Domain      string `yaml:"domain"`
	ProjectName string `yaml:"projectName"`
	Repo        string `yaml:"repo"`
}

// loadProjectConfig reads the PROJECT file and extracts necessary information
func loadProjectConfig() (*ProjectConfig, error) {
	projectPath := filepath.Join(".", "PROJECT")

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		// Return default values if PROJECT file doesn't exist
		return &ProjectConfig{
			Domain:      "example.com",
			ProjectName: "project",
			Repo:        "example.com/project",
		}, nil
	}

	data, err := os.ReadFile(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PROJECT file: %w", err)
	}

	var cfg ProjectConfig
	if err := yamlv3.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PROJECT file: %w", err)
	}

	// Set defaults if fields are empty
	if cfg.Domain == "" {
		cfg.Domain = "example.com"
	}
	if cfg.ProjectName == "" {
		cfg.ProjectName = "project"
		if cfg.Repo != "" {
			// Extract project name from repo path
			cfg.ProjectName = filepath.Base(cfg.Repo)
		}
	}
	if cfg.Repo == "" {
		cfg.Repo = cfg.Domain + "/" + cfg.ProjectName
	}

	return &cfg, nil
}
