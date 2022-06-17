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
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config   config.Config
	resource *resource.Resource
	image    string
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	//nolint: lll
	subcmdMeta.Description = `Scaffold the code implementation to deploy and manage your Operand which is represented by the API 
	informed and will be reconciled by its controller. This plugin will generate the code implementation to help you out.
	
	Note: In general, it’s recommended to have one controller responsible for managing each API created for the project to properly 
	follow the design goals set by Controller Runtime(https://github.com/kubernetes-sigs/controller-runtime).
`
	//nolint: lll
	subcmdMeta.Examples = fmt.Sprintf(`  # Create a frigates API with Group: ship, Version: v1beta1, Kind: Frigate to represent the 
	Image: example.com/frigate:v0.0.1 and its controller with a code to deploy and manage this Operand.
  %[1]s create api --group ship --version v1beta1 --kind Frigate --image example.com/frigate:v0.0.1

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

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	//nolint: lll
	fs.StringVar(&p.image, "image", "", "inform the Operand image. The controller will be scaffolded with an example code to deploy and manage this image.")
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	if len(p.image) == 0 {
		return fmt.Errorf("you must inform the image ")
	}

	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	fmt.Println("updating scaffold with deploy-image/v1alpha1 plugin...")

	scaffolder := scaffolds.NewAPIScaffolder(p.config, *p.resource, p.image)
	scaffolder.InjectFS(fs)
	err := scaffolder.Scaffold()
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

		cfg.Resources = append(cfg.Resources, p.resource.GVK)
		if err := p.config.EncodePluginConfig(pluginKey, cfg); err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	return nil
}
