/*
Copyright 2020 The Kubernetes Authors.

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

package v3

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/cmdutil"
)

type editSubcommand struct {
	config config.Config

	multigroup bool
}

var (
	_ plugin.EditSubcommand = &editSubcommand{}
	_ cmdutil.RunOptions    = &editSubcommand{}
)

func (p *editSubcommand) UpdateMetadata(meta plugin.CLIMetadata) plugin.CommandMetadata {
	return plugin.CommandMetadata{
		Description: `This command will edit the project configuration. You can have single or multi group project.`,
		Examples: fmt.Sprintf(`# Enable the multigroup layout
        %[1]s edit --multigroup

        # Disable the multigroup layout
        %[1]s edit --multigroup=false
	`, meta.CommandName),
	}
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.multigroup, "multigroup", false, "enable or disable multigroup layout")
}

func (p *editSubcommand) InjectConfig(c config.Config) {
	p.config = c
}

func (p *editSubcommand) Run(fs afero.Fs) error {
	return cmdutil.Run(p, fs)
}

func (p *editSubcommand) Validate() error {
	return nil
}

func (p *editSubcommand) GetScaffolder() (cmdutil.Scaffolder, error) {
	return scaffolds.NewEditScaffolder(p.config, p.multigroup), nil
}

func (p *editSubcommand) PostScaffold() error {
	return nil
}
