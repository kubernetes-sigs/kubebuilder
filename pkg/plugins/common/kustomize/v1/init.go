/*
Copyright 2021 The Kubernetes Authors.

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

package v1

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/internal/validation"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1/scaffolds"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config

	// config options
	domain          string
	name            string
	componentConfig bool
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Initialize a common project including the following files:
  - a "PROJECT" file that stores project configuration
  - several YAML files for project deployment under the "config" directory
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a common project with your domain and name in copyright
  %[1]s init --plugins common/v3 --domain example.org 

  # Initialize a common project defining an specific project version
  %[1]s init --plugins common/v3 --project-version 3
`, cliMeta.CommandName)
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&p.domain, "domain", "my.domain", "domain for groups")
	fs.StringVar(&p.name, "project-name", "", "name of this project")
	fs.BoolVar(&p.componentConfig, "component-config", false,
		"create a versioned ComponentConfig file, may be 'true' or 'false'")
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	if err := p.config.SetDomain(p.domain); err != nil {
		return err
	}

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

	if p.componentConfig {
		if err := p.config.SetComponentConfig(); err != nil {
			return err
		}
	}

	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewInitScaffolder(p.config)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return err
	}

	return nil
}
