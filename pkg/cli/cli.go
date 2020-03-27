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
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	internalconfig "sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/internal/validation"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

const (
	noticeColor         = "\033[1;36m%s\033[0m"
	runInProjectRootMsg = `For project-specific information, run this command in the root directory of a
project.
`

	projectVersionFlag = "project-version"
	helpFlag           = "help"
	pluginNamesFlag    = "plugins"
)

// CLI interacts with a command line interface.
type CLI interface {
	// Run runs the CLI, usually returning an error if command line configuration
	// is incorrect.
	Run() error
}

// Option is a function that can configure the cli
type Option func(*cli) error

// cli defines the command line structure and interfaces that are used to
// scaffold kubebuilder project files.
type cli struct {
	// Base command name. Can be injected downstream.
	commandName string
	// Default project version. Used in CLI flag setup.
	defaultProjectVersion string
	// Project version to scaffold.
	projectVersion string
	// True if the project has config file.
	configured bool
	// Whether the command is requesting help.
	doGenericHelp bool

	// Plugins injected by options.
	pluginsFromOptions map[string][]plugin.Base
	// Default plugins injected by options.
	defaultPluginsFromOptions map[string][]plugin.Base
	// A mapping of plugin name to version passed by to --plugins.
	cliPluginKeys map[string]string
	// A filtered set of plugins that should be used by command constructors.
	resolvedPlugins []plugin.Base

	// Base command.
	cmd *cobra.Command
	// Commands injected by options.
	extraCommands []*cobra.Command
}

// New creates a new cli instance.
func New(opts ...Option) (CLI, error) {
	c := &cli{
		commandName:               "kubebuilder",
		defaultProjectVersion:     internalconfig.DefaultVersion,
		pluginsFromOptions:        make(map[string][]plugin.Base),
		defaultPluginsFromOptions: make(map[string][]plugin.Base),
		cliPluginKeys:             make(map[string]string),
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if err := c.initialize(); err != nil {
		return nil, err
	}
	return c, nil
}

// Run runs the cli.
func (c cli) Run() error {
	return c.cmd.Execute()
}

// WithCommandName is an Option that sets the cli's root command name.
func WithCommandName(name string) Option {
	return func(c *cli) error {
		c.commandName = name
		return nil
	}
}

// WithDefaultProjectVersion is an Option that sets the cli's default project
// version. Setting an unknown version will result in an error.
func WithDefaultProjectVersion(version string) Option {
	return func(c *cli) error {
		c.defaultProjectVersion = version
		return nil
	}
}

// WithPlugins is an Option that sets the cli's plugins.
func WithPlugins(plugins ...plugin.Base) Option {
	return func(c *cli) error {
		for _, p := range plugins {
			for _, version := range p.SupportedProjectVersions() {
				if _, ok := c.pluginsFromOptions[version]; !ok {
					c.pluginsFromOptions[version] = []plugin.Base{}
				}
				c.pluginsFromOptions[version] = append(c.pluginsFromOptions[version], p)
			}
		}
		return nil
	}
}

// WithDefaultPlugins is an Option that sets the cli's default plugins.
func WithDefaultPlugins(plugins ...plugin.Base) Option {
	return func(c *cli) error {
		for _, p := range plugins {
			for _, version := range p.SupportedProjectVersions() {
				if _, ok := c.defaultPluginsFromOptions[version]; !ok {
					c.defaultPluginsFromOptions[version] = []plugin.Base{}
				}
				c.defaultPluginsFromOptions[version] = append(c.defaultPluginsFromOptions[version], p)
			}
		}
		return nil
	}
}

// WithExtraCommands is an Option that adds extra subcommands to the cli.
// Adding extra commands that duplicate existing commands results in an error.
func WithExtraCommands(cmds ...*cobra.Command) Option {
	return func(c *cli) error {
		c.extraCommands = append(c.extraCommands, cmds...)
		return nil
	}
}

// initialize initializes the cli.
func (c *cli) initialize() error {
	// Initialize cli with globally-relevant flags or flags that determine
	// certain plugin type's configuration.
	if err := c.parseBaseFlags(); err != nil {
		return err
	}

	// Configure the project version first for plugin retrieval in command
	// constructors.
	projectConfig, err := internalconfig.Read()
	if os.IsNotExist(err) {
		c.configured = false
		if c.projectVersion == "" {
			c.projectVersion = c.defaultProjectVersion
		}
	} else if err == nil {
		c.configured = true
		c.projectVersion = projectConfig.Version
	} else {
		return fmt.Errorf("failed to read config: %v", err)
	}

	// Validate after setting projectVersion but before buildRootCmd so we error
	// out before an error resulting from an incorrect cli is returned downstream.
	if err = c.validate(); err != nil {
		return err
	}

	// With or without a config, a user can:
	// 1. Not set --plugins
	// 2. Set --plugins to a plugin, ex. --plugins=go
	// In case 1, either default plugins or the config layout can be used to
	// determine which plugin to use. Otherwise, the value passed to --plugins
	// is used. Since both layout and a --plugins value are user input, they
	// may need to be resolved to known plugin keys.
	var filterKeys map[string]string
	switch {
	case len(c.cliPluginKeys) != 0:
		// Filter plugins by keys passed in CLI.
		filterKeys = c.cliPluginKeys
	case c.configured && !projectConfig.IsV1():
		// All non-v1 configs must have a layout key. This check will help with
		// migration.
		if projectConfig.Layout == "" {
			return fmt.Errorf("config must have a layout value")
		}
		// Use defaults if no config is found, otherwise infer plugins from the
		// config's layout.
		name, version := plugin.SplitKey(projectConfig.Layout)
		filterKeys = map[string]string{name: version}
	default:
		// Use the default plugins for this project version.
		c.resolvedPlugins = c.defaultPluginsFromOptions[c.projectVersion]
	}
	if len(filterKeys) != 0 {
		plugins := c.pluginsFromOptions[c.projectVersion]
		c.resolvedPlugins, err = resolvePluginsByKeys(plugins, filterKeys)
		if err != nil {
			return err
		}
	}

	c.cmd = c.buildRootCmd()

	// Add extra commands injected by options.
	for _, cmd := range c.extraCommands {
		for _, subCmd := range c.cmd.Commands() {
			if cmd.Name() == subCmd.Name() {
				return fmt.Errorf("command %q already exists", cmd.Name())
			}
		}
		c.cmd.AddCommand(cmd)
	}

	// Write deprecation notices after all commands have been constructed.
	for _, p := range c.resolvedPlugins {
		if d, isDeprecated := p.(plugin.Deprecated); isDeprecated {
			fmt.Printf(noticeColor, fmt.Sprintf("[Deprecation Notice] %s\n\n",
				d.DeprecationWarning()))
		}
	}

	return nil
}

// parseBaseFlags parses the command line arguments, looking for flags that
// affect initialization of a cli. An error is returned only if an error
// unrelated to flag parsing occurs.
func (c *cli) parseBaseFlags() error {
	// Create a dummy "base" flagset to populate from CLI args.
	fs := pflag.NewFlagSet("base", pflag.ExitOnError)
	fs.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}

	// Set base flags that require pre-parsing to initialize c.
	fs.StringVar(&c.projectVersion, projectVersionFlag, c.defaultProjectVersion, "project version")
	help := false
	fs.BoolVarP(&help, helpFlag, "h", false, "print help")
	pluginKeys := []string{}
	fs.StringSliceVar(&pluginKeys, pluginNamesFlag, nil, "plugins to run")

	// Parse current CLI args outside of cobra.
	err := fs.Parse(os.Args[1:])
	// User needs *generic* help if args are incorrect or --help is set and
	// --project-version is not set. Plugin-specific help is given if a
	// plugin.Context is updated, which does not require this field.
	c.doGenericHelp = err != nil || help && !fs.Lookup(projectVersionFlag).Changed

	// Parse plugin keys into a more manageable data structure (map) and check
	// for duplicates.
	for _, key := range pluginKeys {
		pluginName, pluginVersion := plugin.SplitKey(key)
		if pluginName == "" {
			return fmt.Errorf("plugin key %q must at least have a name", key)
		}
		if _, exists := c.cliPluginKeys[pluginName]; exists {
			return fmt.Errorf("duplicate plugin name %q", pluginName)
		}
		c.cliPluginKeys[pluginName] = pluginVersion
	}

	return nil
}

// validate validates fields in a cli.
func (c cli) validate() error {
	// Validate project versions.
	if err := validation.ValidateProjectVersion(c.defaultProjectVersion); err != nil {
		return fmt.Errorf("failed to validate default project version %q: %v", c.defaultProjectVersion, err)
	}
	if err := validation.ValidateProjectVersion(c.projectVersion); err != nil {
		return fmt.Errorf("failed to validate project version %q: %v", c.projectVersion, err)
	}

	if _, versionFound := c.pluginsFromOptions[c.projectVersion]; !versionFound {
		return fmt.Errorf("no plugins for project version %q", c.projectVersion)
	}
	// If --plugins is not set, no layout exists (no config or v1 project),
	// and no defaults exist, we cannot know which plugins to use.
	if (!c.configured || c.projectVersion == config.Version1) && len(c.cliPluginKeys) == 0 {
		_, versionExists := c.defaultPluginsFromOptions[c.projectVersion]
		if !versionExists {
			return fmt.Errorf("no default plugins for project version %s", c.projectVersion)
		}
	}

	// Validate plugin versions and name.
	for _, versionedPlugins := range c.pluginsFromOptions {
		if err := validatePlugins(versionedPlugins...); err != nil {
			return err
		}
	}
	for _, defaultPlugins := range c.defaultPluginsFromOptions {
		if err := validatePlugins(defaultPlugins...); err != nil {
			return err
		}
	}

	// Validate plugin keys set in CLI.
	for pluginName, pluginVersion := range c.cliPluginKeys {
		if err := plugin.ValidateName(pluginName); err != nil {
			return fmt.Errorf("failed to validate plugin name %q: %v", pluginName, err)
		}
		// CLI-set plugins do not have to contain a version.
		if pluginVersion != "" {
			if err := plugin.ValidateVersion(pluginVersion); err != nil {
				return fmt.Errorf("failed to validate plugin %q version %q: %v",
					pluginName, pluginVersion, err)
			}
		}
	}

	return nil
}

// validatePlugins validates the name and versions of a list of plugins.
func validatePlugins(plugins ...plugin.Base) error {
	pluginNameSet := make(map[string]struct{}, len(plugins))
	for _, p := range plugins {
		pluginName := p.Name()
		if err := plugin.ValidateName(pluginName); err != nil {
			return fmt.Errorf("failed to validate plugin name %q: %v", pluginName, err)
		}
		pluginVersion := p.Version()
		if err := plugin.ValidateVersion(pluginVersion); err != nil {
			return fmt.Errorf("failed to validate plugin %q version %q: %v",
				pluginName, pluginVersion, err)
		}
		for _, projectVersion := range p.SupportedProjectVersions() {
			if err := validation.ValidateProjectVersion(projectVersion); err != nil {
				return fmt.Errorf("failed to validate plugin %q supported project version %q: %v",
					pluginName, projectVersion, err)
			}
		}
		// Check for duplicate plugin names. Names outside of a version can
		// conflict because multiple project versions of a plugin may exist.
		if _, seen := pluginNameSet[pluginName]; seen {
			return fmt.Errorf("two plugins have the same name: %q", pluginName)
		}
		pluginNameSet[pluginName] = struct{}{}
	}
	return nil
}

// buildRootCmd returns a root command with a subcommand tree reflecting the
// current project's state.
func (c cli) buildRootCmd() *cobra.Command {
	configuredAndV1 := c.configured && c.projectVersion == config.Version1

	rootCmd := c.defaultCommand()
	rootCmd.PersistentFlags().StringSlice(pluginNamesFlag, nil, "plugins to run")

	// kubebuilder alpha
	alphaCmd := c.newAlphaCmd()
	// kubebuilder alpha webhook (v1 only)
	if configuredAndV1 {
		alphaCmd.AddCommand(c.newCreateWebhookCmd())
	}
	if alphaCmd.HasSubCommands() {
		rootCmd.AddCommand(alphaCmd)
	}

	// kubebuilder create
	createCmd := c.newCreateCmd()
	// kubebuilder create api
	createCmd.AddCommand(c.newCreateAPICmd())
	// kubebuilder create webhook (!v1)
	if !configuredAndV1 {
		createCmd.AddCommand(c.newCreateWebhookCmd())
	}
	if createCmd.HasSubCommands() {
		rootCmd.AddCommand(createCmd)
	}

	// kubebuilder init
	rootCmd.AddCommand(c.newInitCmd())

	return rootCmd
}

// resolvePluginsByKeys filters versionedPlugins by resolving names from a
// set of plugin keys, returning unambiguously matched plugins.
// A plugin passes the filter if:
// - Name and version are the same.
// - Long or short name is the same, and only one version for that name exists.
// If long or short name is the same, but multiple versions for that name exist,
// resolvePluginsByKeys will not guess which version to use and returns an error.
func resolvePluginsByKeys(versionedPlugins []plugin.Base, cliPluginKeys map[string]string) ([]plugin.Base, error) {
	// Find all potential mateches for this plugin's name.
	acceptedMatches, maybeMatchesByKey := []plugin.Base{}, map[string][]plugin.Base{}
	for pluginName, pluginVersion := range cliPluginKeys {
		cliPluginKey := plugin.Key(pluginName, pluginVersion)
		// Prevents duplicate versions with the same name from being added to
		// maybeMatchesByKey.
		foundExactMatch := false
		for _, versionedPlugin := range versionedPlugins {
			longName := versionedPlugin.Name()
			shortName := plugin.GetShortName(longName)

			switch {
			// Exact match.
			case pluginName == longName && pluginVersion == versionedPlugin.Version():
				acceptedMatches = append(acceptedMatches, versionedPlugin)
				delete(maybeMatchesByKey, cliPluginKey)
				foundExactMatch = true
			// Is at least a name match, and the CLI version wasn't set or was set
			// and matches.
			case pluginName == longName || pluginName == shortName:
				if pluginVersion == "" || pluginVersion == versionedPlugin.Version() {
					maybeMatchesByKey[cliPluginKey] = append(maybeMatchesByKey[cliPluginKey], versionedPlugin)
				}
			}
			// No more plugins need to be checked for cliPluginKey.
			if foundExactMatch {
				break
			}
		}
		// No possible or exact plugin match was found.
		_, foundMaybeMatch := maybeMatchesByKey[cliPluginKey]
		if !foundExactMatch && !foundMaybeMatch {
			return nil, fmt.Errorf("no plugins for key %q were found", cliPluginKey)
		}
	}

	// No ambiguously keyed plugins.
	if len(maybeMatchesByKey) == 0 {
		return acceptedMatches, nil
	}

	errMsgs := []string{}
	for key, maybeMatches := range maybeMatchesByKey {
		if len(maybeMatches) == 1 {
			// Only one plugin for a CLI key, which means there's only one version
			// for that plugin and either its short or long name matched the CLI name.
			acceptedMatches = append(acceptedMatches, maybeMatches...)
		} else {
			// Multiple possible plugins the user could be specifying, return an
			// error with ambiguous plugin info.
			pluginKeys := []string{}
			for _, p := range maybeMatches {
				pluginKeys = append(pluginKeys, plugin.KeyFor(p))
			}
			errMsgs = append(errMsgs, fmt.Sprintf("%q: %+q", key, pluginKeys))
		}
	}

	if len(errMsgs) == 0 {
		return acceptedMatches, nil
	}
	return nil, fmt.Errorf("ambiguous plugin keys, possible matches: %s", strings.Join(errMsgs, ", "))
}

// defaultCommand returns the root command without its subcommands.
func (c cli) defaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   c.commandName,
		Short: "Development kit for building Kubernetes extensions and tools.",
		Long: fmt.Sprintf(`Development kit for building Kubernetes extensions and tools.

Provides libraries and tools to create new projects, APIs and controllers.
Includes tools for packaging artifacts into an installer container.

Typical project lifecycle:

- initialize a project:

  %s init --domain example.com --license apache2 --owner "The Kubernetes authors"

- create one or more a new resource APIs and add your code to them:

  %s create api --group <group> --version <version> --kind <Kind>

Create resource will prompt the user for if it should scaffold the Resource and / or Controller. To only
scaffold a Controller for an existing Resource, select "n" for Resource. To only define
the schema for a Resource without writing a Controller, select "n" for Controller.

After the scaffold is written, api will run make on the project.
`,
			c.commandName, c.commandName),
		Example: fmt.Sprintf(`
  # Initialize your project
  %s init --domain example.com --license apache2 --owner "The Kubernetes authors"

  # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
  %s create api --group ship --version v1beta1 --kind Frigate

  # Edit the API Scheme
  nano api/v1beta1/frigate_types.go

  # Edit the Controller
  nano controllers/frigate_controller.go

  # Install CRDs into the Kubernetes cluster using kubectl apply
  make install

  # Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
  make run
`,
			c.commandName, c.commandName),

		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				log.Fatal(err)
			}
		},
	}
}
