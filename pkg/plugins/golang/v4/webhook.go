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

package v4

import (
	"fmt"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds"
)

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

	p.options.UpdateResource(p.resource, p.config)

	if err := p.resource.Validate(); err != nil {
		return err
	}

	if !p.resource.HasDefaultingWebhook() && !p.resource.HasValidationWebhook() && !p.resource.HasConversionWebhook() {
		return fmt.Errorf("%s create webhook requires at least one of --defaulting,"+
			" --programmatic-validation and --conversion to be true", p.commandName)
	}

	// check if resource exist to create webhook
	if r, err := p.config.GetResource(p.resource.GVK); err != nil {
		return fmt.Errorf("%s create webhook requires a previously created API ", p.commandName)
	} else if r.Webhooks != nil && !r.Webhooks.IsEmpty() && !p.force {
		return fmt.Errorf("webhook resource already exists")
	}

	return nil
}

func (p *createWebhookSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewWebhookScaffolder(p.config, *p.resource, p.force)
	scaffolder.InjectFS(fs)
	return scaffolder.Scaffold()
}

func (p *createWebhookSubcommand) PostScaffold() error {
	err := pluginutil.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return err
	}

	err = pluginutil.RunCmd("Running make", "make", "generate")
	if err != nil {
		return err
	}
	fmt.Print("Next: implement your new Webhook and generate the manifests with:\n$ make manifests\n")

	return nil
}
