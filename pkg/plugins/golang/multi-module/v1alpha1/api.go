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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/multi-module/v1alpha1/scaffolds"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config config.Config

	resource *resource.Resource
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Scaffold a Kubernetes API by writing a Resource definition and/or a Controller.

If information about whether the resource and controller should be scaffolded
was not explicitly provided, it will prompt the user if they should be.

After the scaffold is written, the dependencies will be updated and
make generate will be run.

Warning: This will also create multiple go.mod files. If you are not careful, you can break your dependency chain.
The multi-module extension will create replace directives for local development, 
which you might want to drop after creating your first stable API.

For more information, visit 
https://github.com/golang/go/wiki/Modules#should-i-have-multiple-modules-in-a-single-repository
`
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c

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
	fmt.Println("updating scaffold with multi-module support...")

	fmt.Println("using gvk:", p.resource.Group, p.resource.Domain, p.resource.Version)
	apiPath := GetAPIPath(p.config.IsMultiGroup(), p.resource)
	goModPath := filepath.Join(apiPath, "go.mod")

	fmt.Println("using go.mod path: " + goModPath)
	scaffolder := scaffolds.NewAPIScaffolder(p.config, *p.resource, goModPath)
	scaffolder.InjectFS(fs)
	err := scaffolder.Scaffold()
	if err != nil {
		return err
	}

	if err := util.RunInDir(apiPath, func() error {
		err = util.RunCmd("Update dependencies in "+apiPath, "go", "mod", "tidy")
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	if err := util.RunCmd("Add require directive of API module", "go", "mod", "edit", "-require",
		p.resource.Path+"@v0.0.0-v1alpha1"); err != nil {
		return err
	}

	if err := util.RunCmd("Update dependencies", "go", "mod", "edit", "-replace",
		p.resource.Path+"="+"."+string(filepath.Separator)+apiPath); err != nil {
		return err
	}

	// Update Dockerfile
	err = updateDockerfile(apiPath)
	if err != nil {
		return err
	}

	// Track the resources following a declarative approach
	cfg := pluginConfig{}
	if err := p.config.DecodePluginConfig(pluginKey, &cfg); errors.As(err, &config.UnsupportedFieldError{}) {
		// Config doesn't support per-plugin configuration, so we can't track them
	} else {
		// Fail unless they key wasn't found, which just means it is the first resource tracked
		if err != nil && !errors.As(err, &config.PluginKeyNotFoundError{}) {
			return err
		}

		cfg.Resources = append(cfg.Resources, resourceData{
			Group:   p.resource.GVK.Group,
			Domain:  p.resource.GVK.Domain,
			Version: p.resource.GVK.Version,
			Kind:    p.resource.GVK.Kind,
		})

		if err := p.config.EncodePluginConfig(pluginKey, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (p *createAPISubcommand) PostScaffold() error {
	err := util.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return err
	}

	return nil
}
