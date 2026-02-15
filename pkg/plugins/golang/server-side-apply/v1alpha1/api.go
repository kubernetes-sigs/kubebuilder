/*
Copyright 2026 The Kubernetes Authors.

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
	"strings"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/server-side-apply/v1alpha1/scaffolds"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config config.Config

	options *goPlugin.Options

	resource *resource.Resource

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	//nolint:lll
	subcmdMeta.Description = `Scaffold an API with a controller using Server-Side Apply patterns for safer field management.

Server-Side Apply enables safer field management when resources are shared between your controller and users or other controllers. The API server tracks field ownership and prevents accidental overwrites.

Use this plugin when:
- Users customize your CRs (labels, annotations, fields)
- Multiple controllers manage the same resource
- You need partial field management
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Create an Application API with Server-Side Apply controller
  %[1]s create api --group apps --version v1 --kind Application --plugins="%[2]s"

  # Generate manifests and apply configurations
  make manifests generate

  # Run tests
  make test
`,
		cliMeta.CommandName,
		pluginName,
	)
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.runMake, "make", true,
		"if true, run `make generate` after generating files")

	p.options = &goPlugin.Options{}

	fs.StringVar(&p.options.Plural, "plural", "", "resource irregular plural form")

	fs.BoolVar(&p.options.DoAPI, "resource", true,
		"if set, generate the resource without prompting the user")

	fs.BoolVar(&p.options.Namespaced, "namespaced", true, "resource is namespaced")

	fs.BoolVar(&p.options.DoController, "controller", true,
		"if set, generate the controller without prompting the user")
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	// Initialize options if not already set
	if p.options == nil {
		p.options = &goPlugin.Options{
			DoAPI:        true,
			DoController: true,
			Namespaced:   true,
		}
	}

	p.options.UpdateResource(p.resource, p.config)

	if err := p.resource.Validate(); err != nil {
		return fmt.Errorf("error validating resource: %w", err)
	}

	// Force controller and resource to true for this plugin
	p.resource.Controller = true
	if p.resource.API == nil {
		p.resource.API = &resource.API{
			CRDVersion: "v1",
			Namespaced: p.options.Namespaced,
		}
	}

	if err := p.config.UpdateResource(*p.resource); err != nil {
		return fmt.Errorf("error updating resource: %w", err)
	}

	return nil
}

func (p *createAPISubcommand) PreScaffold(machinery.Filesystem) error {
	// Check for plugin conflicts
	for _, pluginKey := range p.config.GetPluginChain() {
		if strings.Contains(pluginKey, "deploy-image") {
			return fmt.Errorf("server-side-apply plugin cannot be used with deploy-image plugin. " +
				"Both plugins scaffold different controller implementations. " +
				"Use one or the other for each API")
		}
	}
	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewAPIScaffolder(p.config, *p.resource)
	scaffolder.InjectFS(fs)

	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding: %w", err)
	}

	return nil
}

func (p *createAPISubcommand) PostScaffold() error {
	// Store resource info in PROJECT file for alpha generate
	key := plugin.KeyFor(Plugin{})
	canonicalKey := plugin.KeyFor(Plugin{})
	cfg := PluginConfig{}

	if err := p.config.DecodePluginConfig(key, &cfg); err != nil {
		switch {
		case errors.As(err, &config.UnsupportedFieldError{}):
			// Config version doesn't support plugin metadata
			return nil
		case errors.As(err, &config.PluginKeyNotFoundError{}):
			if key != canonicalKey {
				if decodeErr := p.config.DecodePluginConfig(canonicalKey, &cfg); decodeErr != nil {
					if errors.As(decodeErr, &config.UnsupportedFieldError{}) {
						return nil
					}
					if !errors.As(decodeErr, &config.PluginKeyNotFoundError{}) {
						return fmt.Errorf("error decoding plugin configuration: %w", decodeErr)
					}
				}
			}
		default:
			return fmt.Errorf("error decoding plugin configuration: %w", err)
		}
	}

	// Add this resource to the config
	cfg.Resources = append(cfg.Resources, ResourceData{
		Group:   p.resource.Group,
		Domain:  p.resource.Domain,
		Version: p.resource.Version,
		Kind:    p.resource.Kind,
	})

	if err := p.config.EncodePluginConfig(key, cfg); err != nil {
		return fmt.Errorf("error encoding plugin configuration: %w", err)
	}

	return nil
}
