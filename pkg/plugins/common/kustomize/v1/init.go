/*
Copyright 2022 The Kubernetes Authors.

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

//go:deprecated This package has been deprecated in favor of v2
package v1

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/internal/validation"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1/scaffolds"
)

var _ plugin.InitSubcommand = &initSubcommand{}

// Verify if the local environment is supported by this plugin
var supportedArchs = []string{"linux/amd64",
	"linux/arm64",
	"darwin/amd64"}

type initSubcommand struct {
	config config.Config

	// config options
	domain          string
	name            string
	componentConfig bool
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = fmt.Sprintf(`Initialize a common project including the following files:
  - a "PROJECT" file that stores project configuration
  - several YAML files for project deployment under the "config" directory

  NOTE: The kustomize/v1 plugin used to do this scaffold uses the v3 release (%s).
Therefore, darwin/arm64 is not supported since Kustomize does not provide v3
binaries for this architecture. The currently supported architectures are %q. 
More info: https://github.com/kubernetes-sigs/kustomize/issues/4612.

`, KustomizeVersion, supportedArchs)

	subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a common project with your domain and name in copyright
  %[1]s init --plugins common/v3 --domain example.org

  # Initialize a common project defining a specific project version
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

func (p *initSubcommand) PreScaffold(machinery.Filesystem) error {
	arch := runtime.GOARCH
	// It probably will never return x86_64. However, we are here checking the support for the binaries
	// So that, x86_64 means getting the Linux/amd64 binary. Then, we just keep this line to ensure
	// that it complies with the same code implementation that we have in the targets. In case someone
	// call the command inform the GOARCH=x86_64 then, we will properly handle the scenario
	// since it will work successfully and will instal the Linux/amd64 binary via the Makefile target.
	arch = strings.Replace(arch, "x86_64", "amd64", -1)
	localPlatform := fmt.Sprintf("%s/%s", strings.TrimSpace(runtime.GOOS), strings.TrimSpace(arch))

	if !hasSupportFor(localPlatform) {
		log.Warnf("the platform of this environment (%s) is not suppported by kustomize v3 (%s) which is "+
			"used in this scaffold. You will be unable to download a binary for the kustomize version supported "+
			"and used by this plugin. The currently supported platforms are: %q",
			localPlatform,
			KustomizeVersion,
			supportedArchs)
	}

	return nil
}

func hasSupportFor(localPlatform string) bool {
	for _, value := range supportedArchs {
		if value == localPlatform {
			return true
		}
	}
	return false
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewInitScaffolder(p.config)
	scaffolder.InjectFS(fs)
	return scaffolder.Scaffold()
}
