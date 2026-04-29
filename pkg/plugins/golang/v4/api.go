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
	"strings"

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

	// force indicates that the resource should be created even if it already exists
	force bool

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Create a Kubernetes API by writing a Resource definition and/or a Controller.

If information about whether the resource and controller should be created
was not explicitly provided, it will prompt the user if they should be.

After the files are created, the dependencies will be updated and
make generate will be run.
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
  %[1]s create api --group ship --version v1beta1 --kind Frigate

  # Add a controller for an existing resource (e.g., an additional reconciler)
  %[1]s create api --group ship --version v1beta1 --kind Frigate \
      --resource=false --controller-name frigate-status

  # Scaffold a controller for an external API (e.g., cert-manager)
  %[1]s create api --group certmanager --version v1 --kind Certificate \
      --resource=false --controller \
      --external-api-path github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \
      --external-api-domain cert-manager.io \
      --external-api-module github.com/cert-manager/cert-manager@v1.18.2

  # Edit the API Scheme
  nano api/v1beta1/frigate_types.go

  # Edit the Controller
  nano internal/controller/frigate/frigate_controller.go

  # Edit the Controller Test
  nano internal/controller/frigate/frigate_controller_test.go

  # Generate the manifests
  make manifests

  # Install CRDs into the Kubernetes cluster using kubectl apply
  make install

  # Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
  make run
`, cliMeta.CommandName)
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.runMake, "make", true,
		"If set, run 'make generate' after creating files. Default: true. "+
			"Use --make=false to skip running make generate")

	fs.BoolVar(&p.force, "force", false,
		"If set, attempt to create the resource even if it already exists")

	p.options = &goPlugin.Options{}

	fs.StringVar(&p.options.Plural, "plural", "",
		"Resource irregular plural form (e.g., 'people' for 'Person'); auto-detected if not provided")

	fs.BoolVar(&p.options.DoAPI, "resource", true,
		"If set, create the resource without prompting. Default: true. "+
			"Use --resource=false to skip resource creation (e.g., when adding a controller for an existing or external API)")
	p.resourceFlag = fs.Lookup("resource")
	fs.BoolVar(&p.options.Namespaced, "namespaced", true,
		"If set, create a namespaced resource. Default: true. "+
			"Use --namespaced=false to create a cluster-scoped resource")

	fs.BoolVar(&p.options.DoController, "controller", true,
		"If set, create the controller. Default: true. "+
			"Use --controller=false to skip controller creation")
	p.controllerFlag = fs.Lookup("controller")

	fs.StringVar(&p.options.ControllerName, "controller-name", "",
		"Name of the controller to scaffold (e.g., frigate-controller); allows multiple controllers per resource")

	fs.StringVar(&p.options.ExternalAPIPath, "external-api-path", "",
		"Go package import path for the external API (e.g., github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1). "+
			"Used to scaffold controllers for resources defined outside this project")

	fs.StringVar(&p.options.ExternalAPIDomain, "external-api-domain", "",
		"Domain name for the external API (e.g., cert-manager.io). "+
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

	reader := bufio.NewReader(os.Stdin)
	if !p.resourceFlag.Changed {
		log.Info("Create Resource [y/n]")
		p.options.DoAPI = util.YesNo(reader)
	}
	if !p.controllerFlag.Changed {
		log.Info("Create Controller [y/n]")
		p.options.DoController = util.YesNo(reader)
	}

	// When scaffolding a controller without an API (--resource=false), copy essential
	// fields from the existing resource in the PROJECT file, such as Path and Plural.
	// Note: API, Controllers, and Webhooks are managed separately by UpdateResource.
	if !p.options.DoAPI {
		if existingRes, err := p.config.GetResource(res.GVK); err == nil {
			p.resource.Path = existingRes.Path
			p.resource.Plural = existingRes.Plural
			p.resource.External = existingRes.External
			p.resource.Core = existingRes.Core
			p.resource.Module = existingRes.Module
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

	p.options.UpdateResource(p.resource, p.config)

	if err := p.resource.Validate(); err != nil {
		return fmt.Errorf("error validating resource: %w", err)
	}

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

	existingRes, err := p.config.GetResource(p.resource.GVK)
	if err != nil {
		// Resource does not exist yet, no validation needed
		return nil
	}

	// Require --controller-name when adding a controller to a resource that already has controllers.
	// Exception: if --resource=true (creating/recreating API), allow --controller=true without
	// a name for backward compatibility.
	if p.options.ControllerName == "" && !p.options.DoAPI &&
		existingRes.Controllers != nil && !existingRes.Controllers.IsEmpty() {
		return errors.New(
			"resource already has controllers defined; please specify '--controller-name' " +
				"to add another controller, or use '--controller=false' to skip controller scaffolding",
		)
	}

	// No controller name specified: using legacy mode, no further validation needed
	if p.options.ControllerName == "" {
		return nil
	}

	// Check that the controller name does not already exist
	if existingRes.Controllers != nil && existingRes.Controllers.HasController(p.options.ControllerName) {
		return fmt.Errorf(
			"controller with name %q already exists for resource %s/%s/%s",
			p.options.ControllerName,
			p.resource.Group,
			p.resource.Version,
			p.resource.Kind,
		)
	}

	// Check for name collisions after normalization (e.g., "foo-bar" vs "foobar")
	if existingRes.Controllers != nil {
		newNormalized := normalizeForCollisionCheck(p.options.ControllerName)
		for _, existing := range *existingRes.Controllers {
			if newNormalized == normalizeForCollisionCheck(existing.Name) {
				return fmt.Errorf(
					"controller name %q conflicts with existing controller %q: "+
						"both would generate the same reconciler struct name",
					p.options.ControllerName,
					existing.Name,
				)
			}
		}
	}

	// Also check legacy controller: true case
	if existingRes.Controller && p.options.ControllerName == strings.ToLower(p.resource.Kind) {
		return fmt.Errorf(
			"controller with name %q already exists for resource %s/%s/%s",
			p.options.ControllerName,
			p.resource.Group,
			p.resource.Version,
			p.resource.Kind,
		)
	}

	return nil
}

// normalizeForCollisionCheck normalizes a controller name to detect potential collisions.
// Different names that normalize to the same value would generate conflicting code.
func normalizeForCollisionCheck(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}
	return strings.ToLower(result.String())
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
