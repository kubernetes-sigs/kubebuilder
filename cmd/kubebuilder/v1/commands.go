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

package v1

import (
	"github.com/spf13/cobra"
)

func AddCmds(cmd *cobra.Command) {
	AddAPICommand(cmd)
	cmd.AddCommand(vendorUpdateCmd())
	cmd.AddCommand(docsCmd())
	cmd.AddCommand(newAlphaCommand())

	cmd.Example = `# Initialize your project
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
	make run`

	cmd.Long = `
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
	`
}

// newAlphaCommand returns alpha subcommand which will be mounted
// at the root command by the caller.
func newAlphaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alpha",
		Short: "Exposes commands which are in experimental or early stages of development",
		Long:  `Command group for commands which are either experimental or in early stages of development`,
		Example: `
# scaffolds webhook server
kubebuilder alpha webhook <params>
`,
	}

	cmd.AddCommand(
		newWebhookCmd(),
	)
	return cmd
}
