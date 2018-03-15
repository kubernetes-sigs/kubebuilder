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
	"github.com/spf13/cobra"
	"path/filepath"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Generate installation config for the API",
	Long: `Generate installation config for the API.  Includes:
- Namespace
- RBAC ClusterRole
- RBAC ClusterRoleBinding
- CRDs
- Controller Deployment
`,
	Run: func(cmd *cobra.Command, args []string) {
		if controllerImage == "" {
			fmt.Printf("Must specify --controller-image\n")
			return
		}
		if name == "" {
			fmt.Printf("Must specify --name\n")
			return
		}
		CodeGenerator{}.Execute()
	},
}

var controllerImage string
var name string
var output string

func AddCreateConfig(cmd *cobra.Command) {
	cmd.AddCommand(configCmd)
	configCmd.Flags().StringVar(&controllerImage, "controller-image", "", "name of the controller container to run.")
	configCmd.Flags().StringVar(&name, "name", "", "name of the installation.")
	configCmd.Flags().StringVar(&output, "output", filepath.Join("hack", "install.yaml"), "location to write yaml to")
}
