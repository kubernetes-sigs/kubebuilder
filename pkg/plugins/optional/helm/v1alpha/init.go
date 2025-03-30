/*
Copyright 2024 The Kubernetes Authors.

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
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Initialize a helm chart to distribute the project under dist/
`
	subcmdMeta.Examples = fmt.Sprintf(`# Initialize a helm chart to distribute the project under dist/
  %[1]s init --plugins=%[2]s

**IMPORTANT** You must use %[1]s edit --plugins=%[2]s to update the chart when changes are made.
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewInitHelmScaffolder(p.config, false)
	scaffolder.InjectFS(fs)
	err := scaffolder.Scaffold()
	if err != nil {
		return fmt.Errorf("error scaffolding helm chart: %w", err)
	}

	// Track the resources following a declarative approach
	return insertPluginMetaToConfig(p.config, pluginConfig{})
}
