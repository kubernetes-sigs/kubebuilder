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
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/external"
)

var retrievePluginsRoot = getPluginsRoot

// Option is a function used as arguments to New in order to configure the resulting CLI.
type Option func(*CLI) error

// WithCommandName is an Option that sets the CLI's root command name.
func WithCommandName(name string) Option {
	return func(c *CLI) error {
		c.commandName = name
		return nil
	}
}

// WithVersion is an Option that defines the version string of the CLI.
func WithVersion(version string) Option {
	return func(c *CLI) error {
		c.version = version
		return nil
	}
}

// WithDescription is an Option that sets the CLI's root description.
func WithDescription(description string) Option {
	return func(c *CLI) error {
		c.description = description
		return nil
	}
}

// WithPlugins is an Option that sets the CLI's plugins.
//
// Specifying any invalid plugin results in an error.
func WithPlugins(plugins ...plugin.Plugin) Option {
	return func(c *CLI) error {
		for _, p := range plugins {
			key := plugin.KeyFor(p)
			if _, isConflicting := c.plugins[key]; isConflicting {
				return fmt.Errorf("two plugins have the same key: %q", key)
			}
			if err := plugin.Validate(p); err != nil {
				return fmt.Errorf("broken pre-set plugin %q: %v", key, err)
			}
			c.plugins[key] = p
		}
		return nil
	}
}

// WithDefaultPlugins is an Option that sets the CLI's default plugins.
//
// Specifying any invalid plugin results in an error.
func WithDefaultPlugins(projectVersion config.Version, plugins ...plugin.Plugin) Option {
	return func(c *CLI) error {
		if err := projectVersion.Validate(); err != nil {
			return fmt.Errorf("broken pre-set project version %q for default plugins: %w", projectVersion, err)
		}
		if len(plugins) == 0 {
			return fmt.Errorf("empty set of plugins provided for project version %q", projectVersion)
		}
		for _, p := range plugins {
			if err := plugin.Validate(p); err != nil {
				return fmt.Errorf("broken pre-set default plugin %q: %v", plugin.KeyFor(p), err)
			}
			if !plugin.SupportsVersion(p, projectVersion) {
				return fmt.Errorf("default plugin %q doesn't support version %q", plugin.KeyFor(p), projectVersion)
			}
			c.defaultPlugins[projectVersion] = append(c.defaultPlugins[projectVersion], plugin.KeyFor(p))
		}
		return nil
	}
}

// WithDefaultProjectVersion is an Option that sets the CLI's default project version.
//
// Setting an invalid version results in an error.
func WithDefaultProjectVersion(version config.Version) Option {
	return func(c *CLI) error {
		if err := version.Validate(); err != nil {
			return fmt.Errorf("broken pre-set default project version %q: %v", version, err)
		}
		c.defaultProjectVersion = version
		return nil
	}
}

// WithExtraCommands is an Option that adds extra subcommands to the CLI.
//
// Adding extra commands that duplicate existing commands results in an error.
func WithExtraCommands(cmds ...*cobra.Command) Option {
	return func(c *CLI) error {
		// We don't know the commands defined by the CLI yet so we are not checking if the extra commands
		// conflict with a pre-existing one yet. We do this after creating the base commands.
		c.extraCommands = append(c.extraCommands, cmds...)
		return nil
	}
}

// WithExtraAlphaCommands is an Option that adds extra alpha subcommands to the CLI.
//
// Adding extra alpha commands that duplicate existing commands results in an error.
func WithExtraAlphaCommands(cmds ...*cobra.Command) Option {
	return func(c *CLI) error {
		// We don't know the commands defined by the CLI yet so we are not checking if the extra alpha commands
		// conflict with a pre-existing one yet. We do this after creating the base commands.
		c.extraAlphaCommands = append(c.extraAlphaCommands, cmds...)
		return nil
	}
}

// WithCompletion is an Option that adds the completion subcommand.
func WithCompletion() Option {
	return func(c *CLI) error {
		c.completionCommand = true
		return nil
	}
}

// WithFilesystem is an Option that allows to set the filesystem used in the CLI.
func WithFilesystem(filesystem machinery.Filesystem) Option {
	return func(c *CLI) error {
		if filesystem.FS == nil {
			return errors.New("invalid filesystem")
		}

		c.fs = filesystem
		return nil
	}
}

// parseExternalPluginArgs returns the program arguments.
func parseExternalPluginArgs() (args []string) {
	// Loop through os.Args and only get flags and their values that should be passed to the plugins
	// this also removes the --plugins flag and its values from the list passed to the external plugin
	for i := range os.Args {
		if strings.Contains(os.Args[i], "--") && !strings.Contains(os.Args[i], "--plugins") {
			args = append(args, os.Args[i])

			// Don't go out of bounds and don't append the next value if it is a flag
			if i+1 < len(os.Args) && !strings.Contains(os.Args[i+1], "--") {
				args = append(args, os.Args[i+1])
			}
		}
	}

	return args
}

// isHostSupported checks whether the host system is supported or not.
func isHostSupported(host string) bool {
	for _, platform := range supportedPlatforms {
		if host == platform {
			return true
		}
	}
	return false
}

// getPluginsRoot gets the plugin root path.
func getPluginsRoot(host string) (pluginsRoot string, err error) {
	if !isHostSupported(host) {
		// freebsd, openbsd, windows...
		return "", fmt.Errorf("host not supported: %v", host)
	}

	// if user provides specific path, return
	if pluginsPath := os.Getenv("EXTERNAL_PLUGINS_PATH"); pluginsPath != "" {
		// verify if the path actually exists
		if _, err = os.Stat(pluginsPath); err != nil {
			if os.IsNotExist(err) {
				// the path does not exist
				return "", fmt.Errorf("the specified path %s does not exist", pluginsPath)
			}
			// some other error
			return "", fmt.Errorf("error checking the path: %v", err)
		}
		// the path exists
		return pluginsPath, nil
	}

	// if no specific path, detects the host system and gets the plugins root based on the host.
	pluginsRelativePath := filepath.Join("kubebuilder", "plugins")
	if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		return filepath.Join(xdgHome, pluginsRelativePath), nil
	}

	switch host {
	case "darwin":
		logrus.Debugf("Detected host is macOS.")
		pluginsRoot = filepath.Join("Library", "Application Support", pluginsRelativePath)
	case "linux":
		logrus.Debugf("Detected host is Linux.")
		pluginsRoot = filepath.Join(".config", pluginsRelativePath)
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error retrieving home dir: %v", err)
	}

	return filepath.Join(userHomeDir, pluginsRoot), nil
}

// DiscoverExternalPlugins discovers the external plugins in the plugins root directory
// and adds them to external.Plugin.
func DiscoverExternalPlugins(filesystem afero.Fs) (ps []plugin.Plugin, err error) {
	pluginsRoot, err := retrievePluginsRoot(runtime.GOOS)
	if err != nil {
		logrus.Errorf("could not get plugins root: %v", err)
		return nil, err
	}

	rootInfo, err := filesystem.Stat(pluginsRoot)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			logrus.Debugf("External plugins dir %q does not exist, skipping external plugin parsing", pluginsRoot)
			return nil, nil
		}
		return nil, err
	}
	if !rootInfo.IsDir() {
		logrus.Debugf("External plugins path %q is not a directory, skipping external plugin parsing", pluginsRoot)
		return nil, nil
	}

	pluginInfos, err := afero.ReadDir(filesystem, pluginsRoot)
	if err != nil {
		return nil, err
	}

	for _, pluginInfo := range pluginInfos {
		if !pluginInfo.IsDir() {
			logrus.Debugf("%q is not a directory so skipping parsing", pluginInfo.Name())
			continue
		}

		versions, err := afero.ReadDir(filesystem, filepath.Join(pluginsRoot, pluginInfo.Name()))
		if err != nil {
			return nil, err
		}

		for _, version := range versions {
			if !version.IsDir() {
				logrus.Debugf("%q is not a directory so skipping parsing", version.Name())
				continue
			}

			pluginFiles, err := afero.ReadDir(filesystem, filepath.Join(pluginsRoot, pluginInfo.Name(), version.Name()))
			if err != nil {
				return nil, err
			}

			for _, pluginFile := range pluginFiles {
				// find the executable that matches the same name as info.Name().
				// if no match is found, compare the external plugin string name before dot
				// and match it with info.Name() which is the external plugin root dir.
				// for example: sample.sh --> sample, externalplugin.py --> externalplugin
				trimmedPluginName := strings.Split(pluginFile.Name(), ".")
				if trimmedPluginName[0] == "" {
					return nil, fmt.Errorf("Invalid plugin name found %q", pluginFile.Name())
				}

				if pluginFile.Name() == pluginInfo.Name() || trimmedPluginName[0] == pluginInfo.Name() {
					// check whether the external plugin is an executable.
					if !isPluginExecutable(pluginFile.Mode()) {
						return nil, fmt.Errorf("External plugin %q found in path is not an executable", pluginFile.Name())
					}

					ep := external.Plugin{
						PName:                     pluginInfo.Name(),
						Path:                      filepath.Join(pluginsRoot, pluginInfo.Name(), version.Name(), pluginFile.Name()),
						PSupportedProjectVersions: []config.Version{cfgv3.Version},
						Args:                      parseExternalPluginArgs(),
					}

					if err := ep.PVersion.Parse(version.Name()); err != nil {
						return nil, err
					}

					logrus.Printf("Adding external plugin: %s", ep.Name())

					ps = append(ps, ep)

				}
			}
		}

	}

	return ps, nil
}

// isPluginExecutable checks if a plugin is an executable based on the bitmask and returns true or false.
func isPluginExecutable(mode fs.FileMode) bool {
	return mode&0o111 != 0
}
