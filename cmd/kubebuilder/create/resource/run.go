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

package resource

import (
	"fmt"
	"log"
	"os"
	"strings"

	controllerct "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/controller"
	createutil "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	generatecmd "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/generate"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/markbates/inflect"
	"github.com/spf13/cobra"
)

var nonNamespacedKind bool
var controller bool
var generate bool

var createResourceCmd = &cobra.Command{
	Use:   "resource",
	Short: "Creates an API group, version and resource",
	Long: `Creates an API group, version and resource.

Also creates:
- tests for the resource
- controller reconcile function
- tests for the controller
`,
	Example: `# Create new resource "Bee" in the "insect" group with version "v1beta"
# Will also create a controller
kubebuilder create resource --group insect --version v1beta1 --kind Bee

# Create new resource without creating a controller for the resource
kubebuilder create resource --group insect --version v1beta1 --kind Bee --controller=false

# Create a non-namespaced resource
kubebuilder create resource --group insect --version v1beta1 --kind Bee --non-namespaced=true
`,
	Run: RunCreateResource,
}

func AddCreateResource(cmd *cobra.Command) {
	createutil.RegisterResourceFlags(createResourceCmd)
	createResourceCmd.Flags().BoolVar(&nonNamespacedKind, "non-namespaced", false, "if set, the API kind will be non namespaced")
	createResourceCmd.Flags().BoolVar(&controller, "controller", true, "if true, generate the controller code for the resource")
	createResourceCmd.Flags().BoolVar(&generate, "generate", true, "generate source code")
	createResourceCmd.Flags().BoolVar(&createutil.AllowPluralKind, "plural-kind", false, "allow the kind to be plural")
	cmd.AddCommand(createResourceCmd)
}

func RunCreateResource(cmd *cobra.Command, args []string) {
	if _, err := os.Stat("pkg"); err != nil {
		log.Fatalf("could not find 'pkg' directory.  must run kubebuilder init before creating resources")
	}

	util.GetDomain()
	createutil.ValidateResourceFlags()

	cr := util.GetCopyright(createutil.Copyright)

	fmt.Printf("Creating API files for you to edit...\n")
	createGroup(cr)
	createVersion(cr)
	createResource(cr)
	if generate {
		fmt.Printf("Generating code for new resource...  " +
			"Regenerate after editing resources files by running `kubebuilder build generated`.\n")
		generatecmd.RunGenerate(cmd, args)
	}
	fmt.Printf("Next: Install the API, run the controller and create an instance with:\n" +
		"$ GOBIN=${PWD}/bin go install ${PWD#$GOPATH/src/}/cmd/controller-manager\n" +
		"$ bin/controller-manager --kubeconfig ~/.kube/config\n" +
		"$ kubectl apply -f hack/sample/" + strings.ToLower(createutil.KindName) + ".yaml\n")
}

func createResource(boilerplate string) {
	args := resourceTemplateArgs{
		boilerplate,
		util.Domain,
		createutil.GroupName,
		createutil.VersionName,
		createutil.KindName,
		createutil.ResourceName,
		util.Repo,
		inflect.NewDefaultRuleset().Pluralize(createutil.KindName),
		nonNamespacedKind,
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Edit your API schema...\n")
	doResource(dir, args)
	doResourceTest(dir, args)

	if controller {
		fmt.Printf("Creating controller ...\n")
		c := controllerct.ControllerArguments{CoreType: false}
		c.CreateController(boilerplate)
	}

	fmt.Printf("Edit your sample resource instance...\n")
	doSample(dir, args)
}
