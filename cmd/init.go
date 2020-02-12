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
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/cmd/internal"
	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
)

type initError struct {
	err error
}

func (e initError) Error() string {
	return fmt.Sprintf("failed to initialize project: %v", e.err)
}

func newInitCmd() *cobra.Command {
	options := &initOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project",
		Long: `Initialize a new project including vendor/ directory and Go package directories.

Writes the following files:
- a boilerplate license file
- a PROJECT file with the project configuration
- a Makefile to build the project
- a go.mod with project dependencies
- a Kustomization.yaml for customizating manifests
- a Patch file for customizing image for manager manifests
- a Patch file for enabling prometheus metrics
- a cmd/manager/main.go to run

project will prompt the user to run 'dep ensure' after writing the project files.
`,
		Example: `# Scaffold a project using the apache2 license with "The Kubernetes authors" as owners
kubebuilder init --domain example.org --license apache2 --owner "The Kubernetes authors"
`,
		Run: func(_ *cobra.Command, _ []string) {
			if err := run(options); err != nil {
				log.Fatal(initError{err})
			}
		},
	}

	options.bindFlags(cmd)

	return cmd
}

var _ commandOptions = &initOptions{}

type initOptions struct {
	config *config.Config

	// boilerplate options
	license string
	owner   string

	// deprecated flags
	depFlag *flag.Flag
	depArgs []string
	dep     bool

	// flags
	fetchDeps          bool
	skipGoVersionCheck bool
}

func (o *initOptions) bindFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.skipGoVersionCheck, "skip-go-version-check",
		false, "if specified, skip checking the Go version")

	// dependency args
	cmd.Flags().BoolVar(&o.fetchDeps, "fetch-deps", true, "ensure dependencies are downloaded")

	// deprecated dependency args
	cmd.Flags().BoolVar(&o.dep, "dep", true, "if specified, determines whether dep will be used.")
	o.depFlag = cmd.Flag("dep")
	cmd.Flags().StringArrayVar(&o.depArgs, "depArgs", nil, "additional arguments for dep")

	if err := cmd.Flags().MarkDeprecated("dep", "use the fetch-deps flag instead"); err != nil {
		log.Printf("error to mark dep flag as deprecated: %v", err)
	}
	if err := cmd.Flags().MarkDeprecated("depArgs", "will be removed with version 1 scaffolding"); err != nil {
		log.Printf("error to mark dep flag as deprecated: %v", err)
	}

	// boilerplate args
	cmd.Flags().StringVar(&o.license, "license", "apache2",
		"license to use to boilerplate, may be one of 'apache2', 'none'")
	cmd.Flags().StringVar(&o.owner, "owner", "", "owner to add to the copyright")

	// project args
	o.config = config.New(config.DefaultPath)
	cmd.Flags().StringVar(&o.config.Repo, "repo", "", "name to use for go module (e.g., github.com/user/repo), "+
		"defaults to the go package of the current working directory.")
	cmd.Flags().StringVar(&o.config.Domain, "domain", "my.domain", "domain for groups")
	cmd.Flags().StringVar(&o.config.Version, "project-version", config.DefaultVersion, "project version")
}

func (o *initOptions) loadConfig() (*config.Config, error) {
	_, err := config.Read()
	if err == nil || os.IsExist(err) {
		return nil, errors.New("already initialized")
	}

	return o.config, nil
}

func (o *initOptions) validate(c *config.Config) error {
	// Requires go1.11+
	if !o.skipGoVersionCheck {
		if err := internal.ValidateGoVersion(); err != nil {
			return err
		}
	}

	// Check if the project name is a valid namespace according to k8s
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error to get the current path: %v", err)
	}
	projectName := filepath.Base(dir)
	if err := internal.IsDNS1123Label(strings.ToLower(projectName)); err != nil {
		return fmt.Errorf("project name (%s) is invalid: %v", projectName, err)
	}

	// Try to guess repository if flag is not set
	if c.Repo == "" {
		repoPath, err := internal.FindCurrentRepo()
		if err != nil {
			return fmt.Errorf("error finding current repository: %v", err)
		}
		c.Repo = repoPath
	}

	// v1 only checks
	if c.IsV1() {
		// v1 is deprecated
		internal.PrintV1DeprecationWarning()

		// Verify dep is installed
		if _, err := exec.LookPath("dep"); err != nil {
			return fmt.Errorf("dep is not installed: %v\n"+
				"Follow steps at: https://golang.github.io/dep/docs/installation.html", err)
		}
	}

	return nil
}

func (o *initOptions) scaffolder(c *config.Config) (scaffold.Scaffolder, error) { //nolint:unparam
	return scaffold.NewInitScaffolder(c, o.license, o.owner), nil
}

func (o *initOptions) postScaffold(c *config.Config) error {
	switch {
	case c.IsV1():
		if !o.depFlag.Changed {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("Run `dep ensure` to fetch dependencies (Recommended) [y/n]?")
			o.dep = internal.YesNo(reader)
		}
		if !o.dep {
			fmt.Println("Skipping fetching dependencies.")
			return nil
		}

		err := internal.RunCmd("Fetching dependencies", "dep", append([]string{"ensure"}, o.depArgs...)...)
		if err != nil {
			return err
		}

	case c.IsV2():
		// Ensure that we are pinning controller-runtime version
		// xref: https://github.com/kubernetes-sigs/kubebuilder/issues/997
		err := internal.RunCmd("Get controller runtime", "go", "get",
			"sigs.k8s.io/controller-runtime@"+scaffold.ControllerRuntimeVersion)
		if err != nil {
			return err
		}

		err = internal.RunCmd("Update go.mod", "go", "mod", "tidy")
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown project version %v", c.Version)
	}

	err := internal.RunCmd("Running make", "make")
	if err != nil {
		return err
	}

	fmt.Println("Next: define a resource with:\n$ kubebuilder create api")
	return nil
}
