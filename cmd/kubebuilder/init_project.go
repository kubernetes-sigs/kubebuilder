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
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/controller-tools/pkg/scaffold"
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/manager"
	"sigs.k8s.io/controller-tools/pkg/scaffold/project"
)

func newInitProjectCmd() *cobra.Command {
	o := projectOptions{}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project",
		Long: `Initialize a new project including vendor/ directory and Go package directories.

Writes the following files:
- a boilerplate license file
- a PROJECT file with the domain and repo
- a Makefile to build the project
- a Gopkg.toml with project dependencies
- a Kustomization.yaml for customizating manifests
- a Patch file for customizing image for manager manifests
- a cmd/manager/main.go to run

project will prompt the user to run 'dep ensure' after writing the project files.
`,
		Example: `# Scaffold a project using the apache2 license with "The Kubernetes authors" as owners
kubebuilder init --domain example.org --license apache2 --owner "The Kubernetes authors"
`,
		Run: func(cmd *cobra.Command, args []string) {
			o.runInit()
		},
	}

	initCmd.Flags().BoolVar(
		&o.dep, "dep", true, "if specified, determines whether dep will be used.")
	o.depFlag = initCmd.Flag("dep")

	o.prj = projectForFlags(initCmd.Flags())
	o.bp = boilerplateForFlags(initCmd.Flags())
	o.gopkg = &project.GopkgToml{}
	o.mgr = &manager.Cmd{}
	o.dkr = &manager.Dockerfile{}

	return initCmd
}

type projectOptions struct {
	prj     *project.Project
	bp      *project.Boilerplate
	gopkg   *project.GopkgToml
	mgr     *manager.Cmd
	dkr     *manager.Dockerfile
	dep     bool
	depFlag *flag.Flag
}

func (o *projectOptions) runInit() {
	checkGoVersion()

	if !depExists() {
		log.Fatalf("Dep is not installed. Follow steps at: https://golang.github.io/dep/docs/installation.html")
	}

	if util.ProjectExist() {
		fmt.Println("Failed to initialize project bacause project is already initialized")
		return
	}
	// project and boilerplate must come before main so the boilerplate exists
	s := &scaffold.Scaffold{
		BoilerplateOptional: true,
		ProjectOptional:     true,
	}

	p, err := o.prj.GetInput()
	if err != nil {
		log.Fatal(err)
	}

	b, err := o.bp.GetInput()
	if err != nil {
		log.Fatal(err)
	}

	err = s.Execute(input.Options{ProjectPath: p.Path, BoilerplatePath: b.Path}, o.prj, o.bp)
	if err != nil {
		log.Fatal(err)
	}

	// default controller manager image name
	imgName := "controller:latest"

	s = &scaffold.Scaffold{}
	err = s.Execute(input.Options{ProjectPath: p.Path, BoilerplatePath: b.Path},
		o.gopkg,
		o.mgr,
		&project.Makefile{Image: imgName},
		o.dkr,
		&manager.APIs{},
		&manager.Controller{},
		&manager.Config{Image: imgName},
		&project.GitIgnore{},
		&project.Kustomize{},
		&project.KustomizeImagePatch{})
	if err != nil {
		log.Fatal(err)
	}

	if !o.depFlag.Changed {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Run `dep ensure` to fetch dependencies (Recommended) [y/n]?")
		o.dep = util.Yesno(reader)
	}
	if o.dep {
		c := exec.Command("dep", "ensure") // #nosec
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		fmt.Println(strings.Join(c.Args, " "))
		if err := c.Run(); err != nil {
			log.Fatal(err)
		}

		fmt.Println("Running make...")
		c = exec.Command("make") // #nosec
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		fmt.Println(strings.Join(c.Args, " "))
		if err := c.Run(); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("Skipping `dep ensure`.  Dependencies will not be fetched.")
	}
	fmt.Printf("Next: Define a resource with:\n" +
		"$ kubebuilder create api\n")
}

// projectForFlags registers flags for Project fields and returns the Project
func projectForFlags(f *flag.FlagSet) *project.Project {
	p := &project.Project{}
	f.StringVar(&p.Repo, "repo", "", "name of the github repo.  "+
		"defaults to the go package of the current working directory.")
	f.StringVar(&p.Domain, "domain", "k8s.io", "domain for groups")
	f.StringVar(&p.Version, "project-version", "1", "project version")
	return p
}

// boilerplateForFlags registers flags for Boilerplate fields and returns the Boilerplate
func boilerplateForFlags(f *flag.FlagSet) *project.Boilerplate {
	b := &project.Boilerplate{}
	f.StringVar(&b.Path, "path", "", "path for boilerplate")
	f.StringVar(&b.License, "license", "apache2", "license to use to boilerplate.  Maybe one of apache2,none")
	f.StringVar(&b.Owner, "owner", "", "Owner to add to the copyright")
	return b
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

func depExists() bool {
	_, err := exec.LookPath("dep")
	return err == nil
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
