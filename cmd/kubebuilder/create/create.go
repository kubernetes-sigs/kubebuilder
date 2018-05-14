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

package create

import (
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/config"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/controller"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/example"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/resource"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Command group for bootstrapping new resources.",
	Long:  `Command group for bootstrapping new resources.`,
	Example: `# Create new resource "Bee" in the "insect" group with version "v1beta"
# Will also create a test, controller and controller test for the resource
kubebuilder create resource --group insect --version v1beta --kind Bee
`,
	Run: RunCreate,
}

func AddCreate(cmd *cobra.Command) {
	cmd.AddCommand(createCmd)
	util.RegisterCopyrightFlag(cmd)
	resource.AddCreateResource(createCmd)
	config.AddCreateConfig(createCmd)
	example.AddCreateExample(createCmd)
	controller.AddCreateController(createCmd)
}

func RunCreate(cmd *cobra.Command, args []string) {
	cmd.Help()
}
