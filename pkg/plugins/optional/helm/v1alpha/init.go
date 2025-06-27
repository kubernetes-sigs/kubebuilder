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
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v1alpha/scaffolds"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config

	// config options
	directory string
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Initialize a helm chart to distribute the project under dist/
`
	subcmdMeta.Examples = fmt.Sprintf(`# Initialize a helm chart to distribute the project under dist/
  %[1]s init --plugins=%[2]s

**IMPORTANT** You must use %[1]s edit --plugins=%[2]s to update the chart when changes are made.
`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&p.directory, "directory", HelmDefaultTargetDirectory, "domain for groups")
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewInitHelmScaffolder(p.config, false, p.directory)
	scaffolder.InjectFS(fs)
	err := scaffolder.Scaffold()
	if err != nil {
		return fmt.Errorf("error scaffolding helm chart: %w", err)
	}

	// Track the resources following a declarative approach
	cfg := PluginConfig{}
	if err = p.config.DecodePluginConfig(pluginKey, &cfg); errors.As(err, &config.UnsupportedFieldError{}) {
		// Skip tracking as the config doesn't support per-plugin configuration
		return nil
	} else if err != nil && !errors.As(err, &config.PluginKeyNotFoundError{}) {
		// Fail unless the key wasn't found, which just means it is the first resource tracked
		return fmt.Errorf("error decoding plugin configuration: %w", err)
	}

	cfg.Options = options{
		Directory: p.directory,
	}

	if err = p.config.EncodePluginConfig(pluginKey, cfg); err != nil {
		return fmt.Errorf("error encoding plugin configuration: %w", err)
	}

	return nil
}
