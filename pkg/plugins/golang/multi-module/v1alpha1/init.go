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
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = subcmdMeta.Description + `
Warning: This will also create multiple go.mod files when creating APIs. 
If you are not careful, you can break your dependency chain.
The multi-module extension will create replace directives for local development, 
which you might want to drop after creating your first stable API.

For more information, visit 
https://github.com/golang/go/wiki/Modules#should-i-have-multiple-modules-in-a-single-repository
`
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	if err := InsertPluginMetaToConfig(p.config, pluginConfig{
		ApiGoModCreated: false,
	}); err != nil {
		return err
	}
	return nil
}
