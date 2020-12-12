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
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"
	goPlugin "sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/internal/util"
)

// defaultWebhookVersion is the default mutating/validating webhook config API version to scaffold.
const defaultWebhookVersion = "v1"

type createWebhookSubcommand struct {
	config *config.Config
	// For help text.
	commandName string

	options *goPlugin.Options

	// runMake indicates whether to run make or not after scaffolding webhooks
	runMake bool
}

var (
	_ plugin.CreateWebhookSubcommand = &createWebhookSubcommand{}
	_ cmdutil.RunOptions             = &createWebhookSubcommand{}
)

func (p *createWebhookSubcommand) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Scaffold a webhook for an API resource. You can choose to scaffold defaulting,
validating and (or) conversion webhooks.
`
	ctx.Examples = fmt.Sprintf(`  # Create defaulting and validating webhooks for CRD of group ship, version v1beta1
  # and kind Frigate.
  %s create webhook --group ship --version v1beta1 --kind Frigate --defaulting --programmatic-validation

  # Create conversion webhook for CRD of group ship, version v1beta1 and kind Frigate.
  %s create webhook --group ship --version v1beta1 --kind Frigate --conversion
`,
		ctx.CommandName, ctx.CommandName)

	p.commandName = ctx.CommandName
}

func (p *createWebhookSubcommand) BindFlags(fs *pflag.FlagSet) {
	p.options = &goPlugin.Options{}
	fs.StringVar(&p.options.Group, "group", "", "resource Group")
	// TODO: allow the user to specify the domain
	p.options.Domain = p.config.Domain
	fs.StringVar(&p.options.Version, "version", "", "resource Version")
	fs.StringVar(&p.options.Kind, "kind", "", "resource Kind")
	fs.StringVar(&p.options.Plural, "resource", "", "resource Resource")

	fs.StringVar(&p.options.WebhookVersion, "webhook-version", defaultWebhookVersion,
		"version of {Mutating,Validating}WebhookConfigurations to scaffold. Options: [v1, v1beta1]")
	fs.BoolVar(&p.options.DoDefaulting, "defaulting", false,
		"if set, scaffold the defaulting webhook")
	fs.BoolVar(&p.options.DoValidation, "programmatic-validation", false,
		"if set, scaffold the validating webhook")
	fs.BoolVar(&p.options.DoConversion, "conversion", false,
		"if set, scaffold the conversion webhook")

	fs.BoolVar(&p.runMake, "make", true, "if true, run make after generating files")
}

func (p *createWebhookSubcommand) InjectConfig(c *config.Config) {
	p.config = c
}

func (p *createWebhookSubcommand) Run() error {
	return cmdutil.Run(p)
}

func (p *createWebhookSubcommand) Validate() error {
	if err := p.options.Validate(); err != nil {
		return err
	}

	if !p.options.DoDefaulting && !p.options.DoValidation && !p.options.DoConversion {
		return fmt.Errorf("%s create webhook requires at least one of --defaulting,"+
			" --programmatic-validation and --conversion to be true", p.commandName)
	}

	// check if resource exist to create webhook
	if p.config.GetResource(p.options.GVK()) == nil {
		return fmt.Errorf("%s create webhook requires an api with the group,"+
			" kind and version provided", p.commandName)
	}

	if !p.config.IsWebhookVersionCompatible(p.options.WebhookVersion) {
		return fmt.Errorf("only one webhook version can be used for all resources, cannot add %q",
			p.options.WebhookVersion)
	}

	return nil
}

func (p *createWebhookSubcommand) GetScaffolder() (cmdutil.Scaffolder, error) {
	// Load the boilerplate
	bp, err := ioutil.ReadFile(filepath.Join("hack", "boilerplate.go.txt")) // nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("unable to load boilerplate: %v", err)
	}

	// Create the actual resource from the resource options
	res := p.options.NewResource(p.config)
	return scaffolds.NewWebhookScaffolder(p.config, string(bp), res), nil
}

func (p *createWebhookSubcommand) PostScaffold() error {
	if p.runMake {
		return util.RunCmd("Running make", "make")
	}
	return nil
}
