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
	log "log/slog"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/autoupdate/v1alpha/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config      config.Config
	useGHModels bool
	delete      bool
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = metaDataDescription

	subcmdMeta.Examples = fmt.Sprintf(`  # Edit a common project with this plugin
  %[1]s edit --plugins=%[2]s

  # Edit a common project with GitHub Models enabled (requires repo permissions)
  %[1]s edit --plugins=%[2]s --use-gh-models
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.useGHModels, "use-gh-models", false,
		"enable GitHub Models AI summary in the scaffolded workflow (requires GitHub Models permissions)")
	fs.BoolVar(&p.delete, "delete", false, "delete auto-update workflow from the project")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *editSubcommand) PreScaffold(machinery.Filesystem) error {
	if len(p.config.GetCliVersion()) == 0 {
		return fmt.Errorf(
			"you must manually upgrade your project to a version that records the CLI version in PROJECT (`cliVersion`) " +
				"to allow the `alpha update` command to work properly before using this plugin.\n" +
				"More info: https://book.kubebuilder.io/migrations",
		)
	}
	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	if p.delete {
		return p.deleteAutoUpdateWorkflow(fs)
	}

	if err := insertPluginMetaToConfig(p.config, PluginConfig{UseGHModels: p.useGHModels}); err != nil {
		return fmt.Errorf("error inserting project plugin meta to configuration: %w", err)
	}

	scaffolder := scaffolds.NewInitScaffolder(p.useGHModels)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding edit subcommand: %w", err)
	}

	return nil
}

func (p *editSubcommand) deleteAutoUpdateWorkflow(fs machinery.Filesystem) error {
	log.Info("Deleting auto-update workflow...")

	workflowFile := filepath.Join(".github", "workflows", "auto_update.yml")

	if exists, _ := afero.Exists(fs.FS, workflowFile); exists {
		if err := fs.FS.Remove(workflowFile); err != nil {
			log.Warn("Failed to delete workflow file", "file", workflowFile, "error", err)
		} else {
			log.Info("Deleted workflow file", "file", workflowFile)
			fmt.Println("\nDeleted auto-update workflow")
		}
	} else {
		log.Warn("Workflow file not found", "file", workflowFile)
		fmt.Println("\nWorkflow file not found (may have been already deleted)")
	}

	// Remove plugin config from PROJECT by encoding empty struct
	// Empty struct will be omitted from YAML during marshaling
	key := plugin.GetPluginKeyForConfig(p.config.GetPluginChain(), Plugin{})
	if err := p.config.EncodePluginConfig(key, struct{}{}); err != nil {
		canonicalKey := plugin.KeyFor(Plugin{})
		if key != canonicalKey {
			if err2 := p.config.EncodePluginConfig(canonicalKey, struct{}{}); err2 != nil {
				log.Warn("Failed to remove plugin configuration from PROJECT file",
					"provided_key_error", err, "canonical_key_error", err2)
			}
		} else {
			log.Warn("Failed to remove plugin config", "error", err)
		}
	}

	return nil
}

func (p *editSubcommand) PostScaffold() error {
	// Inform users about GitHub Models if they didn't enable it
	if !p.useGHModels {
		log.Info("Consider enabling GitHub Models to get an AI summary to help with the update")
		log.Info("Use the --use-gh-models flag if your project/organization has permission to use GitHub Models")
	}
	return nil
}
