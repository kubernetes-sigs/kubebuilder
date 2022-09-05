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

package v1alpha1

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/multi-module/v1alpha1/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config config.Config

	multimodule     bool
	canUseAPIModule bool
	pluginConfig
	apiPath string
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = subcmdMeta.Description + `
  - Toggle between single or multi module projects.
`
	subcmdMeta.Examples = fmt.Sprintf(subcmdMeta.Examples+`
  # Enable the multimodule layout
  %[1]s edit --multimodule

  # Disable the multimodule layout
  %[1]s edit --multimodule=false
`, cliMeta.CommandName)
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.multimodule, "multimodule", false, "enable or disable multimodule layout")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	// Track the config and ensure it exists and can be parsed
	cfg := pluginConfig{}
	if err := p.config.DecodePluginConfig(pluginKey, &cfg); errors.As(err, &config.UnsupportedFieldError{}) {
		// Config doesn't support per-plugin configuration, so we can't track them
	} else {
		// Fail unless they key wasn't found, which just means it is the first resource tracked
		if err != nil && !errors.As(err, &config.PluginKeyNotFoundError{}) {
			return err
		}
	}
	p.pluginConfig = cfg

	if res, err := p.config.GetResources(); err != nil {
		return err
	} else if len(res) == 0 {
		p.canUseAPIModule = false
	} else {
		foundAtLeastOneAPI := false
		for i := range res {
			if res[i].HasAPI() {
				foundAtLeastOneAPI = true
				break
			}
		}
		p.canUseAPIModule = foundAtLeastOneAPI
	}

	p.apiPath = getAPIPath(p.config.IsMultiGroup())

	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	if !p.canUseAPIModule {
		return nil
	}

	if p.multimodule && p.pluginConfig.ApiGoModCreated {
		return nil
	}

	if !p.multimodule && !p.pluginConfig.ApiGoModCreated {
		return nil
	}

	scaffolder := scaffolds.NewAPIScaffolder(p.config, p.apiPath, p.multimodule)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return err
	}

	// we are not doing this in PostScaffold in order to avoid a wrong tidy order with other plugins relaying on tidying up
	// the main module, e.g. declarative
	if err := tidyGoModForAPI(p.apiPath); err != nil {
		return err
	}

	p.pluginConfig.ApiGoModCreated = p.multimodule

	return p.config.EncodePluginConfig(pluginKey, p.pluginConfig)
}
