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

package v4

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config config.Config

	multigroup  bool
	namespaced  bool
	force       bool
	licenseFile string
	license     string
	owner       string
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Edit the project configuration.

Features:
  - Toggle multigroup layout (organize APIs by group).
  - Toggle namespaced layout (namespace-scoped vs cluster-scoped).

Namespaced layout (--namespaced):
  Changes namespace-scoped (watches specific namespaces) vs cluster-scoped (watches all namespaces).
  What changes automatically:
    - Updates PROJECT file (namespaced: true)
    - Scaffolds Role/RoleBinding instead of ClusterRole/ClusterRoleBinding
    - With --force: Regenerates config/manager/manager.yaml with WATCH_NAMESPACE env var
  What you must update manually:
    - Add namespace= to RBAC markers in existing controllers (new controllers get this automatically)
    - Update cmd/main.go to use namespace-scoped cache
    - Run: make manifests
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Enable multigroup layout
  %[1]s edit --multigroup

  # Disable multigroup layout
  %[1]s edit --multigroup=false

  # Enable namespaced layout (--force regenerates config/manager/manager.yaml with WATCH_NAMESPACE)
  %[1]s edit --namespaced --force

  # Enable namespaced layout without force (manually update config/manager/manager.yaml)
  %[1]s edit --namespaced

  # Disable namespaced layout (--force regenerates config/manager/manager.yaml without WATCH_NAMESPACE)
  %[1]s edit --namespaced=false --force

  # Update license header from custom file
  %[1]s edit --license-file ./my-header.txt

  # Update license header to built-in apache2
  %[1]s edit --license apache2 --owner "Your Company"
`, cliMeta.CommandName)
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.multigroup, "multigroup", false, "enable or disable multigroup layout")
	fs.BoolVar(&p.namespaced, "namespaced", false, "enable or disable namespaced layout")
	fs.BoolVar(&p.force, "force", false,
		"overwrite existing files (regenerates manager.yaml with WATCH_NAMESPACE)")
	fs.StringVar(&p.licenseFile, "license-file", "",
		"path to custom license file; content copied to hack/boilerplate.go.txt")
	fs.StringVar(&p.license, "license", "",
		"license to use to boilerplate, may be one of 'apache2', 'none'"+
			" (see: https://book.kubebuilder.io/reference/license-header)")
	fs.StringVar(&p.owner, "owner", "", "owner to add to the copyright")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *editSubcommand) PreScaffold(machinery.Filesystem) error {
	// Trim whitespace from license file path and treat empty/whitespace-only as not provided
	p.licenseFile = strings.TrimSpace(p.licenseFile)

	// Validate license file exists and has proper format before scaffolding begins to prevent errors mid-operation
	if p.licenseFile != "" {
		if _, err := os.Stat(p.licenseFile); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("license file %q does not exist", p.licenseFile)
			}
			return fmt.Errorf("failed to access license file %q: %w", p.licenseFile, err)
		}

		// Validate that the license file is a valid Go comment block
		content, err := os.ReadFile(p.licenseFile)
		if err != nil {
			return fmt.Errorf("failed to read license file %q: %w", p.licenseFile, err)
		}

		// Only validate format if file is not empty (empty files are allowed)
		if len(content) > 0 {
			contentStr := strings.TrimSpace(string(content))
			if !strings.HasPrefix(contentStr, "/*") || !strings.HasSuffix(contentStr, "*/") {
				return fmt.Errorf("license file %q must be a valid Go comment block (start with /* and end with */)", p.licenseFile)
			}
		}
	}

	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewEditScaffolder(p.config, p.multigroup, p.namespaced, p.force,
		p.license, p.owner, p.licenseFile)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to edit scaffold: %w", err)
	}

	return nil
}
