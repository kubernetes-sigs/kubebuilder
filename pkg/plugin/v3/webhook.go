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
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/pflag"
	"k8s.io/klog"

	"sigs.k8s.io/kubebuilder/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds"
)

type createWebhookPlugin struct {
	config *config.Config
	// For help text.
	commandName string

	resource   *resource.Options
	defaulting bool
	validation bool
	conversion bool
	spoke      string
	hub        string
}

var (
	_ plugin.CreateWebhook = &createWebhookPlugin{}
	_ cmdutil.RunOptions   = &createAPIPlugin{}
)

func (p *createWebhookPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Scaffold a webhook for an API resource. You can choose to scaffold defaulting,
validating and (or) conversion webhooks.
`
	ctx.Examples = fmt.Sprintf(`  # Create defaulting and validating webhooks for CRD of group crew, version v1
  # and kind FirstMate.
  %s create webhook --group crew --version v1 --kind FirstMate --defaulting --programmatic-validation

  # Create conversion webhook for CRD of group crew, version v1 and kind FirstMate.
  %s create webhook --group crew --version v1 --kind FirstMate --conversion
`,
		ctx.CommandName, ctx.CommandName)

	p.commandName = ctx.CommandName
}

func (p *createWebhookPlugin) BindFlags(fs *pflag.FlagSet) {
	p.resource = &resource.Options{}
	fs.StringVar(&p.resource.Group, "group", "", "resource Group")
	fs.StringVar(&p.resource.Version, "version", "", "resource Version")
	fs.StringVar(&p.resource.Kind, "kind", "", "resource Kind")
	fs.StringVar(&p.resource.Plural, "resource", "", "resource Resource")

	fs.BoolVar(&p.defaulting, "defaulting", false,
		"if set, scaffold the defaulting webhook")
	fs.BoolVar(&p.validation, "programmatic-validation", false,
		"if set, scaffold the validating webhook")
	fs.BoolVar(&p.conversion, "conversion", false,
		"if set, scaffold the conversion webhook")
	fs.StringVar(&p.spoke, "spoke", "", "provide a hub version for the conversion webhook")
	fs.StringVar(&p.hub, "hub", "", "provide the non-hub (spoke) verision for conversion webhook")
}

func (p *createWebhookPlugin) InjectConfig(c *config.Config) {
	p.config = c
}

func (p *createWebhookPlugin) Run() error {
	return cmdutil.Run(p)
}

func (p *createWebhookPlugin) Validate() error {
	if err := p.resource.Validate(); err != nil {
		return err
	}

	if !p.defaulting && !p.validation && !p.conversion {
		if p.hub == "" && p.spoke == "" {
			return fmt.Errorf("%s create webhook requires at least one of --defaulting,"+
				" --programmatic-validation, --conversion or --spoke and --hub to be specified", p.commandName)
		}
	}

	// check if resource exist to create webhook
	if !p.config.HasResource(p.resource.GVK()) {
		return fmt.Errorf("%s create webhook requires an api with the group,"+
			" kind and version provided", p.commandName)
	}

	// if conversion flag is set but hub and spoke version is not provided, suggest it to user and return
	if p.conversion && (p.spoke == "" && p.hub == "") {
		klog.V(2).Info("Hub and Spoke version can be specified in the command to scaffold Convertible and Hub interfaces")
		return nil
	}

	// set conversion to true if hub and spoke version are specified
	if p.hub != "" && p.spoke != "" {
		p.conversion = true
	}

	// validate the values provided for hub and spoke
	if p.conversion {
		if p.spoke == "" || p.hub == "" {
			return errors.New("Both Hub and Spoke version need to be specified when using conversion webhooks")
		}
		if p.spoke == p.hub {
			return fmt.Errorf("Cannot have the same hub and spoke version %s", p.hub)
		}
	}

	return nil
}

func (p *createWebhookPlugin) GetScaffolder() (scaffold.Scaffolder, error) {
	// Load the boilerplate
	bp, err := ioutil.ReadFile(filepath.Join("hack", "boilerplate.go.txt")) // nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("unable to load boilerplate: %v", err)
	}

	// Create the actual resource from the resource options
	res := p.resource.NewResource(p.config, false)
	return scaffolds.NewWebhookScaffolder(p.config, string(bp), res, p.defaulting, p.validation,
		p.conversion, p.spoke, p.hub), nil
}

func (p *createWebhookPlugin) PostScaffold() error {
	return nil
}
