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
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/cmd/internal"
	"sigs.k8s.io/kubebuilder/cmd/util"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
	"sigs.k8s.io/kubebuilder/plugins/addon"
)

type apiOptions struct {
	apiScaffolder                scaffold.API
	resourceFlag, controllerFlag *flag.Flag

	// runMake indicates whether to run make or not after scaffolding APIs
	runMake bool

	// pattern indicates that we should use a plugin to build according to a pattern
	pattern string
}

func (o *apiOptions) bindCmdFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.runMake, "make", true,
		"if true, run make after generating files")
	cmd.Flags().BoolVar(&o.apiScaffolder.DoResource, "resource", true,
		"if set, generate the resource without prompting the user")
	o.resourceFlag = cmd.Flag("resource")
	cmd.Flags().BoolVar(&o.apiScaffolder.DoController, "controller", true,
		"if set, generate the controller without prompting the user")
	o.controllerFlag = cmd.Flag("controller")
	if os.Getenv("KUBEBUILDER_ENABLE_PLUGINS") != "" {
		cmd.Flags().StringVar(&o.pattern, "pattern", "",
			"generates an API following an extension pattern (addon)")
	}
	cmd.Flags().BoolVar(&o.apiScaffolder.Force, "force", false,
		"attempt to create resource even if it already exists")
	o.apiScaffolder.Resource = resourceForFlags(cmd.Flags())
}

// resourceForFlags registers flags for Resource fields and returns the Resource
func resourceForFlags(f *flag.FlagSet) *resource.Resource {
	r := &resource.Resource{}
	f.StringVar(&r.Kind, "kind", "", "resource Kind")
	f.StringVar(&r.Group, "group", "", "resource Group")
	f.StringVar(&r.Version, "version", "", "resource Version")
	f.BoolVar(&r.Namespaced, "namespaced", true, "resource is namespaced")
	f.BoolVar(&r.CreateExampleReconcileBody, "example", true,
		"if true an example reconcile body should be written while scaffolding a resource.")
	return r
}

// APICmd represents the resource command
func (o *apiOptions) runAddAPI() {
	internal.DieIfNotConfigured()

	switch strings.ToLower(o.pattern) {
	case "":
		// Default pattern

	case "addon":
		o.apiScaffolder.Plugins = append(o.apiScaffolder.Plugins, &addon.Plugin{})

	default:
		log.Fatalf("unknown pattern %q", o.pattern)
	}

	if err := o.apiScaffolder.Validate(); err != nil {
		log.Fatalln(err)
	}

	reader := bufio.NewReader(os.Stdin)
	if !o.resourceFlag.Changed {
		fmt.Println("Create Resource [y/n]")
		o.apiScaffolder.DoResource = util.YesNo(reader)
	}

	if !o.controllerFlag.Changed {
		fmt.Println("Create Controller [y/n]")
		o.apiScaffolder.DoController = util.YesNo(reader)
	}

	fmt.Println("Writing scaffold for you to edit...")

	if err := o.apiScaffolder.Scaffold(); err != nil {
		log.Fatal(err)
	}

	if err := o.postScaffold(); err != nil {
		log.Fatal(err)
	}
}

func (o *apiOptions) postScaffold() error {
	if o.runMake {
		fmt.Println("Running make...")
		cm := exec.Command("make") // #nosec
		cm.Stderr = os.Stderr
		cm.Stdout = os.Stdout
		if err := cm.Run(); err != nil {
			return fmt.Errorf("error running make: %v", err)
		}
	}
	return nil
}

func newAPICommand() *cobra.Command {
	options := apiOptions{
		apiScaffolder: scaffold.API{},
	}

	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Scaffold a Kubernetes API",
		Long: `Scaffold a Kubernetes API by creating a Resource definition and / or a Controller.

create resource will prompt the user for if it should scaffold the Resource and / or Controller.  To only
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
		Run: func(cmd *cobra.Command, args []string) {
			options.runAddAPI()
		},
	}

	options.bindCmdFlags(apiCmd)

	return apiCmd
}
