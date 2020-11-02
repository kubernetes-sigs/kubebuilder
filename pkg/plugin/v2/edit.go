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

package v2

import (
	"fmt"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v2/scaffolds"
)

type editPlugin struct {
	config *config.Config

	multigroup bool
}

var (
	_ plugin.Edit        = &editPlugin{}
	_ cmdutil.RunOptions = &editPlugin{}
)

func (p *editPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `This command will edit the project configuration. You can have single or multi group project.`

	ctx.Examples = fmt.Sprintf(`# Enable the multigroup layout
        %s edit --multigroup

        # Disable the multigroup layout
        %s edit --multigroup=false
	`, ctx.CommandName, ctx.CommandName)
}

func (p *editPlugin) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.multigroup, "multigroup", false, "enable or disable multigroup layout")
}

func (p *editPlugin) InjectConfig(c *config.Config) {
	// v3 project configs get a 'layout' value.
	if c.IsV3() {
		c.Layout = plugin.KeyFor(Plugin{})
	}
	p.config = c
}

func (p *editPlugin) Run() error {
	return cmdutil.Run(p)
}

func (p *editPlugin) Validate() error {
	return nil
}

func (p *editPlugin) GetScaffolder() (scaffold.Scaffolder, error) {
	return scaffolds.NewEditScaffolder(p.config, p.multigroup), nil
}

func (p *editPlugin) PostScaffold() error {
	return nil
}
