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

package controller

import (
	"fmt"
	"log"
	"os"

	createutil "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	generatecmd "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/generate"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/markbates/inflect"
	"github.com/spf13/cobra"
	"strings"
)

var nonNamespacedKind bool
var generate bool
var CoreType bool

var createControllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "Creates a controller for an API group, version and resource",
	Long: `Creates a controller for an API group, version and resource.

Also creates:
- controller reconcile function
- tests for the controller
`,
	Example: `# Create a controller for resource "Bee" in the "insect" group with version "v1beta"
kubebuilder create controller --group insect --version v1beta1 --kind Bee

# Create a controller for k8s core type "Deployment" in the "apps" group with version "v1beta2"
kubebuilder create controller --group apps --version v1beta2 --kind Deployment --core-type
`,
	Run: RunCreateController,
}

func AddCreateController(cmd *cobra.Command) {
	createutil.RegisterResourceFlags(createControllerCmd)
	createControllerCmd.Flags().BoolVar(&nonNamespacedKind, "non-namespaced", false, "if set, the API kind will be non namespaced")
	createControllerCmd.Flags().BoolVar(&generate, "generate", true, "generate controller code")
	createControllerCmd.Flags().BoolVar(&CoreType, "core-type", false, "generate controller for core type")
	cmd.AddCommand(createControllerCmd)
}

func RunCreateController(cmd *cobra.Command, args []string) {
	if _, err := os.Stat("pkg"); err != nil {
		log.Fatalf("could not find 'pkg' directory.  must run kubebuilder init before creating controller")
	}

	util.GetDomain()
	createutil.ValidateResourceFlags()

	cr := util.GetCopyright(createutil.Copyright)

	fmt.Printf("Creating controller ...\n")
	CreateController(cr)
	if generate {
		fmt.Printf("Generating code for new controller... " +
				"Regenerate after editing controller files by running `kubebuilder generate clean; kubebuilder generate`.\n")
		generatecmd.RunGenerate(cmd, args)
	}
	fmt.Printf("Next: Run the controller and create an instance with:\n" +
		"$ GOBIN=${PWD}/bin go install ${PWD#$GOPATH/src/}/cmd/controller-manager\n" +
		"$ bin/controller-manager --kubeconfig ~/.kube/config\n" +
		"$ kubectl apply -f hack/sample/" + strings.ToLower(createutil.KindName) + ".yaml\n")
}

func CreateController(boilerplate string) {
	args := controllerTemplateArgs{
		boilerplate,
		util.Domain,
		createutil.GroupName,
		createutil.VersionName,
		createutil.KindName,
		createutil.ResourceName,
		util.Repo,
		inflect.NewDefaultRuleset().Pluralize(createutil.KindName),
		nonNamespacedKind,
		CoreType,
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Edit your controller function...\n")
	doController(dir, args)
	doControllerTest(dir, args)
}