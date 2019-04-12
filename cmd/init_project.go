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
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/cmd/util"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
)

func newInitProjectCmd() *cobra.Command {
	o := projectOptions{
		projectScaffolder: &scaffold.Project{},
	}

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
			o.initializeProject()
		},
	}

	o.bindCmdlineFlags(initCmd)

	return initCmd
}

type projectOptions struct {
	projectScaffolder *scaffold.Project

	dep                bool
	depFlag            *flag.Flag
	depArgs            []string
	skipGoVersionCheck bool
}

func (o *projectOptions) bindCmdlineFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(
		&o.skipGoVersionCheck, "skip-go-version-check", false, "if specified, skip checking the Go version")
	cmd.Flags().BoolVar(
		&o.dep, "dep", true, "if specified, determines whether dep will be used.")
	o.depFlag = cmd.Flag("dep")
	cmd.Flags().StringArrayVar(&o.depArgs, "depArgs", nil, "Additional arguments for dep")

	o.bindBoilerplateFlags(cmd)
	o.bindProjectFlags(cmd)
}

// projectForFlags registers flags for Project fields and returns the Project
func (o *projectOptions) bindProjectFlags(cmd *cobra.Command) {
	p := &o.projectScaffolder.Info
	cmd.Flags().StringVar(&p.Repo, "repo", "", "name of the github repo.  "+
		"defaults to the go package of the current working directory.")
	cmd.Flags().StringVar(&p.Domain, "domain", "k8s.io", "domain for groups")
	cmd.Flags().StringVar(&p.Version, "project-version", project.Version1, "project version")
}

// bindBoilerplateFlags registers flags for Boilerplate fields and returns the Boilerplate
func (o *projectOptions) bindBoilerplateFlags(cmd *cobra.Command) {
	bp := &o.projectScaffolder.Boilerplate
	cmd.Flags().StringVar(&bp.Path, "path", "", "path for boilerplate")
	cmd.Flags().StringVar(&bp.License, "license", "apache2", "license to use to boilerplate.  Maybe one of apache2,none")
	cmd.Flags().StringVar(&bp.Owner, "owner", "", "Owner to add to the copyright")
}

func (o *projectOptions) initializeProject() {

	if err := o.validate(); err != nil {
		log.Fatal(err)
	}

	if err := o.projectScaffolder.Scaffold(); err != nil {
		log.Fatalf("error scaffolding project: %v", err)
	}

	if err := o.postScaffold(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Next: Define a resource with:\n" +
		"$ kubebuilder create api\n")
}

func (o *projectOptions) validate() error {
	if !o.skipGoVersionCheck {
		if err := validateGoVersion(); err != nil {
			return err
		}
	}

	if !depExists() {
		return fmt.Errorf("Dep is not installed. Follow steps at: https://golang.github.io/dep/docs/installation.html")
	}

	if util.ProjectExist() {
		return fmt.Errorf("Failed to initialize project because project is already initialized")
	}

	return nil
}

func validateGoVersion() error {
	err := fetchAndCheckGoVersion()
	if err != nil {
		return fmt.Errorf("%s. You can skip this check using the --skip-go-version-check flag", err)
	}
	return nil
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

func (o *projectOptions) postScaffold() error {
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
			return err
		}

		fmt.Println("Running make...")
		c = exec.Command("make") // #nosec
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		fmt.Println(strings.Join(c.Args, " "))
		if err := c.Run(); err != nil {
			return err
		}
	} else {
		fmt.Println("Skipping `dep ensure`.  Dependencies will not be fetched.")
	}
	return nil
}
