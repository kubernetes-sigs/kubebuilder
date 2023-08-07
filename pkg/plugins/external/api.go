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

package external

import (
	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/external"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

const (
	defaultAPIVersion = "v1alpha1"
)

type createAPISubcommand struct {
	Path   string
	Args   []string
	config config.Config
}

func (p *createAPISubcommand) InjectResource(*resource.Resource) error {
	// Do nothing since resource flags are passed to the external plugin directly.
	return nil
}

func (p *createAPISubcommand) UpdateMetadata(_ plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	setExternalPluginMetadata("api", p.Path, subcmdMeta)
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	bindExternalPluginFlags(fs, "api", p.Path, p.Args)
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	cfg := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := cfg.Load(); err != nil {
		return err
	}

	req := external.PluginRequest{
		APIVersion: defaultAPIVersion,
		Command:    "create api",
		Args:       p.Args,
		Config:     cfg.Config(),
	}

	err := handlePluginResponse(fs, req, p.Path, p)
	if err != nil {
		return err
	}

	return nil
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *createAPISubcommand) GetConfig() config.Config {
	return p.config
}
