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
	"log"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/cmd/alpha"
	"sigs.k8s.io/kubebuilder/cmd/create"
	cmdinit "sigs.k8s.io/kubebuilder/cmd/init"
	"sigs.k8s.io/kubebuilder/cmd/update"
	"sigs.k8s.io/kubebuilder/cmd/util"
	"sigs.k8s.io/kubebuilder/cmd/version"
)

func main() {
	rootCmd := defaultCommand()

	rootCmd.AddCommand(
		cmdinit.NewInitProjectCmd(),
		create.NewCreateCmd(),
		version.NewVersionCmd(),
	)

	foundProject, version := util.GetProjectVersion()
	if foundProject && version == "1" {
		rootCmd.AddCommand(
			alpha.NewAlphaCommand(),
			update.NewVendorUpdateCmd(),
		)
	}

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

  kubebuilder init --domain example.com --license apache2 --owner "The Kubernetes authors"

- create one or more a new resource APIs and add your code to them:

  kubebuilder create api --group <group> --version <version> --kind <Kind>

Create resource will prompt the user for if it should scaffold the Resource and / or Controller. To only
scaffold a Controller for an existing Resource, select "n" for Resource. To only define
the schema for a Resource without writing a Controller, select "n" for Controller.

After the scaffold is written, api will run make on the project.
`,
		Example: `
	# Initialize your project
	kubebuilder init --domain example.com --license apache2 --owner "The Kubernetes authors"

	# Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
	kubebuilder create api --group ship --version v1beta1 --kind Frigate

	# Edit the API Scheme
	nano api/v1beta1/frigate_types.go

	# Edit the Controller
	nano controllers/frigate_controller.go

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
