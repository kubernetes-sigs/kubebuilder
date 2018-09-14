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
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-tools/pkg/scaffold/manager"
	"sigs.k8s.io/controller-tools/pkg/scaffold/project"
)

type initOptions struct {
	domain         string
	copyright      string
	bazel          bool
	controllerOnly bool
	projectVersion string
	projectOptions
}

func AddInit(cmd *cobra.Command) {
	o := initOptions{}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project",
		Long:  `Initialize a new project including vendor/ directory and Go package directories.`,
		Example: `# Initialize project structure
kubebuilder init --domain mydomain
`,
		Run: func(cmd *cobra.Command, args []string) {
			o.runInitRepo()
		},
	}

	v0comment := "Works only with project-version v0, "
	initCmd.Flags().StringVar(&o.domain, "domain", "", "domain for the API groups")
	initCmd.Flags().StringVar(&o.copyright, "copyright", filepath.Join("hack", "boilerplate.go.txt"), v0comment+"Location of copyright boilerplate file.")
	initCmd.Flags().BoolVar(&o.bazel, "bazel", false, v0comment+"if true, setup Bazel workspace artifacts")
	initCmd.Flags().BoolVar(&o.controllerOnly, "controller-only", false, v0comment+"if true, setup controller only")
	initCmd.Flags().StringVar(&o.projectVersion, "project-version", "v1", "if set to v0, init project with kubebuilder legacy version")

	initCmd.Flags().BoolVar(
		&o.dep, "dep", true, "if specified, determines whether dep will be used.")
	o.depFlag = initCmd.Flag("dep")

	o.prj = projectForFlags(initCmd.Flags())
	o.bp = boilerplateForFlags(initCmd.Flags())
	o.gopkg = &project.GopkgToml{}
	o.mgr = &manager.Cmd{}
	o.dkr = &manager.Dockerfile{}

	cmd.AddCommand(initCmd)
}

func (o *initOptions) runInitRepo() {
	checkGoVersion()

	if !depExists() {
		log.Fatalf("Dep is not installed. Follow steps at: https://golang.github.io/dep/docs/installation.html")
	}

	if o.projectVersion == "v1" {
		if len(o.domain) != 0 {
			o.prj.Domain = o.domain
		}
		o.RunInit()
		return
	}

	if len(o.domain) == 0 {
		log.Fatal("Must specify --domain")
	}
	cr := util.GetCopyright(o.copyright)

	fmt.Printf("Initializing project structure...\n")
	if o.bazel {
		createBazelWorkspace()
	}
	createControllerManager(cr)
	//createInstaller(cr)
	createAPIs(cr, o.domain)
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
	doArgs(cr, o.controllerOnly)
	RunVendorInstall(nil, []string{o.copyright})
	createBoilerplate()
	fmt.Printf("Next: Define a resource with:\n" +
		"$ kubebuilder create resource\n")
}

func checkGoVersion() {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("Could not execute 'go version': %v", err)
	}

	split := strings.Split(string(out), " ")
	if len(split) < 3 {
		log.Fatalf("Invalid go version: %q", string(out))
	}
	goVersion := strings.TrimPrefix(split[2], "go")
	if ver, err := semver.NewVersion(goVersion); err != nil {
		if err != nil {
			log.Fatalf("Invalid go version %q: %v", goVersion, err)
		}
		c, err := semver.NewConstraint(">= 1.10")
		if err != nil {
			log.Fatal("Invalid constraint: %v", err)
		}
		if !c.Check(ver) {
			log.Fatalf("The go version is %v, must be 1.10+", goVersion)
		}
	}
}

func execute(path, templateName, templateValue string, data interface{}) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	util.WriteIfNotFound(filepath.Join(dir, path), templateName, templateValue, data)
}

type templateArgs struct {
	BoilerPlate    string
	Repo           string
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
