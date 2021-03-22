/*
Copyright 2020 The Kubernetes Authors.

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

package v3

import (
	"fmt"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	goPlugin "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
)

// defaultWebhookVersion is the default mutating/validating webhook config API version to scaffold.
const defaultWebhookVersion = "v1"

var _ plugin.CreateWebhookSubcommand = &createWebhookSubcommand{}

type createWebhookSubcommand struct {
	config config.Config
	// For help text.
	commandName string

	options *goPlugin.Options

	resource *resource.Resource

	// force indicates that the resource should be created even if it already exists
	force bool
}

func (p *createWebhookSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	p.commandName = cliMeta.CommandName

	subcmdMeta.Description = `Scaffold a webhook for an API resource. You can choose to scaffold defaulting,
validating and/or conversion webhooks.
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Create defaulting and validating webhooks for Group: ship, Version: v1beta1
  # and Kind: Frigate
  %[1]s create webhook --group ship --version v1beta1 --kind Frigate --defaulting --programmatic-validation

  # Create conversion webhook for Group: ship, Version: v1beta1
  # and Kind: Frigate
  %[1]s create webhook --group ship --version v1beta1 --kind Frigate --conversion
`, cliMeta.CommandName)
}

func (p *createWebhookSubcommand) BindFlags(fs *pflag.FlagSet) {
	p.options = &goPlugin.Options{}

	fs.StringVar(&p.options.Plural, "plural", "", "resource irregular plural form")

	fs.StringVar(&p.options.WebhookVersion, "webhook-version", defaultWebhookVersion,
		"version of {Mutating,Validating}WebhookConfigurations to scaffold. Options: [v1, v1beta1]")
	fs.StringVar(&p.options.Spoke, "spoke", "",
		"the spoke version for a conversion webhook")
	fs.BoolVar(&p.options.DoDefaulting, "defaulting", false,
		"if set, scaffold the defaulting webhook")
	fs.BoolVar(&p.options.DoValidation, "programmatic-validation", false,
		"if set, scaffold the validating webhook")
	fs.BoolVar(&p.options.DoConversion, "conversion", false,
		"if set, scaffold the conversion webhook")

	fs.BoolVar(&p.force, "force", false,
		"attempt to create resource even if it already exists")
}

func (p *createWebhookSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *createWebhookSubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res
	p.options.DoScaffold = true

	p.options.UpdateResource(p.resource, p.config)

	if err := p.resource.Validate(); err != nil {
		return err
	}

	// verify conversion webhook configuration
	if p.options.Spoke != "" {
		if err := p.verifyConversionWHConfig(); err != nil {
			return err
		}
	}

	// Either defaulting, validating or conversion falgs need to be present, or the spoke version can be specified
	if !p.resource.HasDefaultingWebhook() && !p.resource.HasValidationWebhook() && !p.resource.HasConversionWebhook() && p.options.Spoke == "" {
		return fmt.Errorf("%s create webhook requires at least one of --defaulting,"+
			" --programmatic-validation and --conversion to be true. If using conversion webhook"+
			"you can specify the spoke versions in the command.", p.commandName)
	}

	// check if resource exist to create webhook. Perform this check only when version
	// is to be scaffolded.
	if p.options.DoScaffold {
		if r, err := p.config.GetResource(p.resource.GVK); err != nil {
			return fmt.Errorf("%s create webhook requires a previously created API ", p.commandName)
		} else if r.Webhooks != nil && !r.Webhooks.IsEmpty() && !p.force {
			return fmt.Errorf("webhook resource already exists")
		}
	}

	if pluginutil.HasDifferentWebhookVersion(p.config, p.resource.Webhooks.WebhookVersion) {
		return fmt.Errorf("only one webhook version can be used for all resources, cannot add %q",
			p.resource.Webhooks.WebhookVersion)
	}

	return nil
}

func (p *createWebhookSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewWebhookScaffolder(p.config, *p.resource, p.force, p.options.DoScaffold, p.options.Spok)
	scaffolder.InjectFS(fs)
	return scaffolder.Scaffold()
}

// verifyConversionWHConfig verified the following configuration for
// conversion WHs:
// 1. checks if a hub has already been created before.
// 2. checks if spoke has already been scaffolded.
// 3. checks if API has been scaffolded for the spoke.
// If the hub has to be scaffolded, it sets the options.DoScaffold
// to true.
func (p *createWebhookSubcommand) verifyConversionWHConfig() error {
	hub, spokes, err := util.CategorizeHubAndSpokes(p.config, p.resource.GVK)
	if err != nil {
		return err
	}

	if hub != "" {
		// this also verifies the case where spoke is not a hub.
		// Currently, we are restricting the project to have single hub version.
		if hub != p.resource.Version {
			return fmt.Errorf("hub version %s found. Only one hub version is allowed", hub)
		}
		// Since the templates for hub version have already been scaffolded, setting this to
		// false.
		p.options.DoScaffold = false
	}

	for _, sp := range spokes {
		if sp == p.options.Spoke {
			return fmt.Errorf("spoke %s has already been scaffolded", p.options.Spoke)
		}
	}

	// check if api exists for spoke
	spokeGVK := p.resource.Copy().GVK
	spokeGVK.Version = p.resource.Version
	if !p.config.HasResource(spokeGVK) {
		return fmt.Errorf("api needs to exist for spoke %s to be scaffolded", p.options.Spoke)
	}
	return nil
}
