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

package cli

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

const initErrorMsg = "failed to initialize project"

func (c CLI) newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   kubebuilderSubcommandInit,
		Short: "Initialize a new project",
		Long: `Initialize a new project.

For further help about a specific plugin, set --plugins.
`,
		Example: c.getInitHelpExamples(),
		Run:     func(_ *cobra.Command, _ []string) {},
	}

	// Register --project-version on the dynamically created command
	// so that it shows up in help and does not cause a parse error.
	cmd.Flags().String(projectVersionFlag, c.defaultProjectVersion.String(), projectVersionFlagDescription)

	// In case no plugin was resolved, instead of failing the construction of the CLI, fail the execution of
	// this subcommand. This allows the use of subcommands that do not require resolved plugins like help.
	if len(c.resolvedPlugins) == 0 {
		cmdErr(cmd, noResolvedPluginError{})
		return cmd
	}

	// Obtain the plugin keys and subcommands from the plugins that implement plugin.Init.
	subcommands := c.filterSubcommands(
		func(p plugin.Plugin) bool {
			_, isValid := p.(plugin.Init)
			return isValid
		},
		func(p plugin.Plugin) plugin.Subcommand {
			return p.(plugin.Init).GetInitSubcommand()
		},
	)

	// Verify that there is at least one remaining plugin.
	if len(subcommands) == 0 {
		cmdErr(cmd, noAvailablePluginError{"project initialization"})
		return cmd
	}

	c.applySubcommandHooks(cmd, subcommands, initErrorMsg, true)

	// Append plugin table after metadata updates
	c.appendPluginTable(cmd, func(p plugin.Plugin) bool {
		_, isValid := p.(plugin.Init)
		return isValid
	}, "Available plugins that support 'init'")

	return cmd
}

func (c CLI) getInitHelpExamples() string {
	projectVersionExample := c.defaultProjectVersion.String()
	if c.defaultProjectVersion.Validate() != nil {
		versions := c.getAvailableProjectVersions()
		if len(versions) == 0 {
			return fmt.Sprintf(`  # Initialize a new project
  %[1]s init --domain example.org

  # Initialize with optional plugins
  %[1]s init --domain example.org --plugins go/v4,helm/v2-alpha`, c.commandName)
		}
		projectVersionExample = versions[len(versions)-1].String()
	}

	return fmt.Sprintf(`  # Initialize a new project
  %[1]s init --domain example.org

  # Initialize with optional plugins
  %[1]s init --domain example.org --plugins go/v4,helm/v2-alpha

  # Initialize with a specific project config version
  %[1]s init --domain example.org --plugins go/v4 --project-version %[2]s`,
		c.commandName, projectVersionExample)
}

func (c CLI) getAvailableProjectVersions() (projectVersions []config.Version) {
	versionSet := make(map[config.Version]struct{})
	for _, p := range c.plugins {
		// Only return versions of non-deprecated plugins.
		if _, isDeprecated := p.(plugin.Deprecated); !isDeprecated {
			for _, version := range p.SupportedProjectVersions() {
				versionSet[version] = struct{}{}
			}
		}
	}
	for version := range versionSet {
		projectVersions = append(projectVersions, version)
	}
	slices.SortFunc(projectVersions, func(a, b config.Version) int {
		return a.Compare(b)
	})
	return projectVersions
}
