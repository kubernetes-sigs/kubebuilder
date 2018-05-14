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
	"runtime"
	"strconv"
	"strings"

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
var controllerOnly bool

func AddInit(cmd *cobra.Command) {
	cmd.AddCommand(repoCmd)
	repoCmd.Flags().StringVar(&domain, "domain", "", "domain for the API groups")
	repoCmd.Flags().StringVar(&copyright, "copyright", filepath.Join("hack", "boilerplate.go.txt"), "Location of copyright boilerplate file.")
	repoCmd.Flags().BoolVar(&bazel, "bazel", false, "if true, setup Bazel workspace artifacts")
	repoCmd.Flags().BoolVar(&controllerOnly, "controller-only", false, "if true, setup controller only")
}

func runInitRepo(cmd *cobra.Command, args []string) {
	version := runtime.Version()
	if versionCmp(version, "go1.10") < 0 {
		log.Fatalf("The go version is %v, must be 1.10+", version)
	}
	if !depExists() {
		log.Fatalf("Dep is not installed. Follow steps at: https://golang.github.io/dep/docs/installation.html")
	}

	if len(domain) == 0 {
		log.Fatal("Must specify --domain")
	}
	cr := util.GetCopyright(copyright)

	fmt.Printf("Initializing project structure...\n")
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
		filepath.Join("pkg", "inject"),
		//filepath.Join("pkg", "openapi"),
	}

	fmt.Printf("\t%s/\n", filepath.Join("pkg", "controller"))
	for _, p := range pkgs {
		createPackage(cr, p)
	}
	doDockerfile()
	doInject(cr)
	doArgs(cr, controllerOnly)
	//os.MkdirAll("bin", 0700)
	RunVendorInstall(nil, nil)

	createBoilerplate()
	fmt.Printf("Next: Define a resource with:\n" +
		"$ kubebuilder create resource\n")
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
	ControllerOnly bool
}

func versionCmp(v1 string, v2 string) int {
	v1s := strings.Split(strings.Replace(v1, "go", "", 1), ".")
	v2s := strings.Split(strings.Replace(v2, "go", "", 1), ".")
	for i := 0; i < len(v1s) && i < len(v2s); i++ {
		mv1, err1 := strconv.Atoi(v1s[i])
		mv2, err2 := strconv.Atoi(v2s[i])
		if err1 == nil && err2 == nil {
			cmp := mv1 - mv2
			if cmp > 0 {
				return 1
			} else if cmp < 0 {
				return -1
			}
		} else {
			log.Fatalf("Unexpected error comparing %v with %v", v1, v2)
		}
	}
	if len(v1s) == len(v2s) {
		return 0
	} else if len(v1s) > len(v2s) {
		return 1
	} else {
		return -1
	}
}
