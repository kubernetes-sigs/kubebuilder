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
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/cmd/util"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/manager"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
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
- a Patch file for enabling prometheus metrics
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
		&o.skipGoVersionCheck, "skip-go-version-check", false, "if specified, skip checking the Go version")
	initCmd.Flags().BoolVar(
		&o.dep, "dep", true, "if specified, determines whether dep will be used.")
	o.depFlag = initCmd.Flag("dep")
	initCmd.Flags().StringArrayVar(&o.depArgs, "depArgs", nil, "Additional arguments for dep")

	o.prj = projectForFlags(initCmd.Flags())
	o.bp = boilerplateForFlags(initCmd.Flags())
	o.gopkg = &project.GopkgToml{}
	o.mgr = &manager.Cmd{}
	o.dkr = &manager.Dockerfile{}

	return initCmd
}

type projectOptions struct {
	prj                *project.Project
	bp                 *project.Boilerplate
	gopkg              *project.GopkgToml
	mgr                *manager.Cmd
	dkr                *manager.Dockerfile
	dep                bool
	depFlag            *flag.Flag
	depArgs            []string
	skipGoVersionCheck bool
}

func (o *projectOptions) runInit() {
	if !o.skipGoVersionCheck {
		ensureGoVersionIsCompatible()
	}

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
		&manager.Webhook{},
		&manager.Config{Image: imgName},
		&project.GitIgnore{},
		&project.Kustomize{},
		&project.KustomizeImagePatch{},
		&project.KustomizePrometheusMetricsPatch{},
		&project.KustomizeAuthProxyPatch{},
		&project.AuthProxyService{},
		&project.AuthProxyRole{},
		&project.AuthProxyRoleBinding{})
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
		if len(o.depArgs) > 0 {
			c.Args = append(c.Args, o.depArgs...)
		}
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

func ensureGoVersionIsCompatible() {
	err := fetchAndCheckGoVersion()
	if err != nil {
		log.Fatalf("%s. You can skip this check using the --skip-go-version-check flag", err)
	}
}

func fetchAndCheckGoVersion() error {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to retrieve 'go version': %v", string(out))
	}

	split := strings.Split(string(out), " ")
	if len(split) < 3 {
		return fmt.Errorf("Found invalid Go version: %q", string(out))
	}
	goVer := split[2]
	if err := checkGoVersion(goVer); err != nil {
		return fmt.Errorf("Go version '%s' is incompatible because '%s'", goVer, err)
	}
	return nil
}

func checkGoVersion(verStr string) error {
	goVerRegex := `^go?([0-9]+)\.([0-9]+)([\.0-9A-Za-z\-]+)?$`
	m := regexp.MustCompile(goVerRegex).FindStringSubmatch(verStr)
	if m == nil {
		return fmt.Errorf("invalid version string")
	}

	major, err := strconv.Atoi(m[1])
	if err != nil {
		return fmt.Errorf("error parsing major version '%s': %s", m[1], err)
	}

	minor, err := strconv.Atoi(m[2])
	if err != nil {
		return fmt.Errorf("error parsing minor version '%s': %s", m[2], err)
	}

	if major < 1 || minor < 11 {
		return fmt.Errorf("requires version >= 1.11")
	}

	return nil
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
