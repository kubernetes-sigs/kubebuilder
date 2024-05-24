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

package v4

import (
	"fmt"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config config.Config

	multigroup bool
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `This command will edit the project configuration.
Features supported:
  - Toggle between single or multi group projects.
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Enable the multigroup layout
  %[1]s edit --multigroup

  # Disable the multigroup layout
  %[1]s edit --multigroup=false
`, cliMeta.CommandName)
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.multigroup, "multigroup", false, "enable or disable multigroup layout")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewEditScaffolder(p.config, p.multigroup)
	scaffolder.InjectFS(fs)
	return scaffolder.Scaffold()
}
