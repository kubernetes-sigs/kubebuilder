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
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	yamlstore "sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

const initErrorMsg = "failed to initialize project"

func (c CLI) newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project",
		Long: `Initialize a new project.

For further help about a specific plugin, set --plugins.
`,
		Example: c.getInitHelpExamples(),
		Run:     func(cmd *cobra.Command, args []string) {},
	}

	// Register --project-version on the dynamically created command
	// so that it shows up in help and does not cause a parse error.
	cmd.Flags().String(projectVersionFlag, c.defaultProjectVersion.String(), "project version")

	// In case no plugin was resolved, instead of failing the construction of the CLI, fail the execution of
	// this subcommand. This allows the use of subcommands that do not require resolved plugins like help.
	if len(c.resolvedPlugins) == 0 {
		cmdErr(cmd, noResolvedPluginError{})
		return cmd
	}

	// Obtain the plugin keys and subcommands from the plugins that implement plugin.Init.
	pluginKeys, subcommands := c.filterSubcommands(
		func(p plugin.Plugin) bool {
			_, isValid := p.(plugin.Init)
			return isValid
		},
		func(p plugin.Plugin) plugin.Subcommand {
			return p.(plugin.Init).GetInitSubcommand()
		},
	)

	// Verify that there is at least one remaining plugin.
	if len(*subcommands) == 0 {
		cmdErr(cmd, noAvailablePluginError{"project initialization"})
		return cmd
	}

	// Initialization methods.
	_ = c.initializationMethods(cmd, subcommands)

	// Execution methods.
	cfg := yamlstore.New(c.fs)
	cmd.PreRunE = func(*cobra.Command, []string) error {
		// Check if a config is initialized.
		if err := cfg.Load(); err == nil || !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: already initialized", initErrorMsg)
		}

		err := cfg.New(c.projectVersion)
		if err != nil {
			return fmt.Errorf("%s: error initializing project configuration: %w", initErrorMsg, err)
		}

		// We extract the plugin keys again instead of using the ones obtained when filtering subcommands
		// as there plugins are unbundled but we want to keep bundle names in the layout.
		resolvedPluginKeys := make([]string, 0, len(c.resolvedPlugins))
		for _, p := range c.resolvedPlugins {
			resolvedPluginKeys = append(resolvedPluginKeys, plugin.KeyFor(p))
		}
		_ = cfg.Config().SetLayout(strings.Join(resolvedPluginKeys, ","))

		// Inject config method.
		subcommandsCopy := make([]plugin.Subcommand, len(*subcommands))
		copy(subcommandsCopy, *subcommands)
		for i, subcommand := range subcommandsCopy {
			if subcmd, requiresConfig := subcommand.(plugin.RequiresConfig); requiresConfig {
				if err := subcmd.InjectConfig(cfg.Config()); err != nil {
					var exitError plugin.ExitError
					if errors.As(err, &exitError) {
						fmt.Printf("skipping %q: %s\n", pluginKeys[i], exitError.Reason)
						*subcommands = append((*subcommands)[:i], (*subcommands)[i+1:]...)
					} else {
						return fmt.Errorf("%s: unable to inject the configuration to %q: %w",
							initErrorMsg, pluginKeys[i], err)
					}
				}
			}
		}

		// Pre-scaffold method.
		subcommandsCopy = make([]plugin.Subcommand, len(*subcommands))
		copy(subcommandsCopy, *subcommands)
		for i, subcommand := range subcommandsCopy {
			if subcmd, hasPreScaffold := subcommand.(plugin.HasPreScaffold); hasPreScaffold {
				if err := subcmd.PreScaffold(c.fs); err != nil {
					var exitError plugin.ExitError
					if errors.As(err, &exitError) {
						fmt.Printf("skipping %q: %s\n", pluginKeys[i], exitError.Reason)
						*subcommands = append((*subcommands)[:i], (*subcommands)[i+1:]...)
					} else {
						return fmt.Errorf("%s: unable to run pre-scaffold tasks of %q: %w",
							initErrorMsg, pluginKeys[i], err)
					}
				}
			}
		}

		return nil
	}
	cmd.RunE = executionMethodsRunEFunc(pluginKeys, subcommands, c.fs, initErrorMsg)
	cmd.PostRunE = executionMethodsPostRunEFunc(pluginKeys, subcommands, cfg, initErrorMsg)

	return cmd
}

func (c CLI) getInitHelpExamples() string {
	var sb strings.Builder
	for _, version := range c.getAvailableProjectVersions() {
		rendered := fmt.Sprintf(`  # Help for initializing a project with version %[2]s
  %[1]s init --project-version=%[2]s -h

`,
			c.commandName, version)
		sb.WriteString(rendered)
	}
	return strings.TrimSuffix(sb.String(), "\n\n")
}

func (c CLI) getAvailableProjectVersions() (projectVersions []string) {
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
		projectVersions = append(projectVersions, strconv.Quote(version.String()))
	}
	sort.Strings(projectVersions)
	return projectVersions
}

func (c CLI) getAvailablePlugins() (pluginKeys []string) {
	for key, p := range c.plugins {
		// Only return non-deprecated plugins.
		if _, isDeprecated := p.(plugin.Deprecated); !isDeprecated {
			pluginKeys = append(pluginKeys, strconv.Quote(key))
		}
	}
	sort.Strings(pluginKeys)
	return pluginKeys
}
