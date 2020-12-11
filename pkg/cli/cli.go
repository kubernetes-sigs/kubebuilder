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

	internalconfig "sigs.k8s.io/kubebuilder/v2/pkg/cli/internal/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/internal/validation"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"
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
type cli struct {
	/* Fields set by Option */

	// Root command name. It is injected downstream to provide correct help, usage, examples and errors.
	commandName string
	// CLI version string.
	version string
	// Default project version in case none is provided and a config file can't be found.
	defaultProjectVersion string
	// Default plugins in case none is provided and a config file can't be found.
	defaultPlugins map[string][]string
	// Plugins registered in the cli.
	plugins map[string]plugin.Plugin
	// Commands injected by options.
	extraCommands []*cobra.Command
	// Whether to add a completion command to the cli.
	completionCommand bool

	/* Internal fields */

	// Project version to scaffold.
	projectVersion string
	// Plugin keys to scaffold with.
	pluginKeys []string

	// A filtered set of plugins that should be used by command constructors.
	resolvedPlugins []plugin.Plugin

	// Whether some generic help should be printed, i.e. if the binary
	// was invoked outside of a project with incorrect flags or -h|--help.
	doHelp bool

	// Root command.
	cmd *cobra.Command
}

// New creates a new cli instance.
func New(opts ...Option) (CLI, error) {
	// Create the CLI.
	c, err := newCLI(opts...)
	if err != nil {
		return nil, err
	}

	// Get project version and plugin keys.
	if err := c.getInfo(); err != nil {
		return nil, err
	}

	// Resolve plugins for project version and plugin keys.
	if err := c.resolve(); err != nil {
		return nil, err
	}

	// Build the root command.
	c.cmd = c.buildRootCmd()

	// Add extra commands injected by options.
	for _, cmd := range c.extraCommands {
		for _, subCmd := range c.cmd.Commands() {
			if cmd.Name() == subCmd.Name() {
				return nil, fmt.Errorf("command %q already exists", cmd.Name())
			}
		}
		c.cmd.AddCommand(cmd)
	}

	// Write deprecation notices after all commands have been constructed.
	for _, p := range c.resolvedPlugins {
		if d, isDeprecated := p.(plugin.Deprecated); isDeprecated {
			fmt.Printf(noticeColor, fmt.Sprintf(deprecationFmt, d.DeprecationWarning()))
		}
	}

	return c, nil
}

// newCLI creates a default cli instance and applies the provided options.
// It is as a separate function for test purposes.
func newCLI(opts ...Option) (*cli, error) {
	// Default cli options.
	c := &cli{
		commandName:           "kubebuilder",
		defaultProjectVersion: internalconfig.DefaultVersion,
		defaultPlugins:        make(map[string][]string),
		plugins:               make(map[string]plugin.Plugin),
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
func (c *cli) getInfoFromFlags() (string, []string) {
	// Partially parse the command line arguments
	fs := pflag.NewFlagSet("base", pflag.ExitOnError)

	// Omit unknown flags to avoid parsing errors
	fs.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}

	// Define the flags needed for plugin resolution
	var projectVersion, plugins string
	var help bool
	fs.StringVar(&projectVersion, projectVersionFlag, "", "project version")
	fs.StringVar(&plugins, pluginsFlag, "", "plugins to run")
	fs.BoolVarP(&help, "help", "h", false, "help flag")

	// Parse the arguments
	err := fs.Parse(os.Args[1:])

	// User needs *generic* help if args are incorrect or --help is set and
	// --project-version is not set. Plugin-specific help is given if a
	// plugin.Context is updated, which does not require this field.
	c.doHelp = err != nil || help && !fs.Lookup(projectVersionFlag).Changed

	// Split the comma-separated plugins
	var pluginSet []string
	if plugins != "" {
		for _, p := range strings.Split(plugins, ",") {
			pluginSet = append(pluginSet, strings.TrimSpace(p))
		}
	}

	return projectVersion, pluginSet
}

// getInfoFromConfigFile obtains the project version and plugin keys from the project config file.
func getInfoFromConfigFile() (string, []string, error) {
	// Read the project configuration file
	projectConfig, err := internalconfig.Read()
	switch {
	case err == nil:
	case os.IsNotExist(err):
		return "", nil, nil
	default:
		return "", nil, err
	}

	return getInfoFromConfig(projectConfig)
}

// getInfoFromConfig obtains the project version and plugin keys from the project config.
// It is extracted from getInfoFromConfigFile for testing purposes.
func getInfoFromConfig(projectConfig *config.Config) (string, []string, error) {
	// Split the comma-separated plugins
	var pluginSet []string
	if projectConfig.Layout != "" {
		for _, p := range strings.Split(projectConfig.Layout, ",") {
			pluginSet = append(pluginSet, strings.TrimSpace(p))
		}
	}

	return projectConfig.Version, pluginSet, nil
}

// resolveFlagsAndConfigFileConflicts checks if the provided combined input from flags and
// the config file is valid and uses default values in case some info was not provided.
func (c cli) resolveFlagsAndConfigFileConflicts(
	flagProjectVersion, cfgProjectVersion string,
	flagPlugins, cfgPlugins []string,
) (string, []string, error) {
	// Resolve project version
	var projectVersion string
	switch {
	// If they are both blank, use the default
	case flagProjectVersion == "" && cfgProjectVersion == "":
		projectVersion = c.defaultProjectVersion
	// If they are equal doesn't matter which we choose
	case flagProjectVersion == cfgProjectVersion:
		projectVersion = flagProjectVersion
	// If any is blank, choose the other
	case cfgProjectVersion == "":
		projectVersion = flagProjectVersion
	case flagProjectVersion == "":
		projectVersion = cfgProjectVersion
	// If none is blank and they are different error out
	default:
		return "", nil, fmt.Errorf("project version conflict between command line args (%s) "+
			"and project configuration file (%s)", flagProjectVersion, cfgProjectVersion)
	}
	// It still may be empty if default, flag and config project versions are empty
	if projectVersion != "" {
		// Validate the project version
		if err := validation.ValidateProjectVersion(projectVersion); err != nil {
			return "", nil, err
		}
	}

	// Resolve plugins
	var plugins []string
	switch {
	// If they are both blank, use the default
	case len(flagPlugins) == 0 && len(cfgPlugins) == 0:
		plugins = c.defaultPlugins[projectVersion]
	// If they are equal doesn't matter which we choose
	case equalStringSlice(flagPlugins, cfgPlugins):
		plugins = flagPlugins
	// If any is blank, choose the other
	case len(cfgPlugins) == 0:
		plugins = flagPlugins
	case len(flagPlugins) == 0:
		plugins = cfgPlugins
	// If none is blank and they are different error out
	default:
		return "", nil, fmt.Errorf("plugins conflict between command line args (%v) "+
			"and project configuration file (%v)", flagPlugins, cfgPlugins)
	}
	// Validate the plugins
	for _, p := range plugins {
		if err := plugin.ValidateKey(p); err != nil {
			return "", nil, err
		}
	}

	return projectVersion, plugins, nil
}

// getInfo obtains the project version and plugin keys resolving conflicts among flags and the project config file.
func (c *cli) getInfo() error {
	// Get project version and plugin info from flags
	flagProjectVersion, flagPlugins := c.getInfoFromFlags()
	// Get project version and plugin info from project configuration file
	cfgProjectVersion, cfgPlugins, _ := getInfoFromConfigFile()
	// We discard the error because not being able to read a project configuration file
	// is not fatal for some commands. The ones that require it need to check its existence.

	// Resolve project version and plugin keys
	var err error
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
			ver, err := plugin.ParseVersion(version)
			if err != nil {
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

// buildRootCmd returns a root command with a subcommand tree reflecting the
// current project's state.
func (c cli) buildRootCmd() *cobra.Command {
	rootCmd := c.defaultCommand()

	// kubebuilder completion
	// Only add completion if requested
	if c.completionCommand {
		rootCmd.AddCommand(c.newCompletionCmd())
	}

	// kubebuilder create
	createCmd := c.newCreateCmd()
	// kubebuilder create api
	createCmd.AddCommand(c.newCreateAPICmd())
	createCmd.AddCommand(c.newCreateWebhookCmd())
	if createCmd.HasSubCommands() {
		rootCmd.AddCommand(createCmd)
	}

	// kubebuilder edit
	rootCmd.AddCommand(c.newEditCmd())

	// kubebuilder init
	rootCmd.AddCommand(c.newInitCmd())

	// kubebuilder version
	// Only add version if a version string was provided
	if c.version != "" {
		rootCmd.AddCommand(c.newVersionCmd())
	}

	return rootCmd
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

  %[1]s init --domain example.com --license apache2 --owner "The Kubernetes authors"

- create one or more a new resource APIs and add your code to them:

  %[1]s create api --group <group> --version <version> --kind <Kind>

Create resource will prompt the user for if it should scaffold the Resource and / or Controller. To only
scaffold a Controller for an existing Resource, select "n" for Resource. To only define
the schema for a Resource without writing a Controller, select "n" for Controller.

After the scaffold is written, api will run make on the project.
`,
			c.commandName),
		Example: fmt.Sprintf(`
  # Initialize your project
  %[1]s init --domain example.com --license apache2 --owner "The Kubernetes authors"

  # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
  %[1]s create api --group ship --version v1beta1 --kind Frigate

  # Edit the API Scheme
  nano api/v1beta1/frigate_types.go

  # Edit the Controller
  nano controllers/frigate_controller.go

  # Install CRDs into the Kubernetes cluster using kubectl apply
  make install

  # Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
  make run
`,
			c.commandName),
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				log.Fatal(err)
			}
		},
	}
}

// Run implements CLI.Run.
func (c cli) Run() error {
	return c.cmd.Execute()
}
