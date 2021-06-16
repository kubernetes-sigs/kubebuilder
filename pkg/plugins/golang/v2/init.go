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

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
	"sigs.k8s.io/kubebuilder/v3/pkg/internal/validation"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config

	// For help text.
	commandName string

	// boilerplate options
	license string
	owner   string

	// config options
	domain string
	repo   string
	name   string

	// flags
	fetchDeps          bool
	skipGoVersionCheck bool
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	p.commandName = cliMeta.CommandName

	subcmdMeta.Description = `Initialize a new project including the following files:
  - a "go.mod" with project dependencies
  - a "PROJECT" file that stores project configuration
  - a "Makefile" with several useful make targets for the project
  - several YAML files for project deployment under the "config" directory
  - a "main.go" file that creates the manager that will run the project controllers
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a new project with your domain and name in copyright
  %[1]s init --plugins go/v2 --domain example.org --owner "Your name"

  # Initialize a new project defining an specific project version
  %[1]s init --plugins go/v2 --project-version 2
`, cliMeta.CommandName)
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.skipGoVersionCheck, "skip-go-version-check",
		false, "if specified, skip checking the Go version")

	// dependency args
	fs.BoolVar(&p.fetchDeps, "fetch-deps", true, "ensure dependencies are downloaded")

	// boilerplate args
	fs.StringVar(&p.license, "license", "apache2",
		"license to use to boilerplate, may be one of 'apache2', 'none'")
	fs.StringVar(&p.owner, "owner", "", "owner to add to the copyright")

	// project args
	fs.StringVar(&p.domain, "domain", "my.domain", "domain for groups")
	fs.StringVar(&p.repo, "repo", "", "name to use for go module (e.g., github.com/user/repo), "+
		"defaults to the go package of the current working directory.")
	fs.StringVar(&p.name, "project-name", "", "name of this project")
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	if err := p.config.SetDomain(p.domain); err != nil {
		return err
	}

	// Try to guess repository if flag is not set.
	if p.repo == "" {
		repoPath, err := golang.FindCurrentRepo()
		if err != nil {
			return fmt.Errorf("error finding current repository: %v", err)
		}
		p.repo = repoPath
	}
	if err := p.config.SetRepository(p.repo); err != nil {
		return err
	}

	if p.config.GetVersion().Compare(cfgv2.Version) > 0 {
		// Assign a default project name
		if p.name == "" {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting current directory: %v", err)
			}
			p.name = strings.ToLower(filepath.Base(dir))
		}
		// Check if the project name is a valid k8s namespace (DNS 1123 label).
		if err := validation.IsDNS1123Label(p.name); err != nil {
			return fmt.Errorf("project name (%s) is invalid: %v", p.name, err)
		}
		if err := p.config.SetProjectName(p.name); err != nil {
			return err
		}
	}

	return nil
}

func (p *initSubcommand) PreScaffold(machinery.Filesystem) error {
	// Validate the supported go versions
	if !p.skipGoVersionCheck {
		if err := golang.ValidateGoVersion(); err != nil {
			return err
		}
	}

	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewInitScaffolder(p.config, p.license, p.owner)
	scaffolder.InjectFS(fs)
	err := scaffolder.Scaffold()
	if err != nil {
		return err
	}

	if !p.fetchDeps {
		fmt.Println("Skipping fetching dependencies.")
		return nil
	}

	// Ensure that we are pinning controller-runtime version
	// xref: https://github.com/kubernetes-sigs/kubebuilder/issues/997
	err = util.RunCmd("Get controller runtime", "go", "get",
		"sigs.k8s.io/controller-runtime@"+scaffolds.ControllerRuntimeVersion)
	if err != nil {
		return err
	}

	return nil
}

func (p *initSubcommand) PostScaffold() error {
	err := util.RunCmd("Update dependencies", "go", "mod", "tidy")
	if err != nil {
		return err
	}

	fmt.Printf("Next: define a resource with:\n$ %s create api\n", p.commandName)
	return nil
}
