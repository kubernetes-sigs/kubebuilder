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
	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

func (c cli) newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project",
		Long: `Initialize a new project.

For further help about a specific project version, set --project-version.
`,
		Example: c.getInitHelpExamples(),
		Run:     func(cmd *cobra.Command, args []string) {},
	}

	// Register --project-version on the dynamically created command
	// so that it shows up in help and does not cause a parse error.
	cmd.Flags().String(projectVersionFlag, c.defaultProjectVersion.String(),
		fmt.Sprintf("project version, possible values: (%s)", strings.Join(c.getAvailableProjectVersions(), ", ")))
	// The --plugins flag can only be called to init projects v2+.
	if c.projectVersion.Compare(cfgv2.Version) == 1 {
		cmd.Flags().StringSlice(pluginsFlag, nil,
			"Name and optionally version of the plugin to initialize the project with. "+
				fmt.Sprintf("Available plugins: (%s)", strings.Join(c.getAvailablePlugins(), ", ")))
	}

	// Lookup the plugin for projectVersion and bind it to the command.
	c.bindInit(cmd)
	return cmd
}

func (c cli) getInitHelpExamples() string {
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

func (c cli) getAvailableProjectVersions() (projectVersions []string) {
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

func (c cli) getAvailablePlugins() (pluginKeys []string) {
	for key, p := range c.plugins {
		// Only return non-deprecated plugins.
		if _, isDeprecated := p.(plugin.Deprecated); !isDeprecated {
			pluginKeys = append(pluginKeys, strconv.Quote(key))
		}
	}
	sort.Strings(pluginKeys)
	return pluginKeys
}

func (c cli) bindInit(cmd *cobra.Command) {
	if len(c.resolvedPlugins) == 0 {
		cmdErr(cmd, fmt.Errorf("no resolved plugins, please specify plugins with --%s or/and --%s flags",
			projectVersionFlag, pluginsFlag))
		return
	}

	var initPlugin plugin.Init
	for _, p := range c.resolvedPlugins {
		tmpPlugin, isValid := p.(plugin.Init)
		if isValid {
			if initPlugin != nil {
				err := fmt.Errorf("duplicate initialization plugins (%s, %s), use a more specific plugin key",
					plugin.KeyFor(initPlugin), plugin.KeyFor(p))
				cmdErrNoHelp(cmd, err)
				return
			}
			initPlugin = tmpPlugin
		}
	}

	if initPlugin == nil {
		cmdErr(cmd, fmt.Errorf("resolved plugins do not provide a project init plugin: %v", c.pluginKeys))
		return
	}

	subcommand := initPlugin.GetInitSubcommand()

	meta := subcommand.UpdateMetadata(c.metadata())
	if meta.Description != "" {
		cmd.Long = meta.Description
	}
	if meta.Examples != "" {
		cmd.Example = meta.Examples
	}

	subcommand.BindFlags(cmd.Flags())

	cfg := yamlstore.New(c.fs)
	msg := fmt.Sprintf("failed to initialize project with %q", plugin.KeyFor(initPlugin))
	cmd.PreRunE = func(*cobra.Command, []string) error {
		// Check if a config is initialized.
		if err := cfg.Load(); err == nil || !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: already initialized", msg)
		}

		err := cfg.New(c.projectVersion)
		if err != nil {
			return fmt.Errorf("%s: error initializing project configuration: %w", msg, err)
		}

		subcommand.InjectConfig(cfg.Config())
		return nil
	}
	cmd.RunE = runECmdFunc(c.fs, subcommand, msg)
	cmd.PostRunE = postRunECmdFunc(cfg, msg)
}
