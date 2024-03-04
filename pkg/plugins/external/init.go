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
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/external"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	Path   string
	Args   []string
	config config.Config
}

func (p *initSubcommand) UpdateMetadata(_ plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	setExternalPluginMetadata("init", p.Path, subcmdMeta)
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	bindExternalPluginFlags(fs, "init", p.Path, p.Args)
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	fileName := "PROJECT"

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		file, err := os.Create(fileName)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := file.Close(); cerr != nil {
				if err == nil {
					err = cerr
				}
			}
		}()
	}

	cfg := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := cfg.New(config.Version{Number: 3}); err != nil {
		return err
	}

	if err := cfg.Load(); err != nil {
		return err
	}

	req := external.PluginRequest{
		APIVersion: defaultAPIVersion,
		Command:    "init",
		Args:       p.Args,
		Config:     cfg.Config(),
	}

	err := handlePluginResponse(fs, req, p.Path, p)
	if err != nil {
		return err
	}

	return nil
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *initSubcommand) GetConfig() config.Config {
	return p.config
}
