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
	log "log/slog"
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

  # Create defaulting webhook with custom path for Group: ship, Version: v1beta1
  # and Kind: Frigate
  %[1]s create webhook --group ship --version v1beta1 --kind Frigate --defaulting \
    --defaulting-path=/my-custom-mutate-path

  # Create validation webhook with custom path for Group: ship, Version: v1beta1
  # and Kind: Frigate
  %[1]s create webhook --group ship --version v1beta1 --kind Frigate \
    --programmatic-validation --validation-path=/my-custom-validate-path

  # Create both defaulting and validation webhooks with different custom paths
  %[1]s create webhook --group ship --version v1beta1 --kind Frigate \
    --defaulting --programmatic-validation \
    --defaulting-path=/custom-mutate --validation-path=/custom-validate
`, cliMeta.CommandName)
}

func (p *createWebhookSubcommand) BindFlags(fs *pflag.FlagSet) {
	p.options = &goPlugin.Options{}

	fs.BoolVar(&p.runMake, "make", true,
		"Run 'make generate' after generating files (enabled by default; use --make=false to disable)")

	fs.StringVar(&p.options.Plural, "plural", "",
		"Resource irregular plural form (e.g., 'people' for 'Person'); auto-detected from resource kind if not provided")

	fs.BoolVar(&p.options.DoDefaulting, "defaulting", false,
		"If set, scaffold the defaulting webhook")
	fs.BoolVar(&p.options.DoValidation, "programmatic-validation", false,
		"If set, scaffold the validating webhook")
	fs.BoolVar(&p.options.DoConversion, "conversion", false,
		"If set, scaffold the conversion webhook")

	fs.StringSliceVar(&p.options.Spoke, "spoke",
		nil,
		"Comma-separated list of spoke versions to be added to the conversion webhook (e.g., --spoke v1,v2)")

	fs.StringVar(&p.options.DefaultingPath, "defaulting-path", "",
		"[Optional] Custom path for the defaulting/mutating webhook (e.g., /my-custom-mutate-path). "+
			"Only valid with --defaulting")

	fs.StringVar(&p.options.ValidationPath, "validation-path", "",
		"[Optional] Custom path for the validation webhook (e.g., /my-custom-validate-path). "+
			"Only valid with --programmatic-validation")

	fs.StringVar(&p.options.ExternalAPIPath, "external-api-path", "",
		"Go package import path for the external API (e.g., github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1). "+
			"Used to scaffold webhooks for resources defined outside this project")

	fs.StringVar(&p.options.ExternalAPIDomain, "external-api-domain", "",
		"Domain for the external API (e.g., cert-manager.io). Selects the recorded resource when "+
			"several share a group, version, and kind, and is used to generate accurate RBAC markers "+
			"and permissions for the external resources")

	fs.StringVar(&p.options.ExternalAPIModule, "external-api-module", "",
		"External API module with optional version (e.g., github.com/cert-manager/cert-manager@v1.18.2)")

	fs.BoolVar(&p.force, "force", false,
		"If set, attempt to create resource even if it already exists")
}

func (p *createWebhookSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *createWebhookSubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	if err := p.updateResourceFromConfig(res); err != nil {
		return err
	}

	for _, spoke := range p.options.Spoke {
		spoke = strings.TrimSpace(spoke)
		if !isValidVersion(spoke, res, p.config) {
			return fmt.Errorf("invalid spoke version %q", spoke)
		}
		res.Webhooks.Spoke = append(res.Webhooks.Spoke, spoke)
	}

	// Validate path flags are only used with appropriate webhook types
	if p.options.DefaultingPath != "" && !p.options.DoDefaulting {
		return fmt.Errorf("--defaulting-path can only be used with --defaulting")
	}
	if p.options.ValidationPath != "" && !p.options.DoValidation {
		return fmt.Errorf("--validation-path can only be used with --programmatic-validation")
	}

	// Validate that --external-api-module requires --external-api-path
	if len(p.options.ExternalAPIModule) != 0 && len(p.options.ExternalAPIPath) == 0 {
		return errors.New("'--external-api-module' requires '--external-api-path' to be specified")
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
			return fmt.Errorf(
				"no API found for %s/%s, Kind %s: run 'create api' first, "+
					"or pass --external-api-path for an external type",
				p.resource.QualifiedGroup(),
				p.resource.Version,
				p.resource.Kind,
			)
		}
	} else if res.Webhooks != nil && !res.Webhooks.IsEmpty() && !p.force {
		// Check if user is trying to add a webhook type that already exists
		if p.resource.HasDefaultingWebhook() && res.Webhooks.Defaulting {
			return fmt.Errorf("defaulting webhook already exists for this resource")
		}
		if p.resource.HasValidationWebhook() && res.Webhooks.Validation {
			return fmt.Errorf("validation webhook already exists for this resource")
		}
		if p.resource.HasConversionWebhook() && res.Webhooks.Conversion {
			return fmt.Errorf("conversion webhook already exists for this resource")
		}
		// If we're here, user is adding a new webhook type to existing resource
		// Merge the webhook configurations
		if err := p.resource.Webhooks.Update(res.Webhooks); err != nil {
			return fmt.Errorf("error merging webhook configurations: %w", err)
		}
	}

	return nil
}

func (p *createWebhookSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewWebhookScaffolder(p.config, *p.resource, p.force)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold webhook: %w", err)
	}

	return nil
}

func (p *createWebhookSubcommand) PostScaffold() error {
	// If external API with module specified, add it using go get
	if p.resource.IsExternal() && p.resource.Module != "" {
		log.Info("Adding external API dependency", "module", p.resource.Module)
		// Use go get to add the dependency cleanly as a direct requirement
		err := pluginutil.RunCmd("Add external API dependency", "go", "get", p.resource.Module)
		if err != nil {
			return fmt.Errorf("error adding external API dependency: %w", err)
		}
	}

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

// updateResourceFromConfig fills res with the configuration recorded for its
// Group/Version/Kind in the PROJECT file: Domain, Path, Plural, External, Core and Module.
//
// The lookup is by Group/Version/Kind rather than the full GVK because res still carries the
// project domain, while an external resource keeps its own. Only external APIs can register the
// same Group/Version/Kind under different domains, so when several match --external-api-domain
// selects the intended one; without it resolution falls to the non-external (core/project) entry.
func (p *createWebhookSubcommand) updateResourceFromConfig(res *resource.Resource) error {
	resources, err := p.config.GetResources()
	if err != nil {
		return fmt.Errorf("failed to load resources from project configuration: %w", err)
	}

	// Collect every recorded resource sharing this Group/Version/Kind.
	var candidates []resource.Resource
	for _, r := range resources {
		if r.Group == res.Group && r.Version == res.Version && r.Kind == res.Kind {
			candidates = append(candidates, r)
		}
	}

	domain := p.options.ExternalAPIDomain
	var selected *resource.Resource
	switch len(candidates) {
	case 0:
		return nil // nothing recorded for this GVK; keep res as built from the flags
	case 1:
		if domain == "" {
			selected = &candidates[0] // recover the single record
		} else {
			selected, err = selectByDomain(candidates, domain, p.options.ExternalAPIPath)
		}
	default:
		if domain == "" {
			selected, err = selectNonExternal(candidates, res.Domain)
		} else {
			selected, err = selectByDomain(candidates, domain, p.options.ExternalAPIPath)
		}
	}
	if err != nil {
		return err
	}
	if selected == nil {
		return nil // the flags describe a new resource
	}

	res.Domain = selected.Domain
	res.Path = selected.Path
	res.Plural = selected.Plural
	res.External = selected.External
	res.Core = selected.Core
	res.Module = selected.Module

	return nil
}

// selectByDomain resolves the candidate carrying the given --external-api-domain, for any number
// of candidates. When none carries it, a non-empty external path means the flags describe a new
// resource (nil, nil); otherwise the request is refused.
func selectByDomain(candidates []resource.Resource, domain, externalPath string) (*resource.Resource, error) {
	for i := range candidates {
		if candidates[i].Domain == domain {
			return &candidates[i], nil
		}
	}
	if externalPath != "" {
		return nil, nil
	}
	return nil, resolutionError(candidates, domain)
}

// selectNonExternal resolves several same-GVK candidates when no domain is given: the target is
// the non-external (core/project) entry. When a core and project entry collide it keeps the
// project (its domain equals projectDomain); when every candidate is external it refuses.
func selectNonExternal(candidates []resource.Resource, projectDomain string) (*resource.Resource, error) {
	var nonExternal []resource.Resource
	for _, c := range candidates {
		if !c.External {
			nonExternal = append(nonExternal, c)
		}
	}

	switch len(nonExternal) {
	case 0:
		return nil, resolutionError(candidates, "")
	case 1:
		return &nonExternal[0], nil
	default:
		// A core and a project resource share the GVK; keep the project one.
		for i := range nonExternal {
			if nonExternal[i].Domain == projectDomain {
				return &nonExternal[i], nil
			}
		}
		return nil, resolutionError(candidates, "")
	}
}

// resolutionError reports why the candidates could not be resolved to one, naming their domains.
// With a named domain it reports that none matched; without one it asks for --external-api-domain.
func resolutionError(candidates []resource.Resource, domain string) error {
	g, v, k := candidates[0].Group, candidates[0].Version, candidates[0].Kind
	domains := make([]string, len(candidates))
	for i, c := range candidates {
		domains[i] = c.Domain
	}

	if domain != "" {
		return fmt.Errorf(
			"no resource matches --external-api-domain %q for group %q, version %q and kind %q "+
				"(recorded domains: %s)",
			domain, g, v, k, strings.Join(domains, ", "),
		)
	}
	return fmt.Errorf(
		"group %q, version %q and kind %q match more than one resource (domains: %s): "+
			"pass --external-api-domain to choose the one to work on",
		g, v, k, strings.Join(domains, ", "),
	)
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
