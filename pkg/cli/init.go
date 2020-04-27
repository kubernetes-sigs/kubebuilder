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
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	internalconfig "sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

func (c *cli) newInitCmd() *cobra.Command {
	ctx := c.newInitContext()
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialize a new project",
		Long:    ctx.Description,
		Example: ctx.Examples,
		Run:     func(cmd *cobra.Command, args []string) {},
	}

	// Register --project-version on the dynamically created command
	// so that it shows up in help and does not cause a parse error.
	cmd.Flags().String(projectVersionFlag, c.defaultProjectVersion,
		fmt.Sprintf("project version, possible values: (%s)", strings.Join(c.getAvailableProjectVersions(), ", ")))
	// The --plugins flag can only be called to init projects v2+.
	if c.projectVersion != config.Version1 {
		cmd.Flags().StringSlice(pluginsFlag, nil,
			"Name and optionally version of the plugin to initialize the project with. "+
				fmt.Sprintf("Available plugins: (%s)", strings.Join(c.getAvailablePlugins(), ", ")))
	}

	// If only the help flag was set, return the command as is.
	if c.doGenericHelp {
		return cmd
	}

	// Lookup the plugin for projectVersion and bind it to the command.
	c.bindInit(ctx, cmd)
	return cmd
}

func (c cli) newInitContext() plugin.Context {
	return plugin.Context{
		CommandName: c.commandName,
		Description: `Initialize a new project.

For further help about a specific project version, set --project-version.
`,
		Examples: c.getInitHelpExamples(),
	}
}

func (c cli) getInitHelpExamples() string {
	var sb strings.Builder
	for _, version := range c.getAvailableProjectVersions() {
		rendered := fmt.Sprintf(`  # Help for initializing a project with version %s
  %s init --project-version=%s -h

`,
			version, c.commandName, version)
		sb.WriteString(rendered)
	}
	return strings.TrimSuffix(sb.String(), "\n\n")
}

func (c cli) getAvailableProjectVersions() (projectVersions []string) {
	versionSet := make(map[string]struct{})
	for version, versionedPlugins := range c.pluginsFromOptions {
		for _, p := range versionedPlugins {
			// If there's at least one non-deprecated plugin per version, that
			// version is "available".
			if _, isDeprecated := p.(plugin.Deprecated); !isDeprecated {
				versionSet[version] = struct{}{}
				break
			}
		}
	}
	for version := range versionSet {
		projectVersions = append(projectVersions, strconv.Quote(version))
	}
	return projectVersions
}

func (c cli) getAvailablePlugins() (pluginKeys []string) {
	for _, versionedPlugins := range c.pluginsFromOptions {
		for _, p := range versionedPlugins {
			// Only return non-deprecated plugins.
			if _, isDeprecated := p.(plugin.Deprecated); !isDeprecated {
				pluginKeys = append(pluginKeys, strconv.Quote(plugin.KeyFor(p)))
			}
		}
	}
	return pluginKeys
}

func (c cli) bindInit(ctx plugin.Context, cmd *cobra.Command) {
	var getter plugin.InitPluginGetter
	for _, p := range c.resolvedPlugins {
		tmpGetter, isGetter := p.(plugin.InitPluginGetter)
		if isGetter {
			if getter != nil {
				log.Fatalf("duplicate initialization plugins for project version %q: %s, %s",
					c.projectVersion, getter.Name(), p.Name())
			}
			getter = tmpGetter
		}
	}
	if getter == nil {
		if c.cliPluginKey == "" {
			log.Fatalf("project version %q does not support an initialization plugin", c.projectVersion)
		} else {
			log.Fatalf("plugin %q does not support an initialization plugin", c.cliPluginKey)
		}
	}

	cfg := internalconfig.New(internalconfig.DefaultPath)
	cfg.Version = c.projectVersion

	init := getter.GetInitPlugin()
	init.InjectConfig(&cfg.Config)
	init.BindFlags(cmd.Flags())
	init.UpdateContext(&ctx)
	cmd.Long = ctx.Description
	cmd.Example = ctx.Examples
	cmd.RunE = func(*cobra.Command, []string) error {
		// Check if a config is initialized in the command runner so the check
		// doesn't erroneously fail other commands used in initialized projects.
		_, err := internalconfig.Read()
		if err == nil || os.IsExist(err) {
			log.Fatal("config already initialized")
		}
		if err := init.Run(); err != nil {
			return fmt.Errorf("failed to initialize project with version %q: %v", c.projectVersion, err)
		}
		return cfg.Save()
	}
}
