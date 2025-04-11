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
	"os"

	log "github.com/sirupsen/logrus"
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
	subcmdMeta.Description = `Scaffold a Kubernetes API by writing a Resource definition and/or a Controller.

If information about whether the resource and controller should be scaffolded
was not explicitly provided, it will prompt the user if they should be.

After the scaffold is written, the dependencies will be updated and
make generate will be run.
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
  %[1]s create api --group ship --version v1beta1 --kind Frigate

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
	fs.BoolVar(&p.runMake, "make", true, "if true, run `make generate` after generating files")

	fs.BoolVar(&p.force, "force", false,
		"attempt to create resource even if it already exists")

	p.options = &goPlugin.Options{}

	fs.StringVar(&p.options.Plural, "plural", "", "resource irregular plural form")

	fs.BoolVar(&p.options.DoAPI, "resource", true,
		"if set, generate the resource without prompting the user")
	p.resourceFlag = fs.Lookup("resource")
	fs.BoolVar(&p.options.Namespaced, "namespaced", true, "resource is namespaced")

	fs.BoolVar(&p.options.DoController, "controller", true,
		"if set, generate the controller without prompting the user")
	p.controllerFlag = fs.Lookup("controller")

	fs.StringVar(&p.options.ExternalAPIPath, "external-api-path", "",
		"Specify the Go package import path for the external API. This is used to scaffold controllers for resources "+
			"defined outside this project (e.g., github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1).")

	fs.StringVar(&p.options.ExternalAPIDomain, "external-api-domain", "",
		"Specify the domain name for the external API. This domain is used to generate accurate RBAC "+
			"markers and permissions for the external resources (e.g., cert-manager.io).")
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	reader := bufio.NewReader(os.Stdin)
	if !p.resourceFlag.Changed {
		log.Println("Create Resource [y/n]")
		p.options.DoAPI = util.YesNo(reader)
	}
	if !p.controllerFlag.Changed {
		log.Println("Create Controller [y/n]")
		p.options.DoController = util.YesNo(reader)
	}

	// Ensure that external API options cannot be used when creating an API in the project.
	if p.options.DoAPI {
		if len(p.options.ExternalAPIPath) != 0 || len(p.options.ExternalAPIDomain) != 0 {
			return errors.New("cannot use '--external-api-path' or '--external-api-domain' " +
				"when creating an API in the project with '--resource=true'. " +
				"Use '--resource=false' when referencing an external API")
		}
	}

	p.options.UpdateResource(p.resource, p.config)

	if err := p.resource.Validate(); err != nil {
		return fmt.Errorf("error validating resource: %w", err)
	}

	// In case we want to scaffold a resource API we need to do some checks
	if p.options.DoAPI {
		// Check that resource doesn't have the API scaffolded or flag force was set
		if r, err := p.config.GetResource(p.resource.GVK); err == nil && r.HasAPI() && !p.force {
			return errors.New("API resource already exists")
		}

		// Check that the provided group can be added to the project
		if !p.config.IsMultiGroup() && p.config.ResourcesLength() != 0 && !p.config.HasGroup(p.resource.Group) {
			return fmt.Errorf("multiple groups are not allowed by default, " +
				"to enable multi-group visit https://kubebuilder.io/migration/multi-group.html")
		}
	}

	return nil
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
