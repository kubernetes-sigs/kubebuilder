// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Generate installation config for the API",
	Long: `Generate installation config for the API.

May create config for just CRDs or for controller-manager and CRDs.
`,
	Example: `# Generate config to install controller-manager and CRDs into a cluster
kubebuilder create config --controller-image myimage:v1 --name myextensionname

# Generate config to install only CRDs
kubebuilder create config --crds

# Generate config to install controller-manager and CRDs using a Deployment
kubebuilder create config --controller-image myimage:v1 --name myextensionname --controller-type deployment

# Generate config file at a specific location
kubebuilder create config --crds --output myextensionname.yaml

`,
	Run: func(cmd *cobra.Command, args []string) {
		if controllerType != "statefulset" && controllerType != "deployment" {
			fmt.Printf(
				"Invalid value %s for --controller-type, must be statefulset or deployment.\n", controllerType)
			return
		}
		if controllerImage == "" && !crds {
			fmt.Printf("Must either specify --controller-image or set --crds.\n")
			return
		}
		if name == "" && !crds {
			fmt.Printf("Must either specify the name of the extension with --name or set --crds.\n")
			return
		}
		CodeGenerator{}.Execute()
		log.Printf("Config written to %s", output)
	},
}

var (
	controllerType, controllerImage, name, output, crdNamespace string
	crds                                                        bool
)

func AddCreateConfig(cmd *cobra.Command) {
	cmd.AddCommand(configCmd)
	configCmd.Flags().StringVar(&controllerType, "controller-type", "statefulset", "either statefulset or deployment.")
	configCmd.Flags().BoolVar(&crds, "crds", false, "if set to true, only generate crd definitions")
	configCmd.Flags().StringVar(&crdNamespace, "crd-namespace", "", "if set, install CRDs to this namespace.")
	configCmd.Flags().StringVar(&controllerImage, "controller-image", "", "name of the controller container to run.")
	configCmd.Flags().StringVar(&name, "name", "", "name of the installation.  used to generate the namespace and resource names.")
	configCmd.Flags().StringVar(&output, "output", filepath.Join("hack", "install.yaml"), "location to write yaml to")
}
