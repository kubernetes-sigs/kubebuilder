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
	subcmdMeta.Description = `Edit project configuration to enable or disable layout settings.

Multigroup (--multigroup):
  Enable or disable multi-group layout.
  Changes API structure: api/<version>/ becomes api/<group>/<version>/
  Automatic: Updates PROJECT file, future APIs use new structure
  Manual: Move existing API files, update import paths in controllers
  Migration guide: https://book.kubebuilder.io/migration/multi-group.html

Namespaced (--namespaced):
  Enable or disable namespace-scoped deployment.
  Manager watches one or more specific namespaces vs all namespaces.
  Namespaces to watch are configured via WATCH_NAMESPACE environment variable.
  Automatic: Updates PROJECT file, scaffolds Role/RoleBinding, uses --force to regenerate manager.yaml
  Manual: Add namespace= to RBAC markers in existing controllers, update cmd/main.go, run 'make manifests'

Force (--force):
  Overwrite existing scaffolded files to apply configuration changes.
  Example: With --namespaced, regenerates config/manager/manager.yaml to add WATCH_NAMESPACE env var.
  Warning: This overwrites default scaffold files; manual changes in those files may be lost.

Note: To add optional plugins after initialization, use 'kubebuilder edit --plugins <plugin-name>'.
      Run 'kubebuilder edit --plugins --help' to see available plugins.
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Enable multigroup layout
  %[1]s edit --multigroup

  # Enable namespace-scoped permissions
  %[1]s edit --namespaced

  # Enable with automatic file regeneration
  %[1]s edit --namespaced --force

  # Disable multigroup layout
  %[1]s edit --multigroup=false

  # Enable/disable multiple settings
  %[1]s edit --multigroup --namespaced --force
`, cliMeta.CommandName)
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&p.multigroup, "multigroup", false, "enable or disable multigroup layout")
	fs.BoolVar(&p.namespaced, "namespaced", false, "enable or disable namespace-scoped deployment")
	fs.BoolVar(&p.force, "force", false, "overwrite scaffolded files to apply changes (manual edits may be lost)")
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
