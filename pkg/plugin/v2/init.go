/*
Copyright 2020 The Kubernetes Authors.

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

package v2

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/pkg/internal/validation"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	"sigs.k8s.io/kubebuilder/pkg/plugin/internal"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
)

type initPlugin struct {
	config *config.Config
	// For help text.
	commandName string

	// boilerplate options
	license string
	owner   string

	// flags
	fetchDeps          bool
	skipGoVersionCheck bool
}

var (
	_ plugin.Init        = &initPlugin{}
	_ cmdutil.RunOptions = &initPlugin{}
)

func (p *initPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Initialize a new project including vendor/ directory and Go package directories.

Writes the following files:
- a boilerplate license file
- a PROJECT file with the domain and repo
- a Makefile to build the project
- a go.mod with project dependencies
- a Kustomization.yaml for customizating manifests
- a Patch file for customizing image for manager manifests
- a Patch file for enabling prometheus metrics
- a cmd/manager/main.go to run

project will prompt the user to run 'dep ensure' after writing the project files.
`
	ctx.Examples = fmt.Sprintf(`  # Scaffold a project using the apache2 license with "The Kubernetes authors" as owners
  %s init --project-version=2 --domain example.org --license apache2 --owner "The Kubernetes authors"
`,
		ctx.CommandName)

	p.commandName = ctx.CommandName
}

func (p *initPlugin) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.skipGoVersionCheck, "skip-go-version-check",
		false, "if specified, skip checking the Go version")

	// dependency args
	fs.BoolVar(&p.fetchDeps, "fetch-deps", true, "ensure dependencies are downloaded")

	// boilerplate args
	fs.StringVar(&p.license, "license", "apache2",
		"license to use to boilerplate, may be one of 'apache2', 'none'")
	fs.StringVar(&p.owner, "owner", "", "owner to add to the copyright")

	// project args
	fs.StringVar(&p.config.Repo, "repo", "", "name to use for go module (e.g., github.com/user/repo), "+
		"defaults to the go package of the current working directory.")
	fs.StringVar(&p.config.Domain, "domain", "my.domain", "domain for groups")
}

func (p *initPlugin) InjectConfig(c *config.Config) {
	// v3 project configs get a 'layout' value.
	if c.IsV3() {
		c.Layout = plugin.KeyFor(Plugin{})
	}
	p.config = c
}

func (p *initPlugin) Run() error {
	return cmdutil.Run(p)
}

func (p *initPlugin) Validate() error {
	// Requires go1.11+
	if !p.skipGoVersionCheck {
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
	if err := validation.IsDNS1123Label(strings.ToLower(projectName)); err != nil {
		return fmt.Errorf("project name (%s) is invalid: %v", projectName, err)
	}

	// Try to guess repository if flag is not set.
	if p.config.Repo == "" {
		repoPath, err := internal.FindCurrentRepo()
		if err != nil {
			return fmt.Errorf("error finding current repository: %v", err)
		}
		p.config.Repo = repoPath
	}

	return nil
}

func (p *initPlugin) GetScaffolder() (scaffold.Scaffolder, error) {
	return scaffold.NewInitScaffolder(p.config, p.license, p.owner), nil
}

func (p *initPlugin) PostScaffold() error {
	if !p.fetchDeps {
		fmt.Println("Skipping fetching dependencies.")
		return nil
	}

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

	err = internal.RunCmd("Running make", "make")
	if err != nil {
		return err
	}

	fmt.Printf("Next: define a resource with:\n$ %s create api\n", p.commandName)
	return nil
}
