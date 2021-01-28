/*
Copyright 2021 The Kubernetes Authors.

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
	"io"
	"os"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	pluginKeysHeader      = "Plugin keys"
	projectVersionsHeader = "Supported project versions"
)

func (c cli) newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: c.commandName,
		Long: `CLI tool for building Kubernetes extensions and tools.
`,
		Example: c.rootExamples(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		PersistentPostRunE: c.persistChanges,
	}

	// Global flags for all subcommands
	// NOTE: the current plugin resolution doesn't allow to provide values to this flag different to those configured
	//       for the project, so default values need to be empty and considered when these two sources are compared.
	//       Another approach would be to allow users to overwrite the project configuration values. In this case, flags
	//       would take precedence over project configuration, which would take precedence over cli defaults.
	fs := cmd.PersistentFlags()
	fs.String(projectVersionFlag, "", "project version")
	fs.StringSlice(pluginsFlag, nil, "plugin keys of the plugin to initialize the project with")

	return cmd
}

// rootExamples builds the examples string for the root command
func (c cli) rootExamples() string {
	str := fmt.Sprintf(`The first step is to initialize your project:
    %[1]s init --project-version=<PROJECT VERSION> --plugins=<PLUGIN KEYS>

<PLUGIN KEYS> is a comma-separated list of plugin keys from the following table
and <PROJECT VERSION> a supported project version for these plugins.

%[2]s

For more specific help for the init command of a certain plugins and project version
configuration please run:
    %[1]s init --help --project-version=<PROJECT VERSION> --plugins=<PLUGIN KEYS>
`,
		c.commandName, c.getPluginTable())

	str += fmt.Sprintf("\nDefault project version: %s\n", c.defaultProjectVersion)

	if defaultPlugins, hasDefaultPlugins := c.defaultPlugins[c.defaultProjectVersion]; hasDefaultPlugins {
		str += fmt.Sprintf("Default plugin keys: %q\n", strings.Join(defaultPlugins, ","))
	}

	str += fmt.Sprintf(`
After the project has been initialized, run
    %[1]s --help
to obtain further info about available commands.`,
		c.commandName)

	return str
}

// getPluginTable returns an ASCII table of the available plugins and their supported project versions.
func (c cli) getPluginTable() string {
	var (
		maxPluginKeyLength      = len(pluginKeysHeader)
		pluginKeys              = make([]string, 0, len(c.plugins))
		maxProjectVersionLength = len(projectVersionsHeader)
		projectVersions         = make([]string, 0, len(c.plugins))
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
		projectVersions = append(projectVersions, supportedProjectVersionsStr)
	}

	lines := make([]string, 0, len(c.plugins)+2)
	lines = append(lines, fmt.Sprintf(" %[1]*[2]s | %[3]*[4]s",
		maxPluginKeyLength, pluginKeysHeader, maxProjectVersionLength, projectVersionsHeader))
	lines = append(lines, strings.Repeat("-", maxPluginKeyLength+2)+"+"+
		strings.Repeat("-", maxProjectVersionLength+2))
	for i, pluginKey := range pluginKeys {
		supportedProjectVersions := projectVersions[i]
		lines = append(lines, fmt.Sprintf(" %[1]*[2]s | %[3]*[4]s",
			maxPluginKeyLength, pluginKey, maxProjectVersionLength, supportedProjectVersions))
	}

	return strings.Join(lines, "\n")
}

func closeFile(f io.Closer) {
	_ = f.Close()
}

// persistChanges walks the scaffolded files in memory and persist them to disk
func (c cli) persistChanges(*cobra.Command, []string) error {
	err := afero.Walk(c.memory, ".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking memory filesystem at %q: %w", path, err)
		}

		if info.IsDir() {
			// Skip directories, creating files will create the needed directories
			return nil
		}

		var exists bool
		exists, err = afero.Exists(c.disk, path)
		if err != nil {
			return fmt.Errorf("unable to check if %q existed previously: %w", path, err)
		}

		if exists {
			var diskInfo os.FileInfo
			diskInfo, err = c.disk.Stat(path)
			if err != nil {
				return fmt.Errorf("unable to obtain info of %q: %w", path, err)
			}

			if os.SameFile(info, diskInfo) {
				// The file was unchanged, skip it
				return nil
			}
		}

		var source io.ReadCloser
		source, err = c.memory.Open(path)
		if err != nil {
			return fmt.Errorf("unable to read %q from memory: %w", path, err)
		}
		defer closeFile(source)

		var sink io.WriteCloser
		sink, err = c.disk.Create(path)
		if err != nil {
			return fmt.Errorf("unable to create/trucate %q file: %w", path, err)
		}
		defer closeFile(sink)

		_, err = io.Copy(sink, source)
		if err != nil {
			return fmt.Errorf("unable to update %q file: %w", path, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to persist scaffolded changes to disk: %w", err)
	}

	return nil
}
