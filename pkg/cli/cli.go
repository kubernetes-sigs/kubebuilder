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
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	yamlstore "sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

const (
	noticeColor    = "\033[1;33m%s\033[0m"
	deprecationFmt = "[Deprecation Notice] %s\n\n"

	pluginsFlag        = "plugins"
	projectVersionFlag = "project-version"
)

// CLI is the command line utility that is used to scaffold kubebuilder project files.
type CLI struct { //nolint:maligned
	/* Fields set by Option */

	// Root command name. It is injected downstream to provide correct help, usage, examples and errors.
	commandName string
	// CLI version string.
	version string
	// CLI root's command description.
	description string
	// Plugins registered in the CLI.
	plugins map[string]plugin.Plugin
	// Default plugins in case none is provided and a config file can't be found.
	defaultPlugins map[config.Version][]string
	// Default project version in case none is provided and a config file can't be found.
	defaultProjectVersion config.Version
	// Commands injected by options.
	extraCommands []*cobra.Command
	// Alpha commands injected by options.
	extraAlphaCommands []*cobra.Command
	// Whether to add a completion command to the CLI.
	completionCommand bool

	/* Internal fields */

	// Plugin keys to scaffold with.
	pluginKeys []string
	// Project version to scaffold.
	projectVersion config.Version

	// A filtered set of plugins that should be used by command constructors.
	resolvedPlugins []plugin.Plugin

	// Root command.
	cmd *cobra.Command

	// Underlying fs
	fs machinery.Filesystem
}

// New creates a new CLI instance.
//
// It follows the functional options pattern in order to customize the resulting CLI.
//
// It returns an error if any of the provided options fails. As some processing needs
// to be done, execution errors may be found here. Instead of returning an error, this
// function will return a valid CLI that errors in Run so that help is provided to the
// user.
func New(options ...Option) (*CLI, error) {
	// Create the CLI.
	c, err := newCLI(options...)
	if err != nil {
		return nil, err
	}

	// Build the cmd tree.
	if err := c.buildCmd(); err != nil {
		c.cmd.RunE = errCmdFunc(err)
		return c, nil
	}

	// Add extra commands injected by options.
	if err := c.addExtraCommands(); err != nil {
		return nil, err
	}

	// Add extra alpha commands injected by options.
	if err := c.addExtraAlphaCommands(); err != nil {
		return nil, err
	}

	// Write deprecation notices after all commands have been constructed.
	c.printDeprecationWarnings()

	return c, nil
}

// newCLI creates a default CLI instance and applies the provided options.
// It is as a separate function for test purposes.
func newCLI(options ...Option) (*CLI, error) {
	// Default CLI options.
	c := &CLI{
		commandName: "kubebuilder",
		description: `CLI tool for building Kubernetes extensions and tools.
`,
		plugins:        make(map[string]plugin.Plugin),
		defaultPlugins: make(map[config.Version][]string),
		fs:             machinery.Filesystem{FS: afero.NewOsFs()},
	}

	// Apply provided options.
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// buildCmd creates the underlying cobra command and stores it internally.
func (c *CLI) buildCmd() error {
	c.cmd = c.newRootCmd()

	var uve config.UnsupportedVersionError

	// Get project version and plugin keys.
	switch err := c.getInfo(); {
	case err == nil:
	case errors.As(err, &uve) && uve.Version.Compare(config.Version{Number: 3, Stage: stage.Alpha}) == 0:
		// Check if the corresponding stable version exists, set c.projectVersion and break
		stableVersion := config.Version{
			Number: uve.Version.Number,
		}
		if config.IsRegistered(stableVersion) {
			// Use the stableVersion
			c.projectVersion = stableVersion
		} else {
			// stable version not registered, let's bail out
			return err
		}
	default:
		return err
	}

	// Resolve plugins for project version and plugin keys.
	if err := c.resolvePlugins(); err != nil {
		return err
	}

	// Add the subcommands
	c.addSubcommands()

	return nil
}

// getInfo obtains the plugin keys and project version resolving conflicts between the project config file and flags.
func (c *CLI) getInfo() error {
	// Get plugin keys and project version from project configuration file
	// We discard the error if file doesn't exist because not being able to read a project configuration
	// file is not fatal for some commands. The ones that require it need to check its existence later.
	hasConfigFile := true
	if err := c.getInfoFromConfigFile(); errors.Is(err, os.ErrNotExist) {
		hasConfigFile = false
	} else if err != nil {
		return err
	}

	// We can't early return here in case a project configuration file was found because
	// this command call may override the project plugins.

	// Get project version and plugin info from flags
	if err := c.getInfoFromFlags(hasConfigFile); err != nil {
		return err
	}

	// Get project version and plugin info from defaults
	c.getInfoFromDefaults()

	return nil
}

// getInfoFromConfigFile obtains the project version and plugin keys from the project config file.
func (c *CLI) getInfoFromConfigFile() error {
	// Read the project configuration file
	cfg := yamlstore.New(c.fs)
	if err := cfg.Load(); err != nil {
		return err
	}

	return c.getInfoFromConfig(cfg.Config())
}

// getInfoFromConfig obtains the project version and plugin keys from the project config.
// It is extracted from getInfoFromConfigFile for testing purposes.
func (c *CLI) getInfoFromConfig(projectConfig config.Config) error {
	c.pluginKeys = projectConfig.GetPluginChain()
	c.projectVersion = projectConfig.GetVersion()

	for _, pluginKey := range c.pluginKeys {
		if err := plugin.ValidateKey(pluginKey); err != nil {
			return fmt.Errorf("invalid plugin key found in project configuration file: %w", err)
		}
	}

	return nil
}

// getInfoFromFlags obtains the project version and plugin keys from flags.
func (c *CLI) getInfoFromFlags(hasConfigFile bool) error {
	// Partially parse the command line arguments
	fs := pflag.NewFlagSet("base", pflag.ContinueOnError)

	// Load the base command global flags
	fs.AddFlagSet(c.cmd.PersistentFlags())

	// If we were unable to load the project configuration, we should also accept the project version flag
	var projectVersionStr string
	if !hasConfigFile {
		fs.StringVar(&projectVersionStr, projectVersionFlag, "", "project version")
	}

	// FlagSet special cases --help and -h, so we need to create a dummy flag with these 2 values to prevent the default
	// behavior (printing the usage of this FlagSet) as we want to print the usage message of the underlying command.
	fs.BoolP("help", "h", false, fmt.Sprintf("help for %s", c.commandName))

	// Omit unknown flags to avoid parsing errors
	fs.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}

	// Parse the arguments
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	// If any plugin key was provided, replace those from the project configuration file
	if pluginKeys, err := fs.GetStringSlice(pluginsFlag); err != nil {
		return err
	} else if len(pluginKeys) != 0 {
		// Remove leading and trailing spaces and validate the plugin keys
		for i, key := range pluginKeys {
			pluginKeys[i] = strings.TrimSpace(key)
			if err := plugin.ValidateKey(pluginKeys[i]); err != nil {
				return fmt.Errorf("invalid plugin %q found in flags: %w", pluginKeys[i], err)
			}
		}

		c.pluginKeys = pluginKeys
	}

	// If the project version flag was accepted but not provided keep the empty version and try to resolve it later,
	// else validate the provided project version
	if projectVersionStr != "" {
		if err := c.projectVersion.Parse(projectVersionStr); err != nil {
			return fmt.Errorf("invalid project version flag: %w", err)
		}
	}

	return nil
}

// getInfoFromDefaults obtains the plugin keys, and maybe the project version from the default values
func (c *CLI) getInfoFromDefaults() {
	// Should not use default values if a plugin was already set
	// This checks includes the case where a project configuration file was found,
	// as it will always have at least one plugin key set by now
	if len(c.pluginKeys) != 0 {
		// We don't assign a default value for project version here because we may be able to
		// resolve the project version after resolving the plugins.
		return
	}

	// If the user provided a project version, use the default plugins for that project version
	if c.projectVersion.Validate() == nil {
		c.pluginKeys = c.defaultPlugins[c.projectVersion]
		return
	}

	// Else try to use the default plugins for the default project version
	if c.defaultProjectVersion.Validate() == nil {
		var found bool
		if c.pluginKeys, found = c.defaultPlugins[c.defaultProjectVersion]; found {
			c.projectVersion = c.defaultProjectVersion
			return
		}
	}

	// Else check if only default plugins for a project version were provided
	if len(c.defaultPlugins) == 1 {
		for projectVersion, defaultPlugins := range c.defaultPlugins {
			c.pluginKeys = defaultPlugins
			c.projectVersion = projectVersion
			return
		}
	}
}

const unstablePluginMsg = " (plugin version is unstable, there may be an upgrade available: " +
	"https://kubebuilder.io/migration/plugin/plugins.html)"

// resolvePlugins selects from the available plugins those that match the project version and plugin keys provided.
func (c *CLI) resolvePlugins() error {
	knownProjectVersion := c.projectVersion.Validate() == nil

	for _, pluginKey := range c.pluginKeys {
		var extraErrMsg string

		plugins := make([]plugin.Plugin, 0, len(c.plugins))
		for _, p := range c.plugins {
			plugins = append(plugins, p)
		}
		// We can omit the error because plugin keys have already been validated
		plugins, _ = plugin.FilterPluginsByKey(plugins, pluginKey)
		if knownProjectVersion {
			plugins = plugin.FilterPluginsByProjectVersion(plugins, c.projectVersion)
			extraErrMsg += fmt.Sprintf(" for project version %q", c.projectVersion)
		}

		// Plugins are often released as "unstable" (alpha/beta) versions, then upgraded to "stable".
		// This upgrade effectively removes a plugin, which is fine because unstable plugins are
		// under no support contract. However users should be notified _why_ their plugin cannot be found.
		if _, version := plugin.SplitKey(pluginKey); version != "" {
			var ver plugin.Version
			if err := ver.Parse(version); err != nil {
				return fmt.Errorf("error parsing input plugin version from key %q: %v", pluginKey, err)
			}
			if !ver.IsStable() {
				extraErrMsg += unstablePluginMsg
			}
		}

		// Only 1 plugin can match
		switch len(plugins) {
		case 1:
			c.resolvedPlugins = append(c.resolvedPlugins, plugins[0])
		case 0:
			return fmt.Errorf("no plugin could be resolved with key %q%s", pluginKey, extraErrMsg)
		default:
			return fmt.Errorf("ambiguous plugin %q%s", pluginKey, extraErrMsg)
		}
	}

	// Now we can try to resolve the project version if not known by this point
	if !knownProjectVersion && len(c.resolvedPlugins) > 0 {
		// Extract the common supported project versions
		supportedProjectVersions := plugin.CommonSupportedProjectVersions(c.resolvedPlugins...)

		// If there is only one common supported project version, resolve to it
	ProjectNumberVersionSwitch:
		switch len(supportedProjectVersions) {
		case 1:
			c.projectVersion = supportedProjectVersions[0]
		case 0:
			return fmt.Errorf("no project version supported by all the resolved plugins")
		default:
			supportedProjectVersionStrings := make([]string, 0, len(supportedProjectVersions))
			for _, supportedProjectVersion := range supportedProjectVersions {
				// In case one of the multiple supported versions is the default one, choose that and exit the switch
				if supportedProjectVersion.Compare(c.defaultProjectVersion) == 0 {
					c.projectVersion = c.defaultProjectVersion
					break ProjectNumberVersionSwitch
				}
				supportedProjectVersionStrings = append(supportedProjectVersionStrings,
					fmt.Sprintf("%q", supportedProjectVersion))
			}
			return fmt.Errorf("ambiguous project version, resolved plugins support the following project versions: %s",
				strings.Join(supportedProjectVersionStrings, ", "))
		}
	}

	return nil
}

// addSubcommands returns a root command with a subcommand tree reflecting the
// current project's state.
func (c *CLI) addSubcommands() {
	// add the alpha command if it has any subcommands enabled
	c.addAlphaCmd()

	// kubebuilder completion
	// Only add completion if requested
	if c.completionCommand {
		c.cmd.AddCommand(c.newCompletionCmd())
	}

	// kubebuilder create
	createCmd := c.newCreateCmd()
	// kubebuilder create api
	createCmd.AddCommand(c.newCreateAPICmd())
	createCmd.AddCommand(c.newCreateWebhookCmd())
	if createCmd.HasSubCommands() {
		c.cmd.AddCommand(createCmd)
	}

	// kubebuilder edit
	c.cmd.AddCommand(c.newEditCmd())

	// kubebuilder init
	c.cmd.AddCommand(c.newInitCmd())

	// kubebuilder version
	// Only add version if a version string was provided
	if c.version != "" {
		c.cmd.AddCommand(c.newVersionCmd())
	}
}

// addExtraCommands adds the additional commands.
func (c *CLI) addExtraCommands() error {
	for _, cmd := range c.extraCommands {
		for _, subCmd := range c.cmd.Commands() {
			if cmd.Name() == subCmd.Name() {
				return fmt.Errorf("command %q already exists", cmd.Name())
			}
		}
		c.cmd.AddCommand(cmd)
	}
	return nil
}

// printDeprecationWarnings prints the deprecation warnings of the resolved plugins.
func (c CLI) printDeprecationWarnings() {
	for _, p := range c.resolvedPlugins {
		if p != nil && p.(plugin.Deprecated) != nil && len(p.(plugin.Deprecated).DeprecationWarning()) > 0 {
			_, _ = fmt.Fprintf(os.Stderr, noticeColor, fmt.Sprintf(deprecationFmt, p.(plugin.Deprecated).DeprecationWarning()))
		}
	}
}

// metadata returns CLI's metadata.
func (c CLI) metadata() plugin.CLIMetadata {
	return plugin.CLIMetadata{
		CommandName: c.commandName,
	}
}

// Run executes the CLI utility.
//
// If an error is found, command help and examples will be printed.
func (c CLI) Run() error {
	return c.cmd.Execute()
}

// Command returns the underlying root command.
func (c CLI) Command() *cobra.Command {
	return c.cmd
}
