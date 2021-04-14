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

package v1

import (
	"errors"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/config-gen/v1/scaffolds"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config   config.Config
	resource *resource.Resource
}

func (p *createAPISubcommand) UpdateMetadata(plugin.CLIMetadata, *plugin.SubcommandMetadata) {}
func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet)                                   {}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res
	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	cfg := Config{}
	if err := p.config.DecodePluginConfig(pluginName, &cfg); err != nil {
		keyNotFoundErr := config.PluginKeyNotFoundError{}
		if !errors.As(err, &keyNotFoundErr) {
			return err
		}
	}

	scaffolder := scaffolds.NewAPIScaffolder(p.config, p.resource, cfg.WithKustomize)
	scaffolder.InjectFS(fs)
	return scaffolder.Scaffold()
}
