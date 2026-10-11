/*
Copyright 2025 The Kubernetes Authors.

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
	subcmdMeta.Description = "Remove the auto-update GitHub Actions workflow added by this plugin"
	subcmdMeta.Examples = fmt.Sprintf(`  # Remove the auto-update workflow from the project
  %[1]s delete --plugins=%[2]s
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *deleteSubcommand) BindFlags(_ *pflag.FlagSet) {}

func (p *deleteSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *deleteSubcommand) Scaffold(fs machinery.Filesystem) error {
	return deleteAutoUpdate(p.config, fs)
}

// deleteAutoUpdate removes the auto-update workflow file and clears the plugin's PROJECT entry.
func deleteAutoUpdate(cfg config.Config, fs machinery.Filesystem) error {
	workflowFile := filepath.Join(".github", "workflows", "auto_update.yml")

	if exists, _ := afero.Exists(fs.FS, workflowFile); exists {
		if err := fs.FS.Remove(workflowFile); err != nil {
			slog.Warn("Failed to delete auto-update workflow", "file", workflowFile, "error", err)
		} else {
			fmt.Println("Deleted auto-update workflow")
		}
	} else {
		slog.Warn("Auto-update workflow not found; may already be deleted", "file", workflowFile)
	}

	if err := plugin.RemovePluginConfig(cfg, Plugin{}); err != nil {
		slog.Warn("Failed to remove plugin config from PROJECT file", "error", err)
	}

	return nil
}
