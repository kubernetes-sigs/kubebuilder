/*
Copyright 2018 The Kubernetes Authors.

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

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/cmd/internal"
	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/plugins/addon"
)

type apiError struct {
	err error
}

func (e apiError) Error() string {
	return fmt.Sprintf("failed to create API: %v", e.err)
}

func newAPICmd() *cobra.Command {
	options := &apiOptions{}

	cmd := &cobra.Command{
		Use:   "api",
		Short: "Scaffold a Kubernetes API",
		Long: `Scaffold a Kubernetes API by creating a Resource definition and / or a Controller.

kubebuilder create api will prompt the user asking if it should scaffold the Resource and / or Controller. To only
scaffold a Controller for an existing Resource, select "n" for Resource.  To only define
the schema for a Resource without writing a Controller, select "n" for Controller.

After the scaffold is written, api will run make on the project.
`,
		Example: `	# Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
	kubebuilder create api --group ship --version v1beta1 --kind Frigate
	
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
		Run: func(_ *cobra.Command, _ []string) {
			if err := run(options); err != nil {
				log.Fatal(apiError{err})
			}
		},
	}

	options.bindFlags(cmd)

	return cmd
}

var _ commandOptions = &apiOptions{}

type apiOptions struct {
	// pattern indicates that we should use a plugin to build according to a pattern
	pattern string

	resource *resource.Options

	// Check if we have to scaffold resource and/or controller
	resourceFlag   *flag.Flag
	controllerFlag *flag.Flag
	doResource     bool
	doController   bool

	// force indicates that the resource should be created even if it already exists
	force bool

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool
}

func (o *apiOptions) bindFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.runMake, "make", true, "if true, run make after generating files")

	cmd.Flags().BoolVar(&o.doResource, "resource", true,
		"if set, generate the resource without prompting the user")
	o.resourceFlag = cmd.Flag("resource")
	cmd.Flags().BoolVar(&o.doController, "controller", true,
		"if set, generate the controller without prompting the user")
	o.controllerFlag = cmd.Flag("controller")

	if os.Getenv("KUBEBUILDER_ENABLE_PLUGINS") != "" {
		cmd.Flags().StringVar(&o.pattern, "pattern", "",
			"generates an API following an extension pattern (addon)")
	}

	cmd.Flags().BoolVar(&o.force, "force", false,
		"attempt to create resource even if it already exists")

	o.resource = &resource.Options{}
	cmd.Flags().StringVar(&o.resource.Kind, "kind", "", "resource Kind")
	cmd.Flags().StringVar(&o.resource.Group, "group", "", "resource Group")
	cmd.Flags().StringVar(&o.resource.Version, "version", "", "resource Version")
	cmd.Flags().BoolVar(&o.resource.Namespaced, "namespaced", true, "resource is namespaced")
	cmd.Flags().BoolVar(&o.resource.CreateExampleReconcileBody, "example", true,
		"if true an example reconcile body should be written while scaffolding a resource.")
}

func (o *apiOptions) loadConfig() (*config.Config, error) {
	projectConfig, err := config.Load()
	if os.IsNotExist(err) {
		return nil, errors.New("unable to find configuration file, project must be initialized")
	}

	return projectConfig, err
}

func (o *apiOptions) validate(c *config.Config) error {
	if err := o.resource.Validate(); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	if !o.resourceFlag.Changed {
		fmt.Println("Create Resource [y/n]")
		o.doResource = internal.YesNo(reader)
	}
	if !o.controllerFlag.Changed {
		fmt.Println("Create Controller [y/n]")
		o.doController = internal.YesNo(reader)
	}

	// In case we want to scaffold a resource API we need to do some checks
	if o.doResource {
		// Skip the following check for v1 as resources aren't tracked
		if !c.IsV1() {
			// Check that resource doesn't exist or flag force was set
			if !o.force {
				resourceExists := false
				for _, r := range c.Resources {
					if r.Group == o.resource.Group &&
						r.Version == o.resource.Version &&
						r.Kind == o.resource.Kind {
						resourceExists = true
						break
					}
				}
				if resourceExists {
					return errors.New("API resource already exists")
				}
			}
		}

		// The following check is v2 specific as multi-group isn't enabled by default
		if c.IsV2() {
			// Check the group is the same for single-group projects
			if !c.MultiGroup {
				validGroup := true
				for _, existingGroup := range c.ResourceGroups() {
					if !strings.EqualFold(o.resource.Group, existingGroup) {
						validGroup = false
						break
					}
				}
				if !validGroup {
					return fmt.Errorf("multiple groups are not allowed by default, to enable multi-group visit %s",
						"kubebuilder.io/migration/multi-group.html")
				}
			}
		}
	}

	return nil
}

func (o *apiOptions) scaffolder(c *config.Config) (scaffold.Scaffolder, error) {
	// Load the boilerplate
	bp, err := ioutil.ReadFile(filepath.Join("hack", "boilerplate.go.txt")) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("unable to load boilerplate: %v", err)
	}

	// Create the actual resource from the resource options
	var res *resource.Resource
	switch {
	case c.IsV1():
		res = o.resource.NewV1Resource(&c.Config, o.doResource)
	case c.IsV2():
		res = o.resource.NewResource(&c.Config, o.doResource)
	default:
		return nil, fmt.Errorf("unknown project version %v", c.Version)
	}

	// Load the requested plugins
	plugins := make([]model.Plugin, 0)
	switch strings.ToLower(o.pattern) {
	case "":
		// Default pattern

	case "addon":
		plugins = append(plugins, &addon.Plugin{})

	default:
		return nil, fmt.Errorf("unknown pattern %q", o.pattern)
	}

	return scaffold.NewAPIScaffolder(c, string(bp), res, o.doResource, o.doController, plugins), nil
}

func (o *apiOptions) postScaffold(_ *config.Config) error {
	if o.runMake {
		return internal.RunCmd("Running make", "make")
	}

	return nil
}
