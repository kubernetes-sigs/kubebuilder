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
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/grafana/v1alpha/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config config.Config
	delete bool
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = metaDataDescription

	subcmdMeta.Examples = fmt.Sprintf(`  # Edit a common project with this plugin
  %[1]s edit --plugins=%[2]s
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.delete, "delete", false, "delete Grafana manifests from the project")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	if p.delete {
		return p.deleteGrafanaManifests(fs)
	}

	if err := InsertPluginMetaToConfig(p.config, pluginConfig{}); err != nil {
		return fmt.Errorf("error inserting project plugin meta to configuration: %w", err)
	}

	scaffolder := scaffolds.NewEditScaffolder()
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding edit subcommand: %w", err)
	}

	return nil
}

func (p *editSubcommand) deleteGrafanaManifests(fs machinery.Filesystem) error {
	slog.Info("Deleting Grafana manifests...")

	grafanaFiles := []string{
		filepath.Join("grafana", "controller-runtime-metrics.json"),
		filepath.Join("grafana", "custom-metrics", "config.yaml"),
		filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json"),
	}

	deletedCount := 0
	for _, file := range grafanaFiles {
		if exists, _ := afero.Exists(fs.FS, file); exists {
			if err := fs.FS.Remove(file); err != nil {
				slog.Warn("Failed to delete Grafana file", "file", file, "error", err)
			} else {
				slog.Info("Deleted Grafana file", "file", file)
				deletedCount++
			}
		}
	}

	// Try to remove directories (using RemoveAll to handle non-empty directories)
	if exists, _ := afero.DirExists(fs.FS, "grafana/custom-metrics"); exists {
		_ = fs.FS.RemoveAll("grafana/custom-metrics")
	}
	if exists, _ := afero.DirExists(fs.FS, "grafana"); exists {
		_ = fs.FS.RemoveAll("grafana")
	}

	// Remove plugin config from PROJECT by encoding empty struct
	key := plugin.GetPluginKeyForConfig(p.config.GetPluginChain(), Plugin{})
	if err := p.config.EncodePluginConfig(key, struct{}{}); err != nil {
		canonicalKey := plugin.KeyFor(Plugin{})
		if key != canonicalKey {
			if err2 := p.config.EncodePluginConfig(canonicalKey, struct{}{}); err2 != nil {
				slog.Warn("Failed to remove plugin configuration from PROJECT file",
					"provided_key_error", err, "canonical_key_error", err2)
			}
		} else {
			slog.Warn("Failed to remove plugin config", "error", err)
		}
	}

	fmt.Printf("\nDeleted %d Grafana manifest(s)\n", deletedCount)
	fmt.Println("Grafana dashboards removed from project")

	return nil
}
