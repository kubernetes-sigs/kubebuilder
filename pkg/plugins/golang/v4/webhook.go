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
	"errors"
	"fmt"
	"strings"

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

	// Deprecated - TODO: remove it for go/v5
	// isLegacyPath indicates that the resource should be created in the legacy path under the api
	isLegacyPath bool

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool
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
  %[1]s create webhook --group ship --version v1beta1 --kind Frigate --conversion --spoke v1
`, cliMeta.CommandName)
}

func (p *createWebhookSubcommand) BindFlags(fs *pflag.FlagSet) {
	p.options = &goPlugin.Options{}

	fs.BoolVar(&p.runMake, "make", true, "if true, run `make generate` after generating files")

	fs.StringVar(&p.options.Plural, "plural", "", "resource irregular plural form")

	fs.BoolVar(&p.options.DoDefaulting, "defaulting", false,
		"if set, scaffold the defaulting webhook")
	fs.BoolVar(&p.options.DoValidation, "programmatic-validation", false,
		"if set, scaffold the validating webhook")
	fs.BoolVar(&p.options.DoConversion, "conversion", false,
		"if set, scaffold the conversion webhook")

	fs.StringSliceVar(&p.options.Spoke, "spoke",
		nil,
		"Comma-separated list of spoke versions to be added to the conversion webhook (e.g., --spoke v1,v2)")

	// TODO: remove for go/v5
	fs.BoolVar(&p.isLegacyPath, "legacy", false,
		"[DEPRECATED] Attempts to create resource under the API directory (legacy path). "+
			"This option will be removed in future versions.")

	fs.StringVar(&p.options.ExternalAPIPath, "external-api-path", "",
		"Specify the Go package import path for the external API. This is used to scaffold controllers for resources "+
			"defined outside this project (e.g., github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1).")

	fs.StringVar(&p.options.ExternalAPIDomain, "external-api-domain", "",
		"Specify the domain name for the external API. This domain is used to generate accurate RBAC "+
			"markers and permissions for the external resources (e.g., cert-manager.io).")

	fs.BoolVar(&p.force, "force", false,
		"attempt to create resource even if it already exists")
}

func (p *createWebhookSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *createWebhookSubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	if len(p.options.ExternalAPIPath) != 0 && len(p.options.ExternalAPIDomain) != 0 && p.isLegacyPath {
		return errors.New("you cannot scaffold webhooks for external types using the legacy path")
	}

	for _, spoke := range p.options.Spoke {
		spoke = strings.TrimSpace(spoke)
		if !isValidVersion(spoke, res, p.config) {
			return fmt.Errorf("invalid spoke version %q", spoke)
		}
		res.Webhooks.Spoke = append(res.Webhooks.Spoke, spoke)
	}

	p.options.UpdateResource(p.resource, p.config)

	if err := p.resource.Validate(); err != nil {
		return fmt.Errorf("error validating resource: %w", err)
	}

	if !p.resource.HasDefaultingWebhook() && !p.resource.HasValidationWebhook() && !p.resource.HasConversionWebhook() {
		return fmt.Errorf("%s create webhook requires at least one of --defaulting,"+
			" --programmatic-validation and --conversion to be true", p.commandName)
	}

	// check if resource exist to create webhook
	resValue, err := p.config.GetResource(p.resource.GVK)
	res = &resValue
	if err != nil {
		if !p.resource.External && !p.resource.Core {
			return fmt.Errorf("%s create webhook requires a previously created API ", p.commandName)
		}
	} else if res.Webhooks != nil && !res.Webhooks.IsEmpty() && !p.force {
		// FIXME: This is a temporary fix to allow we move forward
		// However, users should be able to call the command to create an webhook
		// even if the resource already has one when the webhook is not of the same type.
		return fmt.Errorf("webhook resource already exists")
	}

	return nil
}

func (p *createWebhookSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewWebhookScaffolder(p.config, *p.resource, p.force, p.isLegacyPath)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold webhook: %w", err)
	}

	return nil
}

func (p *createWebhookSubcommand) PostScaffold() error {
	err := pluginutil.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return fmt.Errorf("error updating go dependencies: %w", err)
	}

	if p.runMake {
		err = pluginutil.RunCmd("Running make", "make", "generate")
		if err != nil {
			return fmt.Errorf("error running make generate: %w", err)
		}
	}

	fmt.Print("Next: implement your new Webhook and generate the manifests with:\n$ make manifests\n")

	return nil
}

// Helper function to validate spoke versions
func isValidVersion(version string, res *resource.Resource, cfg config.Config) bool {
	// Fetch all resources in the config
	resources, err := cfg.GetResources()
	if err != nil {
		return false
	}

	// Iterate through resources and validate if the given version exists for the same Group and Kind
	for _, r := range resources {
		if r.Group == res.Group && r.Kind == res.Kind && r.Version == version {
			return true
		}
	}

	// If no matching version is found, return false
	return false
}
