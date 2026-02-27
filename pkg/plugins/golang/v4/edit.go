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

	// fs stores the FlagSet to check if flags were explicitly set
	fs *pflag.FlagSet
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Edit project configuration to enable or disable layout settings.
  
  WARNING:
	Webhooks and Namespace-Scoped Mode:
  	Webhooks remain cluster-scoped even in namespace-scoped mode.
  	The manager cache is restricted to WATCH_NAMESPACE, but webhooks receive requests
  	from ALL namespaces. You must configure namespaceSelector or objectSelector to align
  	webhook scope with the cache.

  Note: 
    To add optional plugins after initialization, use 'kubebuilder edit --plugins <plugin-name>'.
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
	p.fs = fs
	fs.BoolVar(&p.multigroup, "multigroup", false, "Enable/disable multi-group layout to organize APIs by group. "+
		"Changes API structure: api/<version>/ becomes api/<group>/<version>/ "+
		"Automatic: Updates PROJECT file, future APIs use new structure. "+
		"Manual: Move existing API files, update import paths in controllers. "+
		"More info: https://book.kubebuilder.io/migration/multi-group.html")
	fs.BoolVar(&p.namespaced, "namespaced", false, "Enable/disable namespace-scoped deployment. "+
		"Manager watches one or more specific namespaces rather than all namespaces. "+
		"Namespaces to watch are configured via WATCH_NAMESPACE environment variable. "+
		"Automatic: Updates PROJECT file, scaffolds Role/RoleBinding, uses --force to regenerate manager.yaml. "+
		"Manual: Add namespace= to RBAC markers in existing controllers, update cmd/main.go, run 'make manifests'. "+
		"More info: https://book.kubebuilder.io/migration/namespace-scoped.html . ")
	fs.BoolVar(&p.force, "force", false, "Overwrite existing scaffolded files to apply configuration changes. "+
		"Example: With --namespaced, regenerates config/manager/manager.yaml to add WATCH_NAMESPACE env var. "+
		"Warning: This overwrites default scaffold files; manual changes in those files may be lost.")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *editSubcommand) PreScaffold(machinery.Filesystem) error {
	// If flags were not explicitly set, preserve existing PROJECT file values
	// This prevents one flag from clearing another when using default values
	if !p.fs.Changed("multigroup") {
		p.multigroup = p.config.IsMultiGroup()
	}
	if !p.fs.Changed("namespaced") {
		p.namespaced = p.config.IsNamespaced()
	}

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
