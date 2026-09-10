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

package v1alpha

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ plugin.DeleteSubcommand = &deleteSubcommand{}

type deleteSubcommand struct {
	config config.Config
}

func (p *deleteSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = "Remove Grafana dashboard files added by this plugin"
	subcmdMeta.Examples = fmt.Sprintf(`  # Remove Grafana dashboards from the project
  %[1]s delete --plugins=%[2]s
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *deleteSubcommand) BindFlags(_ *pflag.FlagSet) {}

func (p *deleteSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *deleteSubcommand) Scaffold(fs machinery.Filesystem) error {
	return deleteGrafanaManifests(p.config, fs)
}

// deleteGrafanaManifests removes Grafana dashboard files and clears the plugin's PROJECT entry.
func deleteGrafanaManifests(cfg config.Config, fs machinery.Filesystem) error {
	grafanaFiles := []string{
		filepath.Join("grafana", "controller-runtime-metrics.json"),
		filepath.Join("grafana", "controller-resources-metrics.json"),
		filepath.Join("grafana", "custom-metrics", "config.yaml"),
		filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json"),
	}

	deleted := 0
	for _, file := range grafanaFiles {
		if exists, _ := afero.Exists(fs.FS, file); exists {
			if err := fs.FS.Remove(file); err != nil {
				slog.Warn("Failed to delete Grafana file", "file", file, "error", err)
			} else {
				deleted++
			}
		}
	}

	// Remove directories after files so removal succeeds even if files were already gone.
	if exists, _ := afero.DirExists(fs.FS, "grafana/custom-metrics"); exists {
		_ = fs.FS.RemoveAll("grafana/custom-metrics")
	}
	if exists, _ := afero.DirExists(fs.FS, "grafana"); exists {
		_ = fs.FS.RemoveAll("grafana")
	}

	if err := plugin.RemovePluginConfig(cfg, Plugin{}); err != nil {
		slog.Warn("Failed to remove plugin config from PROJECT file", "error", err)
	}

	fmt.Printf("Deleted %d Grafana manifest(s)\n", deleted)
	return nil
}
