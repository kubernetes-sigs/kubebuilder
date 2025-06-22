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

//nolint:dupl
package v1alpha

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
)

var _ plugin.Subcommand = &subcommand{}

type subcommand struct {
	config             config.Config
	scaffolder         plugins.Scaffolder
	cmd                string
	exampleDescription string
}

func (p *subcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = metaDataDescription

	subcmdMeta.Examples = fmt.Sprintf(`  %s
  %[1]s %s --plugins=%[2]s
`, p.exampleDescription, cliMeta.CommandName, p.cmd, pluginKey)
}

func (p *subcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *subcommand) Scaffold(fs machinery.Filesystem) error {
	if err := InsertPluginMetaToConfig(p.config, pluginConfig{}); err != nil {
		return fmt.Errorf("error inserting project plugin meta to configuration: %w", err)
	}

	p.scaffolder.InjectFS(fs)
	if err := p.scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding %q subcommand: %w", p.cmd, err)
	}

	return nil
}
