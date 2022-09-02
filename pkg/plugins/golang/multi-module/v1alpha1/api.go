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
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config config.Config

	resource *resource.Resource

	pluginConfig

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	p.runMake, _ = fs.GetBool("make")
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = subcmdMeta.Description + `
Warning: This will also create multiple go.mod files. If you are not careful, you can break your dependency chain.
The multi-module extension will create replace directives for local development, 
which you might want to drop after creating your first stable API.

For more information, visit 
https://github.com/golang/go/wiki/Modules#should-i-have-multiple-modules-in-a-single-repository
`
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
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

	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	if !p.resource.HasAPI() {
		return plugin.ExitError{
			Plugin: pluginName,
			Reason: "multi-module pattern is only supported when API is scaffolded",
		}
	}

	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	if p.pluginConfig.ApiGoModCreated {
		fmt.Println("using existing multi-module layout, updating submodules...")
		return tidyGoModForAPI(p.config.IsMultiGroup())
	}

	if err := createGoModForAPI(fs, p.config); err != nil {
		return err
	}
	if err := tidyGoModForAPI(p.config.IsMultiGroup()); err != nil {
		return err
	}

	p.pluginConfig.ApiGoModCreated = true

	return p.config.EncodePluginConfig(pluginKey, p.pluginConfig)
}
