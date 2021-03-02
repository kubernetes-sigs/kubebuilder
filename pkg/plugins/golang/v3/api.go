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
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	goPlugin "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/util"
	"sigs.k8s.io/kubebuilder/v3/plugins/addon"
)

const (
	// KbDeclarativePatternVersion is the sigs.k8s.io/kubebuilder-declarative-pattern version
	// (used only to gen api with --pattern=addon)
	// TODO: remove this when a better solution for using addons is implemented.
	KbDeclarativePatternVersion = "b84d99da021778217217885dd9582ed3cc879ebe"

	// defaultCRDVersion is the default CRD API version to scaffold.
	defaultCRDVersion = "v1"
)

// DefaultMainPath is default file path of main.go
const DefaultMainPath = "main.go"

type createAPISubcommand struct {
	config config.Config

	// pattern indicates that we should use a plugin to build according to a pattern
	pattern string

	options *goPlugin.Options

	resource resource.Resource

	// Check if we have to scaffold resource and/or controller
	resourceFlag   *pflag.Flag
	controllerFlag *pflag.Flag

	// force indicates that the resource should be created even if it already exists
	force bool

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool
}

var (
	_ plugin.CreateAPISubcommand = &createAPISubcommand{}
	_ cmdutil.RunOptions         = &createAPISubcommand{}
)

func (p createAPISubcommand) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Scaffold a Kubernetes API by creating a Resource definition and / or a Controller.

create resource will prompt the user for if it should scaffold the Resource and / or Controller.  To only
scaffold a Controller for an existing Resource, select "n" for Resource.  To only define
the schema for a Resource without writing a Controller, select "n" for Controller.

After the scaffold is written, api will run make on the project.
`
	ctx.Examples = fmt.Sprintf(`  # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
  %s create api --group ship --version v1beta1 --kind Frigate

  # Edit the API Scheme
  nano api/v1beta1/frigate_types.go

  # Edit the Controller
  nano controllers/frigate/frigate_controller.go

  # Edit the Controller Test
  nano controllers/frigate/frigate_controller_test.go

  # Install CRDs into the Kubernetes cluster using kubectl apply
  make install

  # Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
  make run
	`,
		ctx.CommandName)
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.runMake, "make", true, "if true, run `make generate` after generating files")

	// TODO: remove this when a better solution for using addons is implemented.
	if os.Getenv("KUBEBUILDER_ENABLE_PLUGINS") != "" {
		fs.StringVar(&p.pattern, "pattern", "",
			"generates an API following an extension pattern (addon)")
	}

	fs.BoolVar(&p.force, "force", false,
		"attempt to create resource even if it already exists")

	p.options = &goPlugin.Options{}
	fs.StringVar(&p.options.Group, "group", "", "resource Group")
	p.options.Domain = p.config.GetDomain()
	fs.StringVar(&p.options.Version, "version", "", "resource Version")
	fs.StringVar(&p.options.Kind, "kind", "", "resource Kind")
	fs.StringVar(&p.options.Plural, "plural", "", "resource irregular plural form")

	fs.BoolVar(&p.options.DoAPI, "resource", true,
		"if set, generate the resource without prompting the user")
	p.resourceFlag = fs.Lookup("resource")
	fs.StringVar(&p.options.CRDVersion, "crd-version", defaultCRDVersion,
		"version of CustomResourceDefinition to scaffold. Options: [v1, v1beta1]")
	fs.BoolVar(&p.options.Namespaced, "namespaced", true, "resource is namespaced")

	fs.BoolVar(&p.options.DoController, "controller", true,
		"if set, generate the controller without prompting the user")
	p.controllerFlag = fs.Lookup("controller")
}

func (p *createAPISubcommand) InjectConfig(c config.Config) {
	p.config = c
}

func (p *createAPISubcommand) Run(fs afero.Fs) error {
	// TODO: re-evaluate whether y/n input still makes sense. We should probably always
	// scaffold the resource and controller.
	reader := bufio.NewReader(os.Stdin)
	if !p.resourceFlag.Changed {
		fmt.Println("Create Resource [y/n]")
		p.options.DoAPI = util.YesNo(reader)
	}
	if !p.controllerFlag.Changed {
		fmt.Println("Create Controller [y/n]")
		p.options.DoController = util.YesNo(reader)
	}

	// Create the resource from the options
	p.resource = p.options.NewResource(p.config)

	return cmdutil.Run(p, fs)
}

func (p *createAPISubcommand) Validate() error {
	if err := p.options.Validate(); err != nil {
		return err
	}

	if err := p.resource.Validate(); err != nil {
		return err
	}

	// check if main.go is present in the root directory
	if _, err := os.Stat(DefaultMainPath); os.IsNotExist(err) {
		return fmt.Errorf("%s file should present in the root directory", DefaultMainPath)
	}

	// In case we want to scaffold a resource API we need to do some checks
	if p.options.DoAPI {
		// Check that resource doesn't have the API scaffolded or flag force was set
		if res, err := p.config.GetResource(p.resource.GVK); err == nil && res.HasAPI() && !p.force {
			return errors.New("API resource already exists")
		}

		// Check that the provided group can be added to the project
		if !p.config.IsMultiGroup() && p.config.ResourcesLength() != 0 && !p.config.HasGroup(p.resource.Group) {
			return fmt.Errorf("multiple groups are not allowed by default, " +
				"to enable multi-group visit https://kubebuilder.io/migration/multi-group.html")
		}

		// Check CRDVersion against all other CRDVersions in p.config for compatibility.
		if pluginutil.HasDifferentCRDVersion(p.config, p.resource.API.CRDVersion) {
			return fmt.Errorf("only one CRD version can be used for all resources, cannot add %q",
				p.resource.API.CRDVersion)
		}
	}

	return nil
}

func (p *createAPISubcommand) GetScaffolder() (cmdutil.Scaffolder, error) {
	// Load the requested plugins
	plugins := make([]model.Plugin, 0)
	switch strings.ToLower(p.pattern) {
	case "":
		// Default pattern
	case "addon":
		plugins = append(plugins, &addon.Plugin{})
	default:
		return nil, fmt.Errorf("unknown pattern %q", p.pattern)
	}

	return scaffolds.NewAPIScaffolder(p.config, p.resource, p.force, plugins), nil
}

func (p *createAPISubcommand) PostScaffold() error {
	// Load the requested plugins
	switch strings.ToLower(p.pattern) {
	case "":
		// Default pattern
	case "addon":
		// Ensure that we are pinning sigs.k8s.io/kubebuilder-declarative-pattern version
		// TODO: either find a better way to inject this version (ex. tools.go).
		err := util.RunCmd("Get kubebuilder-declarative-pattern dependency", "go", "get",
			"sigs.k8s.io/kubebuilder-declarative-pattern@"+KbDeclarativePatternVersion)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown pattern %q", p.pattern)
	}

	err := util.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return err
	}

	if p.runMake { // TODO: check if API was scaffolded
		return util.RunCmd("Running make", "make", "generate")
	}
	return nil
}
