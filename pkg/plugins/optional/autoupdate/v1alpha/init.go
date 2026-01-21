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

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/autoupdate/v1alpha/scaffolds"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config      config.Config
	useGHModels bool
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = metaDataDescription

	subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a common project with this plugin
  %[1]s init --plugins=%[2]s

  # Initialize with GitHub Models enabled (requires repo permissions)
  %[1]s init --plugins=%[2]s --use-gh-models
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.useGHModels, "use-gh-models", false,
		"enable GitHub Models AI summary in the scaffolded workflow (requires GitHub Models permissions)")
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	if err := insertPluginMetaToConfig(p.config, PluginConfig{UseGHModels: p.useGHModels}); err != nil {
		return fmt.Errorf("error inserting project plugin meta to configuration: %w", err)
	}

	scaffolder := scaffolds.NewInitScaffolder(p.useGHModels)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding init subcommand: %w", err)
	}

	return nil
}

func (p *initSubcommand) PostScaffold() error {
	// Inform users about GitHub Models if they didn't enable it
	if !p.useGHModels {
		log.Info("Consider enabling GitHub Models to get an AI summary to help with the update")
		log.Info("Use the --use-gh-models flag if your project/organization has permission to use GitHub Models")
	}
	return nil
}
