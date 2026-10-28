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
	"bufio"
	"errors"
	"fmt"
	log "log/slog"
	"os"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds"
)

// DefaultMainPath is default file path of main.go
const DefaultMainPath = "cmd/main.go"

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config config.Config

	options *goPlugin.Options

	resource *resource.Resource

	// Check if we have to scaffold resource and/or controller
	resourceFlag   *pflag.Flag
	controllerFlag *pflag.Flag
	ssaFlag        *pflag.Flag

	// force indicates that the resource should be created even if it already exists
	force bool

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Scaffold a Kubernetes API by writing a Resource definition and/or a Controller.

If --resource or --controller is not explicitly set, Kubebuilder prompts for what to scaffold.

After writing the scaffold, Kubebuilder updates dependencies. When an API resource is scaffolded,
Kubebuilder runs make generate unless --make=false is set.
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Create a namespaced API resource and controller
  %[1]s create api --group crew --version v1 --kind Captain

  # Create a cluster-scoped API resource without a controller
  %[1]s create api --group crew --version v1 --kind Admiral --namespaced=false --controller=false

  # Create an API resource scaffolded with Server-Side Apply support (alpha)
  %[1]s create api --group crew --version v1 --kind Captain --ssa

  # Create a controller for an external API type
  %[1]s create api --group cert-manager --version v1 --kind Certificate \
    --resource=false --controller=true \
    --external-api-path github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1
`, cliMeta.CommandName)
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.runMake, "make", true,
		"Run 'make generate' after generating files (enabled by default; use --make=false to disable)")

	fs.BoolVar(&p.force, "force", false,
		"If set, attempt to create resource even if it already exists")

	p.options = &goPlugin.Options{}

	fs.StringVar(&p.options.Plural, "plural", "",
		"Resource irregular plural form (e.g., 'people' for 'Person'); auto-detected if not provided")

	fs.BoolVar(&p.options.DoAPI, "resource", true,
		"Generate the resource without prompting the user (enabled by default; use --resource=false to disable)")
	p.resourceFlag = fs.Lookup("resource")
	fs.BoolVar(&p.options.Namespaced, "namespaced", true,
		"Resource is namespaced by default; use --namespaced=false to create a cluster-scoped resource")

	fs.BoolVar(&p.options.SSA, "ssa", false,
		"(ALPHA) If set, scaffold this API with Server-Side Apply support "+
			"(adds the +genclient and +kubebuilder:ac:generate markers and updates the Makefile manifests target). "+
			"Alpha feature: may change in future releases")
	p.ssaFlag = fs.Lookup("ssa")

	fs.BoolVar(&p.options.DoController, "controller", true,
		"Prompt whether to generate the controller by default; "+
			"use --controller=true or --controller=false to skip the prompt")
	p.controllerFlag = fs.Lookup("controller")

	fs.StringVar(&p.options.ControllerName, "controller-name", "",
		"Name of the controller to scaffold (e.g., frigate-controller); defaults to the lowercase kind if unset; "+
			"allows multiple controllers per resource")

	fs.StringVar(&p.options.ExternalAPIPath, "external-api-path", "",
		"Go package import path for the external API (e.g., github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1). "+
			"Used to scaffold controllers for resources defined outside this project")

	fs.StringVar(&p.options.ExternalAPIDomain, "external-api-domain", "",
		"Domain suffix for the external API, combined with --group to form the qualified group "+
			"(e.g., --group cert-manager --external-api-domain io => cert-manager.io). "+
			"Used to generate accurate RBAC markers and permissions for the external resources")

	fs.StringVar(&p.options.ExternalAPIModule, "external-api-module", "",
		"External API module with optional version (e.g., github.com/cert-manager/cert-manager@v1.18.2)")
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	// Server-Side Apply can only be turned on, so reject an explicit --ssa=false.
	if p.ssaFlag != nil && p.ssaFlag.Changed && !p.options.SSA {
		return errors.New("'--ssa=false' is not supported: Server-Side Apply can only be turned on. " +
			"Omit '--ssa' to scaffold without it")
	}

	reader := bufio.NewReader(os.Stdin)
	if !p.resourceFlag.Changed {
		log.Info("Create Resource [y/n]")
		p.options.DoAPI = util.YesNo(reader)
	}
	if !p.controllerFlag.Changed {
		log.Info("Create Controller [y/n]")
		p.options.DoController = util.YesNo(reader)
	}

	if !p.options.DoAPI {
		// When scaffolding a controller without an API (--resource=false), copy essential
		// fields from the existing resource in the PROJECT file, such as Domain, Path and Plural.
		// Several recorded resources can share the Group/Version/Kind under different domains,
		// so resolve the target the same way create webhook does instead of by the full GVK.
		// Note: API, Controllers, and Webhooks are managed separately by UpdateResource.
		existingRes, err := findRecordedResource(p.config, res, p.options.ExternalAPIDomain, p.options.ExternalAPIPath)
		if err != nil {
			return err
		}
		if existingRes != nil {
			p.resource.Domain = existingRes.Domain
			p.resource.Path = existingRes.Path
			p.resource.Plural = existingRes.Plural
			p.resource.External = existingRes.External
			p.resource.Core = existingRes.Core
			p.resource.Module = existingRes.Module
		}
	} else if existingRes, err := p.config.GetResource(res.GVK); err == nil {
		if existingRes.API != nil && existingRes.API.SSA {
			// SSA cannot be disabled, so keep the value tracked in the PROJECT file.
			p.options.SSA = true
		} else if p.force && p.options.SSA && existingRes.HasAPI() {
			// SSA is chosen when the API is first created, so a forced re-run cannot add it.
			return fmt.Errorf("unable to add '--ssa' to the existing API %s/%s, Kind=%s: "+
				"Server-Side Apply can only be enabled when the API is first created",
				res.Group, res.Version, res.Kind)
		}
	}

	// Ensure that external API options cannot be used when creating an API in the project.
	if p.options.DoAPI &&
		(len(p.options.ExternalAPIPath) != 0 ||
			len(p.options.ExternalAPIDomain) != 0 ||
			len(p.options.ExternalAPIModule) != 0) {
		return errors.New(
			"cannot use '--external-api-path', '--external-api-domain', or '--external-api-module' " +
				"when creating an API in the project with '--resource=true'. " +
				"Use '--resource=false' when referencing an external API",
		)
	}

	// Validate that --external-api-module requires --external-api-path
	if len(p.options.ExternalAPIModule) != 0 && len(p.options.ExternalAPIPath) == 0 {
		return errors.New("'--external-api-module' requires '--external-api-path' to be specified")
	}

	// Validate that --ssa requires --resource=true
	if p.options.SSA && !p.options.DoAPI {
		return errors.New("'--ssa' can only be used when creating an API resource ('--resource=true')")
	}

	// UpdateResource cannot report an invalid controller name, so validate it first.
	if p.options.DoController && p.options.ControllerName != "" {
		if err := (resource.Controller{Name: p.options.ControllerName}).Validate(); err != nil {
			return fmt.Errorf("invalid '--controller-name': %w", err)
		}
	}

	p.options.UpdateResource(p.resource, p.config)

	if err := p.resource.Validate(); err != nil {
		return fmt.Errorf("error validating resource: %w", err)
	}

	// Options may change the external API domain, so validate the final GVK.
	if err := p.validateAPI(); err != nil {
		return err
	}
	return p.validateController()
}

func (p *createAPISubcommand) validateAPI() error {
	if !p.options.DoAPI {
		return nil
	}

	// Check that resource doesn't have the API scaffolded or flag force was set
	if r, err := p.config.GetResource(p.resource.GVK); err == nil && r.HasAPI() && !p.force {
		return errors.New("API resource already exists")
	}

	// Check that the provided group can be added to the project
	if !p.config.IsMultiGroup() && p.config.ResourcesLength() != 0 && !p.config.HasGroup(p.resource.Group) {
		return fmt.Errorf(
			"multiple groups are not allowed by default, " +
				"to enable multi-group visit https://kubebuilder.io/migration/multi-group.html",
		)
	}

	return nil
}

func (p *createAPISubcommand) validateController() error {
	if !p.options.DoController {
		return nil
	}

	if err := p.validateControllerAcrossResources(); err != nil {
		return err
	}

	existingRes, err := p.config.GetResource(p.resource.GVK)
	if err != nil {
		// Resource does not exist yet, no validation needed
		return nil
	}

	existing := existingRes.GetControllerNames()
	if len(existing) == 0 {
		return nil
	}

	if p.options.ControllerName == "" {
		// Allow unnamed controllers when recreating an API.
		if p.options.DoAPI {
			return nil
		}
		// An unnamed request selects the single default controller.
		if len(existing) == 1 && existing[0] == resource.DefaultControllerName(p.resource.Kind) {
			return nil
		}
		return errors.New(
			"resource already has controllers defined; please specify '--controller-name' " +
				"to add another controller, or use '--controller=false' to skip controller scaffolding",
		)
	}

	newReconciler := resource.NormalizeReconcilerName(p.options.ControllerName, p.resource.Kind)
	for _, name := range existing {
		if name == p.options.ControllerName {
			return fmt.Errorf(
				"controller with name %q already exists for resource %s/%s/%s",
				p.options.ControllerName,
				p.resource.Group,
				p.resource.Version,
				p.resource.Kind,
			)
		}

		if resource.NormalizeReconcilerName(name, p.resource.Kind) == newReconciler {
			return fmt.Errorf(
				"controller name %q conflicts with existing controller %q: both generate %s",
				p.options.ControllerName,
				name,
				newReconciler,
			)
		}
	}

	return nil
}

// validateControllerAcrossResources rejects duplicate reconciler names and filenames
// in the same controller package, including default controller names.
func (p *createAPISubcommand) validateControllerAcrossResources() error {
	resources, err := p.config.GetResources()
	if err != nil {
		return nil
	}

	newName := p.options.ControllerName
	if newName == "" {
		newName = resource.DefaultControllerName(p.resource.Kind)
	}
	newReconciler := resource.NormalizeReconcilerName(newName, p.resource.Kind)

	for _, res := range resources {
		if res.IsEqualTo(p.resource.GVK) || !p.sharesControllerPackage(res) {
			continue
		}

		for _, name := range res.GetControllerNames() {
			if resource.NormalizeReconcilerName(name, res.Kind) != newReconciler {
				if name == newName {
					return fmt.Errorf("controller name %q conflicts with controller of %s/%s/%s: "+
						"both generate %s_controller.go in the same package; use '--controller-name' to pick another name",
						newName, res.Group, res.Version, res.Kind, resource.NormalizeFileName(newName))
				}
				continue
			}

			return fmt.Errorf(
				"controller %q conflicts with controller %q of %s/%s/%s: "+
					"both generate %s in the same package; use '--controller-name' to pick "+
					"another name, or '--controller=false' if that controller already reconciles this resource",
				newName, name, res.Group, res.Version, res.Kind, newReconciler,
			)
		}
	}

	return nil
}

// sharesControllerPackage reports whether resources use the same controller directory.
func (p *createAPISubcommand) sharesControllerPackage(other resource.Resource) bool {
	return !p.config.IsMultiGroup() || p.resource.Group == other.Group
}

func (p *createAPISubcommand) PreScaffold(machinery.Filesystem) error {
	// check if main.go is present in the root directory
	if _, err := os.Stat(DefaultMainPath); os.IsNotExist(err) {
		return fmt.Errorf("%s file should present in the root directory", DefaultMainPath)
	}

	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewAPIScaffolder(p.config, *p.resource, p.force)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding API: %w", err)
	}

	return nil
}

func (p *createAPISubcommand) PostScaffold() error {
	// If external API with module specified, add it using go get
	if p.resource.IsExternal() && p.resource.Module != "" {
		log.Info("Adding external API dependency", "module", p.resource.Module)
		// Use go get to add the dependency cleanly as a direct requirement
		err := util.RunCmd("Add external API dependency", "go", "get", p.resource.Module)
		if err != nil {
			return fmt.Errorf("error adding external API dependency: %w", err)
		}
	}

	err := util.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return fmt.Errorf("error updating go dependencies: %w", err)
	}
	if p.runMake && p.resource.HasAPI() {
		err = util.RunCmd("Running make", "make", "generate")
		if err != nil {
			return fmt.Errorf("error running make generate: %w", err)
		}
		fmt.Print("Next: implement your new API and generate the manifests (e.g. CRDs,CRs) with:\n$ make manifests\n")
	}

	return nil
}
