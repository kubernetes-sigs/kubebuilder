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

package v2

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	"sigs.k8s.io/kubebuilder/pkg/plugin/internal"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/plugins/addon"
)

type createAPIPlugin struct {
	config *config.Config

	// pattern indicates that we should use a plugin to build according to a pattern
	pattern string

	resource *resource.Options

	// Check if we have to scaffold resource and/or controller
	resourceFlag   *pflag.Flag
	controllerFlag *pflag.Flag
	doResource     bool
	doController   bool

	// force indicates that the resource should be created even if it already exists
	force bool

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool
}

var (
	_ plugin.CreateAPI   = &createAPIPlugin{}
	_ cmdutil.RunOptions = &createAPIPlugin{}
)

func (p createAPIPlugin) UpdateContext(ctx *plugin.Context) {
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

func (p *createAPIPlugin) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.runMake, "make", true, "if true, run make after generating files")

	fs.BoolVar(&p.doResource, "resource", true,
		"if set, generate the resource without prompting the user")
	p.resourceFlag = fs.Lookup("resource")
	fs.BoolVar(&p.doController, "controller", true,
		"if set, generate the controller without prompting the user")
	p.controllerFlag = fs.Lookup("controller")

	if os.Getenv("KUBEBUILDER_ENABLE_PLUGINS") != "" {
		fs.StringVar(&p.pattern, "pattern", "",
			"generates an API following an extension pattern (addon)")
	}

	fs.BoolVar(&p.force, "force", false,
		"attempt to create resource even if it already exists")
	p.resource = &resource.Options{}
	fs.StringVar(&p.resource.Kind, "kind", "", "resource Kind")
	fs.StringVar(&p.resource.Group, "group", "", "resource Group")
	fs.StringVar(&p.resource.Version, "version", "", "resource Version")
	fs.BoolVar(&p.resource.Namespaced, "namespaced", true, "resource is namespaced")
}

func (p *createAPIPlugin) InjectConfig(c *config.Config) {
	p.config = c
}

func (p *createAPIPlugin) Run() error {
	return cmdutil.Run(p)
}

func (p *createAPIPlugin) Validate() error {
	if err := p.resource.Validate(); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	if !p.resourceFlag.Changed {
		fmt.Println("Create Resource [y/n]")
		p.doResource = internal.YesNo(reader)
	}
	if !p.controllerFlag.Changed {
		fmt.Println("Create Controller [y/n]")
		p.doController = internal.YesNo(reader)
	}

	// In case we want to scaffold a resource API we need to do some checks
	if p.doResource {
		// Check that resource doesn't exist or flag force was set
		if !p.force && p.config.HasResource(p.resource.GVK()) {
			return errors.New("API resource already exists")
		}

		// Check that the provided group can be added to the project
		if (p.config.IsV2() || p.config.IsV3()) && !p.config.MultiGroup &&
			len(p.config.Resources) != 0 && !p.config.HasGroup(p.resource.Group) {
			return fmt.Errorf("multiple groups are not allowed by default, to enable multi-group visit %s",
				"kubebuilder.io/migration/multi-group.html")
		}
	}

	return nil
}

func (p *createAPIPlugin) GetScaffolder() (scaffold.Scaffolder, error) {
	// Load the boilerplate
	bp, err := ioutil.ReadFile(filepath.Join("hack", "boilerplate.go.txt")) // nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("unable to load boilerplate: %v", err)
	}

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

	// Create the actual resource from the resource options
	res := p.resource.NewResource(p.config, p.doResource)
	return scaffold.NewAPIScaffolder(p.config, string(bp), res, p.doResource, p.doController, plugins), nil
}

func (p *createAPIPlugin) PostScaffold() error {
	if p.runMake {
		return internal.RunCmd("Running make", "make")
	}
	return nil
}
