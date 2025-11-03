/*
Copyright 2021 The Kubernetes Authors.

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
package external

import (
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	Path        string
	Args        []string
	pluginChain []string
	config      config.Config
}

// InjectConfig injects the project configuration to access plugin chain information

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	if c == nil {
		return nil
	}

	if chain := c.GetPluginChain(); len(chain) > 0 {
		p.pluginChain = append([]string(nil), chain...)
	}

	return nil
}

func (p *initSubcommand) SetPluginChain(chain []string) {
	if len(chain) == 0 {
		p.pluginChain = nil
		return
	}

	p.pluginChain = append([]string(nil), chain...)
}

func (p *initSubcommand) UpdateMetadata(_ plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	setExternalPluginMetadata("init", p.Path, subcmdMeta)
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	bindExternalPluginFlags(fs, "init", p.Path, p.Args)
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	req := external.PluginRequest{
		APIVersion:  defaultAPIVersion,
		Command:     "init",
		Args:        p.Args,
		PluginChain: p.pluginChain,
	}

	err := handlePluginResponse(fs, req, p.Path, p.config)
	if err != nil {
		return err
	}

	return nil
}
