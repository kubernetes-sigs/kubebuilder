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

package initproject

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new project",
	Long:  `Initialize a new project including vendor/ directory and go package directories.`,
	Example: `# Initialize project structure
kubebuilder init repo --domain mydomain
`,
	Run: runInitRepo,
}

var domain string
var copyright string
var bazel bool

func AddInit(cmd *cobra.Command) {
	cmd.AddCommand(repoCmd)
	repoCmd.Flags().StringVar(&domain, "domain", "", "domain for the API groups")
	repoCmd.Flags().StringVar(&copyright, "copyright", "boilerplate.go.txt", "Location of copyright boilerplate file.")
	repoCmd.Flags().BoolVar(&bazel, "bazel", false, "if true, setup Bazel workspace artifacts")
}

func runInitRepo(cmd *cobra.Command, args []string) {
	if len(domain) == 0 {
		log.Fatal("Must specify --domain")
	}
	cr := util.GetCopyright(copyright)

	fmt.Printf("Initializing project structure...\n")
	RunVendorInstall(nil, nil)
	if bazel {
		createBazelWorkspace()
	}
	createControllerManager(cr)
	//createInstaller(cr)
	createAPIs(cr)
	//runCreateApiserver(cr)

	pkgs := []string{
		filepath.Join("hack"),
		filepath.Join("pkg"),
		filepath.Join("pkg", "controller"),
		filepath.Join("pkg", "controller", "args"),
		filepath.Join("pkg", "inject"),
		//filepath.Join("pkg", "openapi"),
	}

	fmt.Printf("\t%s/\n", filepath.Join("pkg", "controller"))
	for _, p := range pkgs {
		createPackage(cr, p)
	}
	doDockerfile()
	doInject(cr)
	doArgs(cr)
	//os.MkdirAll("bin", 0700)

	createBoilerplate()
	fmt.Printf("Next: Create a resource using `kubebuilder create resource`.\n")
}

func execute(path, templateName, templateValue string, data interface{}) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	util.WriteIfNotFound(filepath.Join(dir, path), templateName, templateValue, data)
}

type templateArgs struct {
	BoilerPlate string
	Repo        string
}
