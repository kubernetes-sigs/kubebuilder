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
	"os"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	internalconfig "sigs.k8s.io/kubebuilder/v3/pkg/cli/internal/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

const (
	noticeColor    = "\033[1;36m%s\033[0m"
	deprecationFmt = "[Deprecation Notice] %s\n\n"

	projectVersionFlag = "project-version"
	pluginsFlag        = "plugins"

	noPluginError = "invalid config file please verify that the version and layout fields are set and valid"
)

// equalStringSlice checks if two string slices are equal.
func equalStringSlice(a, b []string) bool {
	// Check lengths
	if len(a) != len(b) {
		return false
	}

	// Check elements
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

// CLI interacts with a command line interface.
type CLI interface {
	// Run runs the CLI, usually returning an error if command line configuration
	// is incorrect.
	Run() error
}

// cli defines the command line structure and interfaces that are used to
// scaffold kubebuilder project files.
type cli struct { //nolint:maligned
	/* Fields set by Option */

	// Root command name. It is injected downstream to provide correct help, usage, examples and errors.
	commandName string
	// CLI version string.
	version string
	// Default project version in case none is provided and a config file can't be found.
	defaultProjectVersion config.Version
	// Default plugins in case none is provided and a config file can't be found.
	defaultPlugins map[config.Version][]string
	// Plugins registered in the cli.
	plugins map[string]plugin.Plugin
	// Commands injected by options.
	extraCommands []*cobra.Command
	// Whether to add a completion command to the cli.
	completionCommand bool

	/* Internal fields */

	// Project version to scaffold.
	projectVersion config.Version
	// Plugin keys to scaffold with.
	pluginKeys []string

	// A filtered set of plugins that should be used by command constructors.
	resolvedPlugins []plugin.Plugin

	// Root command.
	cmd *cobra.Command

	// Underlying fs
	fs afero.Fs
}

// New creates a new cli instance.
// Developer errors (e.g. not registering any plugins, extra commands with conflicting names) return an error
// while user errors (e.g. errors while parsing flags, unresolvable plugins) create a command which return the error.
func New(opts ...Option) (CLI, error) {
	// Create the CLI.
	c, err := newCLI(opts...)
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

	// Write deprecation notices after all commands have been constructed.
	c.printDeprecationWarnings()

	return c, nil
}

// newCLI creates a default cli instance and applies the provided options.
// It is as a separate function for test purposes.
func newCLI(opts ...Option) (*cli, error) {
	// Default cli options.
	c := &cli{
		commandName:           "kubebuilder",
		defaultProjectVersion: cfgv3.Version,
		defaultPlugins:        make(map[config.Version][]string),
		plugins:               make(map[string]plugin.Plugin),
		fs:                    afero.NewOsFs(),
	}

	// Apply provided options.
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// getInfoFromFlags obtains the project version and plugin keys from flags.
func (c *cli) getInfoFromFlags() (string, []string, error) {
	// Partially parse the command line arguments
	fs := pflag.NewFlagSet("base", pflag.ContinueOnError)

	// Load the base command global flags
	fs.AddFlagSet(c.cmd.PersistentFlags())

	// Omit unknown flags to avoid parsing errors
	fs.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}

	// FlagSet special cases --help and -h, so we need to create a dummy flag with these 2 values to prevent the default
	// behavior (printing the usage of this FlagSet) as we want to print the usage message of the underlying command.
	fs.BoolP("help", "h", false, fmt.Sprintf("help for %s", c.commandName))

	// Parse the arguments
	if err := fs.Parse(os.Args[1:]); err != nil {
		return "", []string{}, err
	}

	// Define the flags needed for plugin resolution
	var (
		projectVersion string
		plugins        []string
		err            error
	)
	// GetXxxxx methods will not yield errors because we know for certain these flags exist and types match.
	projectVersion, err = fs.GetString(projectVersionFlag)
	if err != nil {
		return "", []string{}, err
	}
	plugins, err = fs.GetStringSlice(pluginsFlag)
	if err != nil {
		return "", []string{}, err
	}

	// Remove leading and trailing spaces
	for i, key := range plugins {
		plugins[i] = strings.TrimSpace(key)
	}

	return projectVersion, plugins, nil
}

// getInfoFromConfigFile obtains the project version and plugin keys from the project config file.
func (c cli) getInfoFromConfigFile() (config.Version, []string, error) {
	// Read the project configuration file
	projectConfig, err := internalconfig.Read(c.fs)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		return config.Version{}, nil, nil
	default:
		return config.Version{}, nil, err
	}

	return getInfoFromConfig(projectConfig)
}

// getInfoFromConfig obtains the project version and plugin keys from the project config.
// It is extracted from getInfoFromConfigFile for testing purposes.
func getInfoFromConfig(projectConfig config.Config) (config.Version, []string, error) {
	// Split the comma-separated plugins
	var pluginSet []string
	if projectConfig.GetLayout() != "" {
		for _, p := range strings.Split(projectConfig.GetLayout(), ",") {
			pluginSet = append(pluginSet, strings.TrimSpace(p))
		}
	}

	return projectConfig.GetVersion(), pluginSet, nil
}

// resolveFlagsAndConfigFileConflicts checks if the provided combined input from flags and
// the config file is valid and uses default values in case some info was not provided.
func (c cli) resolveFlagsAndConfigFileConflicts(
	flagProjectVersionString string,
	cfgProjectVersion config.Version,
	flagPlugins, cfgPlugins []string,
) (config.Version, []string, error) {
	// Parse project configuration version from flags
	var flagProjectVersion config.Version
	if flagProjectVersionString != "" {
		if err := flagProjectVersion.Parse(flagProjectVersionString); err != nil {
			return config.Version{}, nil, fmt.Errorf("unable to parse project version flag: %w", err)
		}
	}

	// Resolve project version
	var projectVersion config.Version
	isFlagProjectVersionInvalid := flagProjectVersion.Validate() != nil
	isCfgProjectVersionInvalid := cfgProjectVersion.Validate() != nil
	switch {
	// If they are both invalid (empty is invalid), use the default
	case isFlagProjectVersionInvalid && isCfgProjectVersionInvalid:
		projectVersion = c.defaultProjectVersion
	// If any is invalid (empty is invalid), choose the other
	case isCfgProjectVersionInvalid:
		projectVersion = flagProjectVersion
	case isFlagProjectVersionInvalid:
		projectVersion = cfgProjectVersion
	// If they are equal doesn't matter which we choose
	case flagProjectVersion.Compare(cfgProjectVersion) == 0:
		projectVersion = flagProjectVersion
	// If both are valid (empty is invalid) and they are different error out
	default:
		return config.Version{}, nil, fmt.Errorf("project version conflict between command line args (%s) "+
			"and project configuration file (%s)", flagProjectVersionString, cfgProjectVersion)
	}

	// Resolve plugins
	var plugins []string
	isFlagPluginsEmpty := len(flagPlugins) == 0
	isCfgPluginsEmpty := len(cfgPlugins) == 0
	switch {
	// If they are both empty, use the default
	case isFlagPluginsEmpty && isCfgPluginsEmpty:
		if defaults, hasDefaults := c.defaultPlugins[projectVersion]; hasDefaults {
			plugins = defaults
		}
	// If any is empty, choose the other
	case isCfgPluginsEmpty:
		plugins = flagPlugins
	case isFlagPluginsEmpty:
		plugins = cfgPlugins
	// If they are equal doesn't matter which we choose
	case equalStringSlice(flagPlugins, cfgPlugins):
		plugins = flagPlugins
	// If none is empty and they are different error out
	default:
		return config.Version{}, nil, fmt.Errorf("plugins conflict between command line args (%v) "+
			"and project configuration file (%v)", flagPlugins, cfgPlugins)
	}
	// Validate the plugins
	for _, p := range plugins {
		if err := plugin.ValidateKey(p); err != nil {
			return config.Version{}, nil, err
		}
	}

	return projectVersion, plugins, nil
}

// getInfo obtains the project version and plugin keys resolving conflicts among flags and the project config file.
func (c *cli) getInfo() error {
	// Get project version and plugin info from flags
	flagProjectVersion, flagPlugins, err := c.getInfoFromFlags()
	if err != nil {
		return err
	}
	// Get project version and plugin info from project configuration file
	cfgProjectVersion, cfgPlugins, _ := c.getInfoFromConfigFile()
	// We discard the error because not being able to read a project configuration file
	// is not fatal for some commands. The ones that require it need to check its existence.

	// Resolve project version and plugin keys
	c.projectVersion, c.pluginKeys, err = c.resolveFlagsAndConfigFileConflicts(
		flagProjectVersion, cfgProjectVersion, flagPlugins, cfgPlugins,
	)
	return err
}

const unstablePluginMsg = " (plugin version is unstable, there may be an upgrade available: " +
	"https://kubebuilder.io/migration/plugin/plugins.html)"

// resolve selects from the available plugins those that match the project version and plugin keys provided.
func (c *cli) resolve() error {
	var plugins []plugin.Plugin
	for _, pluginKey := range c.pluginKeys {
		name, version := plugin.SplitKey(pluginKey)
		shortName := plugin.GetShortName(name)

		// Plugins are often released as "unstable" (alpha/beta) versions, then upgraded to "stable".
		// This upgrade effectively removes a plugin, which is fine because unstable plugins are
		// under no support contract. However users should be notified _why_ their plugin cannot be found.
		var extraErrMsg string
		if version != "" {
			var ver plugin.Version
			if err := ver.Parse(version); err != nil {
				return fmt.Errorf("error parsing input plugin version from key %q: %v", pluginKey, err)
			}
			if !ver.IsStable() {
				extraErrMsg = unstablePluginMsg
			}
		}

		var resolvedPlugins []plugin.Plugin
		isFullName := shortName != name
		hasVersion := version != ""

		switch {
		// If it is fully qualified search it
		case isFullName && hasVersion:
			p, isKnown := c.plugins[pluginKey]
			if !isKnown {
				return fmt.Errorf("unknown fully qualified plugin %q%s", pluginKey, extraErrMsg)
			}
			if !plugin.SupportsVersion(p, c.projectVersion) {
				return fmt.Errorf("plugin %q does not support project version %q", pluginKey, c.projectVersion)
			}
			plugins = append(plugins, p)
			continue
		// Shortname with version
		case hasVersion:
			for _, p := range c.plugins {
				// Check that the shortname and version match
				if plugin.GetShortName(p.Name()) == name && p.Version().String() == version {
					resolvedPlugins = append(resolvedPlugins, p)
				}
			}
		// Full name without version
		case isFullName:
			for _, p := range c.plugins {
				// Check that the name matches
				if p.Name() == name {
					resolvedPlugins = append(resolvedPlugins, p)
				}
			}
		// Shortname without version
		default:
			for _, p := range c.plugins {
				// Check that the shortname matches
				if plugin.GetShortName(p.Name()) == name {
					resolvedPlugins = append(resolvedPlugins, p)
				}
			}
		}

		// Filter the ones that do not support the required project version
		i := 0
		for _, resolvedPlugin := range resolvedPlugins {
			if plugin.SupportsVersion(resolvedPlugin, c.projectVersion) {
				resolvedPlugins[i] = resolvedPlugin
				i++
			}
		}
		resolvedPlugins = resolvedPlugins[:i]

		// Only 1 plugin can match
		switch len(resolvedPlugins) {
		case 0:
			return fmt.Errorf("no plugin could be resolved with key %q for project version %q%s",
				pluginKey, c.projectVersion, extraErrMsg)
		case 1:
			plugins = append(plugins, resolvedPlugins[0])
		default:
			return fmt.Errorf("ambiguous plugin %q for project version %q", pluginKey, c.projectVersion)
		}
	}

	c.resolvedPlugins = plugins
	return nil
}

// addSubcommands returns a root command with a subcommand tree reflecting the
// current project's state.
func (c *cli) addSubcommands() {
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

// buildCmd creates the underlying cobra command and stores it internally.
func (c *cli) buildCmd() error {
	c.cmd = c.newRootCmd()

	// Get project version and plugin keys.
	if err := c.getInfo(); err != nil {
		return err
	}

	// Resolve plugins for project version and plugin keys.
	if err := c.resolve(); err != nil {
		return err
	}

	// Add the subcommands
	c.addSubcommands()

	return nil
}

// addExtraCommands adds the additional commands.
func (c *cli) addExtraCommands() error {
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
func (c cli) printDeprecationWarnings() {
	for _, p := range c.resolvedPlugins {
		if d, isDeprecated := p.(plugin.Deprecated); isDeprecated {
			fmt.Printf(noticeColor, fmt.Sprintf(deprecationFmt, d.DeprecationWarning()))
		}
	}
}

// Run implements CLI.Run.
func (c cli) Run() error {
	return c.cmd.Execute()
}
