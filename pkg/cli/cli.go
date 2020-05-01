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
	pluginsFlag        = "plugins"
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
	// A plugin key passed to --plugins on invoking 'init'.
	cliPluginKey string
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
		if err := validation.ValidateProjectVersion(version); err != nil {
			return fmt.Errorf("broken pre-set default project version %q: %v", version, err)
		}
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
		for _, plugins := range c.pluginsFromOptions {
			if err := validatePlugins(plugins...); err != nil {
				return fmt.Errorf("broken pre-set plugins: %v", err)
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
		for _, plugins := range c.defaultPluginsFromOptions {
			if err := validatePlugins(plugins...); err != nil {
				return fmt.Errorf("broken pre-set default plugins: %v", err)
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

		if projectConfig.IsV1() {
			return fmt.Errorf(noticeColor, "The v1 projects are no longer supported.\n"+
				"See how to upgrade your project to v2: https://book.kubebuilder.io/migration/guide.html\n")
		}
	} else {
		return fmt.Errorf("failed to read config: %v", err)
	}

	// Validate after setting projectVersion but before buildRootCmd so we error
	// out before an error resulting from an incorrect cli is returned downstream.
	if err = c.validate(); err != nil {
		return err
	}

	// When invoking 'init', a user can:
	// 1. Not set --plugins
	// 2. Set --plugins to a plugin, ex. --plugins=go-x
	// In case 1, default plugins will be used to determine which plugin to use.
	// In case 2, the value passed to --plugins is used.
	// For all other commands, a config's 'layout' key is used. Since both
	// layout and --plugins values can be short (ex. "go/v2.0.0") or unversioned
	// (ex. "go.kubebuilder.io") keys or both, their values may need to be
	// resolved to known plugins by key.
	plugins := c.pluginsFromOptions[c.projectVersion]
	switch {
	case c.cliPluginKey != "":
		// Filter plugin by keys passed in CLI.
		c.resolvedPlugins, err = resolvePluginsByKey(plugins, c.cliPluginKey)
	case c.configured && projectConfig.IsV3():
		// All non-v1 configs must have a layout key. This check will help with
		// migration.
		if projectConfig.Layout == "" {
			return fmt.Errorf("config must have a layout value")
		}
		// Filter plugin by config's layout value.
		c.resolvedPlugins, err = resolvePluginsByKey(plugins, projectConfig.Layout)
	default:
		// Use the default plugins for this project version.
		c.resolvedPlugins = c.defaultPluginsFromOptions[c.projectVersion]
	}
	if err != nil {
		return err
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

	var help bool
	// Set base flags that require pre-parsing to initialize c.
	fs.BoolVarP(&help, helpFlag, "h", false, "print help")
	fs.StringVar(&c.projectVersion, projectVersionFlag, c.defaultProjectVersion, "project version")
	fs.StringVar(&c.cliPluginKey, pluginsFlag, "", "plugins to run")

	// Parse current CLI args outside of cobra.
	err := fs.Parse(os.Args[1:])
	// User needs *generic* help if args are incorrect or --help is set and
	// --project-version is not set. Plugin-specific help is given if a
	// plugin.Context is updated, which does not require this field.
	c.doGenericHelp = err != nil || help && !fs.Lookup(projectVersionFlag).Changed
	c.cliPluginKey = strings.TrimSpace(c.cliPluginKey)

	return nil
}

// validate validates fields in a cli.
func (c cli) validate() error {
	// Validate project version.
	if err := validation.ValidateProjectVersion(c.projectVersion); err != nil {
		return fmt.Errorf("invalid project version %q: %v", c.projectVersion, err)
	}

	if _, versionFound := c.pluginsFromOptions[c.projectVersion]; !versionFound {
		return fmt.Errorf("no plugins for project version %q", c.projectVersion)
	}
	// If --plugins is not set, no layout exists (no config or project is v1 or v2),
	// and no defaults exist, we cannot know which plugins to use.
	isLayoutSupported := c.projectVersion == config.Version3Alpha
	if (!c.configured || !isLayoutSupported) && c.cliPluginKey == "" {
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

	// Validate plugin keys set in CLI.
	if c.cliPluginKey != "" {
		pluginName, pluginVersion := plugin.SplitKey(c.cliPluginKey)
		if err := plugin.ValidateName(pluginName); err != nil {
			return fmt.Errorf("invalid plugin name %q: %v", pluginName, err)
		}
		// CLI-set plugins do not have to contain a version.
		if pluginVersion != "" {
			if err := plugin.ValidateVersion(pluginVersion); err != nil {
				return fmt.Errorf("invalid plugin %q version %q: %v",
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
			return fmt.Errorf("invalid plugin name %q: %v", pluginName, err)
		}
		pluginVersion := p.Version()
		if err := plugin.ValidateVersion(pluginVersion); err != nil {
			return fmt.Errorf("invalid plugin %q version %q: %v",
				pluginName, pluginVersion, err)
		}
		for _, projectVersion := range p.SupportedProjectVersions() {
			if err := validation.ValidateProjectVersion(projectVersion); err != nil {
				return fmt.Errorf("invalid plugin %q supported project version %q: %v",
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
	rootCmd := c.defaultCommand()

	// kubebuilder alpha
	alphaCmd := c.newAlphaCmd()

	// Only add alpha group if it has subcommands
	if alphaCmd.HasSubCommands() {
		rootCmd.AddCommand(alphaCmd)
	}

	// kubebuilder create
	createCmd := c.newCreateCmd()
	// kubebuilder create api
	createCmd.AddCommand(c.newCreateAPICmd())
	createCmd.AddCommand(c.newCreateWebhookCmd())
	if createCmd.HasSubCommands() {
		rootCmd.AddCommand(createCmd)
	}

	// kubebuilder init
	rootCmd.AddCommand(c.newInitCmd())

	return rootCmd
}

// resolvePluginsByKey finds a plugin for pluginKey if it exactly matches
// some form of a known plugin's key. Those forms can be a:
// - Fully qualified key: "go.kubebuilder.io/v2.0.0"
// - Short key: "go/v2.0.0"
// - Fully qualified name: "go.kubebuilder.io"
// - Short name: "go"
// Some of these keys may conflict, ex. the fully-qualified and short names of
// "go.kubebuilder.io/v1.0.0" and "go.kubebuilder.io/v2.0.0" have ambiguous
// unversioned names "go.kubernetes.io" and "go". If pluginKey is ambiguous
// or does not match any known plugin's key, an error is returned.
//
// Note: resolvePluginsByKey returns a slice so initialize() can generalize
// setting default plugins if no pluginKey is set.
func resolvePluginsByKey(versionedPlugins []plugin.Base, pluginKey string) ([]plugin.Base, error) {
	// Make a set of all possible key combinations to check pluginKey against.
	// If the key is not ambiguous, set a valid pointer to the plugin for that
	// key, otherwise set a tombstone so we know it is ambiguous. There will
	// always be at least one key per plugin if their names are fully-qualified.
	//
	// Note: this isn't actually that inefficient compared to a memory-efficient
	// solution since we're working with very small N's; it is also very simple.
	allPluginKeyCombos := make(map[string]*plugin.Base)
	for i, p := range versionedPlugins {
		key := plugin.KeyFor(p)
		// Short-circuit if we have an exact match.
		if key == pluginKey {
			return []plugin.Base{p}, nil
		}
		name := p.Name()
		keys := []string{key, name}
		if shortName := plugin.GetShortName(name); name != shortName {
			keys = append(keys, shortName)
			keys = append(keys, plugin.Key(shortName, p.Version()))
		}

		pp := &versionedPlugins[i]
		for _, k := range keys {
			if _, hasKey := allPluginKeyCombos[k]; hasKey {
				allPluginKeyCombos[k] = nil
			} else {
				allPluginKeyCombos[k] = pp
			}
		}
	}

	pp, hasKey := allPluginKeyCombos[pluginKey]
	if !hasKey {
		return nil, fmt.Errorf("plugin key %q does not match a known plugin", pluginKey)
	}
	if pp == nil {
		return nil, fmt.Errorf("plugin key %q matches more than one known plugin", pluginKey)
	}

	return []plugin.Base{*pp}, nil
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
