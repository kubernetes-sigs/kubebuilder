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

package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const (
	pluginKeysHeader      = "Plugin keys"
	projectVersionsHeader = "Supported project versions"
)

var supportedPlatforms = []string{"darwin", "linux"}

func (c CLI) newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     c.commandName,
		Long:    c.description,
		Example: c.rootExamples(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	// Global flags for all subcommands.
	cmd.PersistentFlags().StringSlice(pluginsFlag, nil, "plugin keys to be used for this subcommand execution")

	// Register --project-version on the root command so that it shows up in help.
	cmd.Flags().String(projectVersionFlag, c.defaultProjectVersion.String(), "project version")

	// As the root command will be used to shot the help message under some error conditions,
	// like during plugin resolving, we need to allow unknown flags to prevent parsing errors.
	cmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}

	return cmd
}

// rootExamples builds the examples string for the root command before resolving plugins
func (c CLI) rootExamples() string {
	str := fmt.Sprintf(`The first step is to initialize your project:
    %[1]s init [--plugins=<PLUGIN KEYS> [--project-version=<PROJECT VERSION>]]

<PLUGIN KEYS> is a comma-separated list of plugin keys from the following table
and <PROJECT VERSION> a supported project version for these plugins.

%[2]s

For more specific help for the init command of a certain plugins and project version
configuration please run:
    %[1]s init --help --plugins=<PLUGIN KEYS> [--project-version=<PROJECT VERSION>]
`,
		c.commandName, c.getPluginTable())

	if len(c.defaultPlugins) != 0 {
		if defaultPlugins, found := c.defaultPlugins[c.defaultProjectVersion]; found {
			str += fmt.Sprintf("\nDefault plugin keys: %q\n", strings.Join(defaultPlugins, ","))
		}
	}

	if c.defaultProjectVersion.Validate() == nil {
		str += fmt.Sprintf("Default project version: %q\n", c.defaultProjectVersion)
	}

	return str
}

// getPluginTable returns an ASCII table of the available plugins and their supported project versions.
func (c CLI) getPluginTable() string {
	var (
		maxPluginKeyLength      = len(pluginKeysHeader)
		pluginKeys              = make([]string, 0, len(c.plugins))
		maxProjectVersionLength = len(projectVersionsHeader)
		projectVersions         = make(map[string]string, len(c.plugins))
	)

	for pluginKey, plugin := range c.plugins {
		if len(pluginKey) > maxPluginKeyLength {
			maxPluginKeyLength = len(pluginKey)
		}
		pluginKeys = append(pluginKeys, pluginKey)
		supportedProjectVersions := plugin.SupportedProjectVersions()
		supportedProjectVersionStrs := make([]string, 0, len(supportedProjectVersions))
		for _, version := range supportedProjectVersions {
			supportedProjectVersionStrs = append(supportedProjectVersionStrs, version.String())
		}
		supportedProjectVersionsStr := strings.Join(supportedProjectVersionStrs, ", ")
		if len(supportedProjectVersionsStr) > maxProjectVersionLength {
			maxProjectVersionLength = len(supportedProjectVersionsStr)
		}
		projectVersions[pluginKey] = supportedProjectVersionsStr
	}

	lines := make([]string, 0, len(c.plugins)+2)
	lines = append(lines, fmt.Sprintf(" %[1]*[2]s | %[3]*[4]s",
		maxPluginKeyLength, pluginKeysHeader, maxProjectVersionLength, projectVersionsHeader))
	lines = append(lines, strings.Repeat("-", maxPluginKeyLength+2)+"+"+
		strings.Repeat("-", maxProjectVersionLength+2))

	sort.Strings(pluginKeys)
	for _, pluginKey := range pluginKeys {
		supportedProjectVersions := projectVersions[pluginKey]
		lines = append(lines, fmt.Sprintf(" %[1]*[2]s | %[3]*[4]s",
			maxPluginKeyLength, pluginKey, maxProjectVersionLength, supportedProjectVersions))
	}

	return strings.Join(lines, "\n")
}
