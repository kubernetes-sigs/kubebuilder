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

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/autoupdate/v1alpha/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config config.Config
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = metaDataDescription

	subcmdMeta.Examples = fmt.Sprintf(`  # Edit a common project with this plugin
  %[1]s edit --plugins=%[2]s
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
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
	if err := insertPluginMetaToConfig(p.config, pluginConfig{}); err != nil {
		return fmt.Errorf("error inserting project plugin meta to configuration: %w", err)
	}

	scaffolder := scaffolds.NewInitScaffolder()
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding edit subcommand: %w", err)
	}

	return nil
}
