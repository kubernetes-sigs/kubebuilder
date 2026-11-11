/*
Copyright 2026 The Kubernetes Authors.

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

package v1alpha

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/multicluster-runtime/v1alpha/scaffolds"
)

var _ plugin.EditSubcommand = &editSubcommand{}

type editSubcommand struct {
	config        config.Config
	provider      string
	kubeconfigDir string
}

func (p *editSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Switch the multicluster provider used in cmd/main.go.

Rewrites cmd/main.go while preserving all +kubebuilder:scaffold markers so that
future kubebuilder create api and create webhook commands continue to work.`
	subcmdMeta.Examples = fmt.Sprintf(`  # Switch to namespace provider
  %[1]s edit --plugins %[2]s --provider namespace`, cliMeta.CommandName, plugin.KeyFor(Plugin{}))
}

func (p *editSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&p.provider, "provider", "kubeconfig",
		fmt.Sprintf("Switch the multicluster provider (%s)", strings.Join(validProviders, "|")))
	fs.StringVar(&p.kubeconfigDir, "kubeconfig-dir", "/etc/kubeconfig",
		"Directory of per-cluster kubeconfig files (file provider only)")
}

func (p *editSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *editSubcommand) Scaffold(fs machinery.Filesystem) error {
	if err := validateProvider(p.provider); err != nil {
		return err
	}
	s := scaffolds.NewEditScaffolder(p.config, p.provider, p.kubeconfigDir)
	s.InjectFS(fs)
	if err := s.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold edit: %w", err)
	}
	return nil
}

// PostScaffold ensures sigs.k8s.io/multicluster-runtime is present and tidies go.mod.
func (p *editSubcommand) PostScaffold() error {
	if err := pluginutil.RunCmd("Get multicluster-runtime", "go", "get",
		"sigs.k8s.io/multicluster-runtime@"+scaffolds.MulticlusterRuntimeVersion); err != nil {
		return fmt.Errorf("error getting multicluster-runtime: %w", err)
	}
	// The cluster-api and file providers live in separate Go modules and require
	// an additional go get to resolve their import paths.
	switch p.provider {
	case "cluster-api":
		// Pin both the provider and the exact cluster-api version it requires.
		// go/v4's `go mod tidy` may have resolved cluster-api to a newer, incompatible
		// version before this PostScaffold runs.
		if err := pluginutil.RunCmd("Get cluster-api provider", "go", "get",
			"sigs.k8s.io/multicluster-runtime/providers/cluster-api@"+scaffolds.MulticlusterRuntimeVersion,
			"sigs.k8s.io/cluster-api@"+scaffolds.ClusterAPIVersion); err != nil {
			return fmt.Errorf("error getting cluster-api provider: %w", err)
		}
	case "file":
		if err := pluginutil.RunCmd("Get file provider", "go", "get",
			"sigs.k8s.io/multicluster-runtime/providers/file@"+scaffolds.MulticlusterRuntimeVersion); err != nil {
			return fmt.Errorf("error getting file provider: %w", err)
		}
	}
	if err := pluginutil.RunCmd("Update dependencies", "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("error updating go dependencies: %w", err)
	}
	return nil
}
