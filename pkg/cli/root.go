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
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var (
	supportedPlatforms = []string{"darwin", "linux"}
	// errHelpDisplayed is returned when help is displayed to prevent command execution
	errHelpDisplayed = errors.New("help displayed")
)

// isHelpFlag checks if the given string is a help flag
func isHelpFlag(s string) bool {
	return s == "--help" || s == "-h" || s == "help"
}

// getShortKey converts a full plugin key to a short display key
// Example: "deploy-image.go.kubebuilder.io/v1-alpha" -> "deploy-image/v1-alpha"
func getShortKey(fullKey string) string {
	name, version := plugin.SplitKey(fullKey)

	// Extract the short name (part before .kubebuilder.io or other domain)
	shortName := name
	if strings.Contains(name, ".kubebuilder.io") {
		shortName = strings.TrimSuffix(name, ".kubebuilder.io")
	} else if idx := strings.LastIndex(name, "."); idx > 0 {
		// For external plugins, try to get a reasonable short name
		// Keep the part before the last dot if it looks like a domain
		parts := strings.Split(name, ".")
		if len(parts) > 2 {
			shortName = strings.Join(parts[:len(parts)-1], ".")
		}
	}

	// Strip common suffixes for cleaner display
	// e.g., "deploy-image.go" -> "deploy-image", "kustomize.common" -> "kustomize"
	shortName = strings.TrimSuffix(shortName, ".go")
	shortName = strings.TrimSuffix(shortName, ".common")

	if version == "" {
		return shortName
	}
	return shortName + "/" + version
}

// getPluginDescription returns a short description for a plugin key
// This is a fallback for plugins that don't implement Describable interface
func getPluginDescription(_ string) string {
	// Fallback for external plugins that don't provide descriptions
	return "External or custom plugin"
}

func (c CLI) newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     c.commandName,
		Long:    c.description,
		Example: c.rootExamples(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Check if --plugins flag contains help flags (--help, -h, help)
			// This handles cases like: kubebuilder init --plugins --help
			if pluginKeys, err := cmd.Flags().GetStringSlice(pluginsFlag); err == nil {
				for _, key := range pluginKeys {
					key = strings.TrimSpace(key)
					if isHelpFlag(key) {
						// Help was requested, show help and stop execution
						cmd.SilenceUsage = true
						cmd.SilenceErrors = true
						_ = cmd.Help()
						return errHelpDisplayed
					}
				}
			}
			return nil
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
	str := fmt.Sprintf(`Get started by initializing a new project:

    %[1]s init --domain <YOUR_DOMAIN>

The default plugin scaffold includes everything you need. To use optional plugins:

    %[1]s init --plugins=<PLUGIN_KEYS>

Available plugins:

%[2]s

To see which plugins support a specific command:

    %[1]s <init|edit|create> --help
`,
		c.commandName, c.getPluginTable())

	if len(c.defaultPlugins) != 0 {
		if defaultPlugins, found := c.defaultPlugins[c.defaultProjectVersion]; found {
			str += fmt.Sprintf("\nDefault plugin: %q\n", strings.Join(defaultPlugins, ","))
		}
	}

	return str
}

// getPluginTable returns an ASCII table of the available plugins and their supported project versions.
func (c CLI) getPluginTable() string {
	return c.getPluginTableFiltered(nil)
}

// getPluginTableFilteredForSubcommand returns a filtered list of plugins for subcommands,
// excluding the default scaffold bundle and its component plugins.
func (c CLI) getPluginTableFilteredForSubcommand(filter func(plugin.Plugin) bool) string {
	return c.getPluginTableFilteredWithOptions(filter, true)
}

// getPluginTableFiltered returns a formatted list of plugins filtered by a predicate.
// If filter is nil, all plugins are included.
// Deprecated plugins are automatically excluded from help output.
func (c CLI) getPluginTableFiltered(filter func(plugin.Plugin) bool) string {
	return c.getPluginTableFilteredWithOptions(filter, false)
}

// getPluginTableFilteredWithOptions returns a formatted list of plugins with filtering options.
func (c CLI) getPluginTableFilteredWithOptions(filter func(plugin.Plugin) bool, excludeDefaultScaffold bool) string {
	type pluginInfo struct {
		shortKey    string
		fullKey     string
		description string
		versions    string
	}

	plugins := make([]pluginInfo, 0, len(c.plugins))

	for pluginKey, p := range c.plugins {
		// Skip deprecated plugins in help output
		if deprecated, ok := p.(plugin.Deprecated); ok {
			if deprecated.DeprecationWarning() != "" {
				continue
			}
		}

		// Apply filter if provided
		if filter != nil && !filter(p) {
			continue
		}

		// Skip base.go plugin to avoid duplication with go plugin
		if strings.Contains(pluginKey, "base.go.kubebuilder.io") {
			continue
		}

		// For subcommands, skip default scaffold and its component plugins
		if excludeDefaultScaffold {
			if pluginKey == "go.kubebuilder.io/v4" ||
				pluginKey == "kustomize.common.kubebuilder.io/v2" {
				continue
			}
		}

		shortKey := getShortKey(pluginKey)

		// Get description from plugin if it implements Describable, otherwise use fallback
		var desc string
		if describable, ok := p.(plugin.Describable); ok {
			desc = describable.Description()
		} else {
			desc = getPluginDescription(pluginKey)
		}

		// Get supported project versions
		supportedVersions := p.SupportedProjectVersions()
		versionStrs := make([]string, 0, len(supportedVersions))
		for _, ver := range supportedVersions {
			versionStrs = append(versionStrs, ver.String())
		}
		versionsStr := strings.Join(versionStrs, ", ")

		plugins = append(plugins, pluginInfo{
			shortKey:    shortKey,
			fullKey:     pluginKey,
			description: desc,
			versions:    versionsStr,
		})
	}

	if len(plugins) == 0 {
		return "No plugins available for this subcommand"
	}

	// Sort by short key for better readability
	slices.SortFunc(plugins, func(a, b pluginInfo) int {
		return strings.Compare(a.shortKey, b.shortKey)
	})

	// Calculate max width for KEY column
	maxKeyWidth := len("KEY")
	for _, p := range plugins {
		if len(p.shortKey) > maxKeyWidth {
			maxKeyWidth = len(p.shortKey)
		}
	}

	// Build aligned column output
	lines := make([]string, 0, len(plugins)+1)
	// Header
	lines = append(lines, fmt.Sprintf("  %-*s  %s", maxKeyWidth, "KEY", "DESCRIPTION"))
	// Entries
	for _, p := range plugins {
		lines = append(lines, fmt.Sprintf("  %-*s  %s", maxKeyWidth, p.shortKey, p.description))
	}

	return strings.Join(lines, "\n")
}
