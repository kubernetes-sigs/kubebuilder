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
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/cmd/util"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/controller"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

type apiOptions struct {
	r                                *resource.Resource
	resourceFlag, controllerFlag     *flag.Flag
	doResource, doController, doMake bool
}

// APICmd represents the resource command

func (o *apiOptions) runAddAPI() {
	dieIfNoProject()

	reader := bufio.NewReader(os.Stdin)
	if !o.resourceFlag.Changed {
		fmt.Println("Create Resource under pkg/apis [y/n]?")
		o.doResource = util.Yesno(reader)
	}

	if !o.controllerFlag.Changed {
		fmt.Println("Create Controller under pkg/controller [y/n]?")
		o.doController = util.Yesno(reader)
	}

	if o.r.Group == "" {
		log.Fatalf("Must specify --group")
	}
	if o.r.Version == "" {
		log.Fatalf("Must specify --version")
	}
	if o.r.Kind == "" {
		log.Fatalf("Must specify --kind")
	}

	fmt.Println("Writing scaffold for you to edit...")

	r := o.r
	if o.doResource {
		fmt.Println(filepath.Join("pkg", "apis", r.Group, r.Version,
			fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind))))
		fmt.Println(filepath.Join("pkg", "apis", r.Group, r.Version,
			fmt.Sprintf("%s_types_test.go", strings.ToLower(r.Kind))))

		err := (&scaffold.Scaffold{}).Execute(input.Options{},
			&resource.Register{Resource: r},
			&resource.Types{Resource: r},
			&resource.VersionSuiteTest{Resource: r},
			&resource.TypesTest{Resource: r},
			&resource.Doc{Resource: r},
			&resource.Group{Resource: r},
			&resource.AddToScheme{Resource: r},
			&resource.CRDSample{Resource: r},
			&resource.CRDStatus{Resource: r},
		)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// disable generation of example reconcile body if not scaffolding resource
		// because this could result in a fork-bomb of k8s resources where watching a
		// deployment, replicaset etc. results in generating deployment which
		// end up generating replicaset, pod etc recursively.
		r.CreateExampleReconcileBody = false
	}

	if o.doController {
		fmt.Println(filepath.Join("pkg", "controller", strings.ToLower(r.Kind),
			fmt.Sprintf("%s_controller.go", strings.ToLower(r.Kind))))
		fmt.Println(filepath.Join("pkg", "controller", strings.ToLower(r.Kind),
			fmt.Sprintf("%s_controller_test.go", strings.ToLower(r.Kind))))

		err := (&scaffold.Scaffold{}).Execute(input.Options{},
			&controller.Controller{Resource: r},
			&controller.AddController{Resource: r},
			&controller.Test{Resource: r},
			&controller.SuiteTest{Resource: r},
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	if o.doMake {
		fmt.Println("Running make...")
		cm := exec.Command("make") // #nosec
		cm.Stderr = os.Stderr
		cm.Stdout = os.Stdout
		if err := cm.Run(); err != nil {
			log.Fatal(err)
		}
	}
}

func newAPICommand() *cobra.Command {
	o := apiOptions{}

	apiCmd := &cobra.Command{
		Use:   "create api",
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
	nano pkg/apis/ship/v1beta1/frigate_types.go

	# Edit the Controller
	nano pkg/controller/frigate/frigate_controller.go

	# Edit the Controller Test
	nano pkg/controller/frigate/frigate_controller_test.go

	# Install CRDs into the Kubernetes cluster using kubectl apply
	make install

	# Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
	make run
`,
		Run: func(cmd *cobra.Command, args []string) {
			o.runAddAPI()
		},
	}

	apiCmd.Flags().BoolVar(&o.doMake, "make", true,
		"if true, run make after generating files")
	apiCmd.Flags().BoolVar(&o.doResource, "resource", true,
		"if set, generate the resource without prompting the user")
	o.resourceFlag = apiCmd.Flag("resource")
	apiCmd.Flags().BoolVar(&o.doController, "controller", true,
		"if set, generate the controller without prompting the user")
	o.controllerFlag = apiCmd.Flag("controller")
	o.r = ResourceForFlags(apiCmd.Flags())

	return apiCmd
}

// dieIfNoProject checks to make sure the command is run from a directory containing a project file.
func dieIfNoProject() {
	if _, err := os.Stat("PROJECT"); os.IsNotExist(err) {
		log.Fatalf("Command must be run from a directory containing %s", "PROJECT")
	}
}

// ResourceForFlags registers flags for Resource fields and returns the Resource
func ResourceForFlags(f *flag.FlagSet) *resource.Resource {
	r := &resource.Resource{}
	f.StringVar(&r.Kind, "kind", "", "resource Kind")
	f.StringVar(&r.Group, "group", "", "resource Group")
	f.StringVar(&r.Version, "version", "", "resource Version")
	f.BoolVar(&r.Namespaced, "namespaced", true, "resource is namespaced")
	f.BoolVar(&r.CreateExampleReconcileBody, "example", true,
		"if true an example reconcile body should be written while scaffolding a resource.")
	return r
}
