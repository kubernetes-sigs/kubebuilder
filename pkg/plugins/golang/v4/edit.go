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

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config config.Config

	multigroup bool
	namespaced bool
	force      bool
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
`, cliMeta.CommandName)
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.multigroup, "multigroup", false, "enable or disable multigroup layout")
	fs.BoolVar(&p.namespaced, "namespaced", false, "enable or disable namespaced layout")
	fs.BoolVar(&p.force, "force", false, "overwrite existing files (regenerates manager.yaml with WATCH_NAMESPACE)")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewEditScaffolder(p.config, p.multigroup, p.namespaced, p.force)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to edit scaffold: %w", err)
	}

	return nil
}
