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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/declarative/v1/internal/templates"
	goPluginV3 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/util"
)

const (
	// kbDeclarativePattern is the sigs.k8s.io/kubebuilder-declarative-pattern version
	kbDeclarativePatternForV2 = "v0.0.0-20200522144838-848d48e5b073"
	kbDeclarativePatternForV3 = "v0.0.0-20210113160450-b84d99da0217"

	exampleManifestVersion = "0.0.1"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config config.Config

	resource *resource.Resource
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	if !p.resource.HasAPI() || !p.resource.HasController() {
		return plugin.ExitError{
			Plugin: pluginName,
			Reason: "declarative pattern is only supported when API and controller are scaffolded",
		}
	}

	return nil
}

func (p *createAPISubcommand) Scaffold(fs afero.Fs) error {
	fmt.Println("updating scaffold with declarative pattern...")

	// Load the boilerplate
	bp, err := afero.ReadFile(fs, filepath.Join("hack", "boilerplate.go.txt"))
	if err != nil {
		return fmt.Errorf("error updating scaffold: unable to load boilerplate: %w", err)
	}
	boilerplate := string(bp)

	if err := machinery.NewScaffold(fs).Execute(
		model.NewUniverse(
			model.WithConfig(p.config),
			model.WithBoilerplate(boilerplate),
			model.WithResource(p.resource),
		),
		&templates.Types{},
		&templates.Controller{},
		&templates.Channel{ManifestVersion: exampleManifestVersion},
		&templates.Manifest{ManifestVersion: exampleManifestVersion},
	); err != nil {
		return fmt.Errorf("error updating scaffold: %w", err)
	}

	// Ensure that we are pinning sigs.k8s.io/kubebuilder-declarative-pattern version
	kbDeclarativePattern := kbDeclarativePatternForV2
	if strings.Split(p.config.GetLayout(), ",")[0] == plugin.KeyFor(goPluginV3.Plugin{}) {
		kbDeclarativePattern = kbDeclarativePatternForV3
	}
	err = util.RunCmd("Get declarative pattern", "go", "get",
		"sigs.k8s.io/kubebuilder-declarative-pattern@"+kbDeclarativePattern)
	if err != nil {
		return err
	}

	return nil
}
