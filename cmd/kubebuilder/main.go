/*
Copyright 2017 The Kubernetes Authors.

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
	gobuild "go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/version"
	"github.com/spf13/cobra"
)

func main() {
	util.CheckInstall()
	gopath := os.Getenv("GOPATH")
	if len(gopath) == 0 {
		gopath = gobuild.Default.GOPATH
	}
	util.GoSrc = filepath.Join(gopath, "src")

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if !strings.HasPrefix(filepath.Dir(wd), util.GoSrc) {
		log.Fatalf("kubebuilder must be run from the project root under $GOPATH/src/<package>. "+
			"\nCurrent GOPATH=%s.  \nCurrent directory=%s", gopath, wd)
	}
	util.Repo = strings.Replace(wd, util.GoSrc+string(filepath.Separator), "", 1)

	rootCmd := defaultCommand()

	rootCmd.AddCommand(
		newInitProjectCmd(),
		newAPICommand(),
		version.NewVersionCmd(),
		newDocsCmd(),
		newVendorUpdateCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func defaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "kubebuilder",
		Short: "Development kit for building Kubernetes extensions and tools.",
		Long: `
Development kit for building Kubernetes extensions and tools.

Provides libraries and tools to create new projects, APIs and controllers.
Includes tools for packaging artifacts into an installer container.

Typical project lifecycle:

- initialize a project:

  kubebuilder init --domain k8s.io --license apache2 --owner "The Kubernetes authors

- create one or more a new resource APIs and add your code to them:

  kubebuilder create api --group <group> --version <version> --kind <Kind>

create resource will prompt the user for if it should scaffold the Resource and / or Controller.  To only
scaffold a Controller for an existing Resource, select "n" for Resource.  To only define
the schema for a Resource without writing a Controller, select "n" for Controller.

After the scaffold is written, api will run make on the project.
`,
		Example: `
	# Initialize your project
    kubebuilder init --domain example.com --license apache2 --owner "The Kubernetes authors"

    # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
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
			cmd.Help()
		},
	}
}
