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
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/declarative/v1/internal/templates"
	goPluginV2 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2"
)

const (
	// kbDeclarativePattern is the sigs.k8s.io/kubebuilder-declarative-pattern version
	kbDeclarativePatternForV2 = "v0.0.0-20200522144838-848d48e5b073"
	kbDeclarativePatternForV3 = "f77bb4933dfbae404f03e34b01c84e268cc4b966"

	exampleManifestVersion = "0.0.1"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config config.Config

	resource *resource.Resource
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `
Scaffold a Kubernetes API by writing a Resource definition and a Controller.

After the scaffold is written, the dependencies will be updated and
make generate will be run.
`
	subcmdMeta.Examples = fmt.Sprintf(` # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
  %[1]s create api --group ship --version v1beta1 --kind Frigate --resource --controller

  # Edit the API Scheme
  nano api/v1beta1/frigate_types.go

  # Edit the Controller Test
  nano controllers/frigate/frigate_controller_test.go

  # Generate the manifests
  make manifests

  # Install CRDs into the Kubernetes cluster using kubectl apply
  make install

  # Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
  make run
`, cliMeta.CommandName)
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

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	fmt.Println("updating scaffold with declarative pattern...")

	// Load the boilerplate
	bp, err := afero.ReadFile(fs.FS, filepath.Join("hack", "boilerplate.go.txt"))
	if err != nil {
		return fmt.Errorf("error updating scaffold: unable to load boilerplate: %w", err)
	}
	boilerplate := string(bp)

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(p.config),
		machinery.WithBoilerplate(boilerplate),
		machinery.WithResource(p.resource),
	)

	if err := scaffold.Execute(
		&templates.Types{},
		&templates.Controller{},
		&templates.Channel{ManifestVersion: exampleManifestVersion},
		&templates.Manifest{ManifestVersion: exampleManifestVersion},
	); err != nil {
		return fmt.Errorf("error updating scaffold: %w", err)
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

		cfg.Resources = append(cfg.Resources, p.resource.GVK)
		if err := p.config.EncodePluginConfig(pluginKey, cfg); err != nil {
			return err
		}
	}

	// Ensure that we are pinning sigs.k8s.io/kubebuilder-declarative-pattern version
	// Just pin an old value for go/v2. It shows fine for now. However, we should improve/change it
	// if we see that more rules based on the plugins version are required.
	kbDeclarativePattern := kbDeclarativePatternForV3
	for _, pluginKey := range p.config.GetPluginChain() {
		if pluginKey == plugin.KeyFor(goPluginV2.Plugin{}) {
			kbDeclarativePattern = kbDeclarativePatternForV2
			break
		}
	}
	err = util.RunCmd("Get declarative pattern", "go", "get",
		"sigs.k8s.io/kubebuilder-declarative-pattern@"+kbDeclarativePattern)
	if err != nil {
		return err
	}

	return nil
}
