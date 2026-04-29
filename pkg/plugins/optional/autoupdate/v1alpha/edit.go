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

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config      config.Config
	useGHModels bool
	openGHIssue bool
	openGHPR    bool
	flagSet     *pflag.FlagSet

	// Merged config after Scaffold() - used in PostScaffold()
	mergedConfig PluginConfig
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = metaDataDescription

	subcmdMeta.Examples = fmt.Sprintf(`  # Edit a common project with this plugin (default: creates both Issues and PRs)
  %[1]s edit --plugins=%[2]s

  # Edit a common project with GitHub Models enabled (requires repo permissions)
  %[1]s edit --plugins=%[2]s --use-gh-models

  # Edit to create only PRs (no issue notifications)
  %[1]s edit --plugins=%[2]s --open-gh-issue=false

  # Edit to create only Issues (no PRs)
  %[1]s edit --plugins=%[2]s --open-gh-pr=false
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	p.flagSet = fs
	fs.BoolVar(&p.useGHModels, "use-gh-models", false,
		"If set, enable GitHub Models AI summary in the scaffolded workflow (requires GitHub Models permissions)")
	fs.BoolVar(&p.openGHIssue, "open-gh-issue", true,
		"By default, create GitHub Issues to notify about updates. Disable with --open-gh-issue=false")
	fs.BoolVar(&p.openGHPR, "open-gh-pr", true,
		"By default, create GitHub Pull Requests with the update changes. Disable with --open-gh-pr=false")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *editSubcommand) PreScaffold(machinery.Filesystem) error {
	if len(p.config.GetCliVersion()) == 0 {
		return fmt.Errorf(
			"you must manually upgrade your project to a version that records the CLI version in PROJECT (`cliVersion`) " +
				"to allow the `kubebuilder alpha update` command to work properly before using this plugin.\n" +
				"More info: https://book.kubebuilder.io/migrations",
		)
	}
	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	var cfg PluginConfig

	// Use flag values (includes Cobra defaults)
	cfg.UseGHModels = p.useGHModels
	cfg.OpenGHIssue = p.openGHIssue
	cfg.OpenGHPR = p.openGHPR

	// Validate the merged config: --use-gh-models requires --open-gh-pr
	// AI summaries only work with PRs
	if cfg.UseGHModels && !cfg.OpenGHPR {
		return fmt.Errorf(
			"the --use-gh-models flag requires --open-gh-pr=true " +
				"(AI summaries only work with Pull Requests)")
	}

	if err := insertPluginMetaToConfig(p.config, cfg); err != nil {
		return fmt.Errorf("error inserting project plugin meta to configuration: %w", err)
	}

	// Store merged config for PostScaffold()
	p.mergedConfig = cfg

	// Always overwrite the workflow file to keep it in sync with configuration
	scaffolder := scaffolds.NewInitScaffolder(cfg.UseGHModels, cfg.OpenGHIssue, cfg.OpenGHPR)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding edit subcommand: %w", err)
	}

	return nil
}

func (p *editSubcommand) PostScaffold() error {
	// Inform users about GitHub Models if they didn't enable it
	// Note: AI summaries only work when PRs are enabled
	if !p.mergedConfig.UseGHModels && p.mergedConfig.OpenGHPR {
		log.Info("Consider enabling GitHub Models to get an AI summary in PRs")
		log.Info("Use the --use-gh-models flag if your project/organization has permission to use GitHub Models")
	}
	return nil
}
