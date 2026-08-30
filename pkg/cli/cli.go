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
	log "log/slog"
	"os"
	"slices"
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
	pluginsFlagArg     = "--plugins"
	projectVersionFlag = "project-version"
	helpFlagArg        = "--help"
	helpShorthandArg   = "-h"
	shellZsh           = "zsh"

	kubebuilderCommandName          = "kubebuilder"
	kubebuilderSubcommandHelp       = "help"
	kubebuilderSubcommandInit       = "init"
	kubebuilderSubcommandVersion    = "version"
	kubebuilderSubcommandCompletion = "completion"
	pluginGoKubebuilderV4           = "go.kubebuilder.io/v4"
	pluginGoKubebuilderV3           = "go.kubebuilder.io/v3"
	pluginGoKubebuilderV2           = "go.kubebuilder.io/v2"
	generateSubcommand              = "generate"

	pluginsFlagDescription = "Comma-separated list of plugin keys to use. " +
		"If unset, Kubebuilder uses the plugin chain from PROJECT or the CLI default"
	projectVersionFlagDescription = "Project config version used to select compatible plugins and write PROJECT " +
		"(e.g., 3). If unset, Kubebuilder uses the version from PROJECT or the CLI default"
)

// CLI is the command line utility that is used to scaffold kubebuilder project files.
type CLI struct {
	/* Fields set by Option */

	// Root command name. It is injected downstream to provide correct help, usage, examples and errors.
	commandName string
	// Full CLI version string.
	version string
	// CLI version string (just the CLI version number, no extra information).
	cliVersion string
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
	// args contains the command-line arguments used to resolve the project configuration and plugin chain.
	args []string

	/* Internal fields */

	// Plugin keys to scaffold with.
	pluginKeys []string
	// Project version to scaffold.
	projectVersion config.Version
	// configErr stores an error found while resolving the project configuration or plugin chain.
	configErr error
	// flagErr stores an error found while reading the command line.
	flagErr error
	// configSkipped reports whether the project configuration was left unread.
	configSkipped bool

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
// It returns an error if any of the provided options fails. A project configuration that cannot be
// read is not an error here. The command tree is still built, and the failure is reported when
// running a command that consumes the configuration.
func New(options ...Option) (*CLI, error) {
	// Create the CLI.
	c, err := newCLI(options...)
	if err != nil {
		return nil, err
	}

	// Build the cmd tree.
	c.buildCmd()

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
		commandName: kubebuilderCommandName,
		description: `CLI tool for building Kubernetes extensions and tools.
`,
		plugins:        make(map[string]plugin.Plugin),
		defaultPlugins: make(map[config.Version][]string),
		fs:             machinery.Filesystem{FS: afero.NewOsFs()},
		args:           defaultArgs(),
	}

	// Apply provided options.
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// defaultArgs returns the command-line arguments of the running program without its name.
func defaultArgs() []string {
	if len(os.Args) < 2 {
		return nil
	}

	return os.Args[1:]
}

// buildCmd creates the underlying cobra command and stores it internally.
func (c *CLI) buildCmd() {
	c.cmd = c.newRootCmd()

	c.configSkipped = isSubcommandWithoutConfig(c.args)

	if err := c.resolveInfo(!c.configSkipped); err != nil {
		// Cobra has not parsed the command line yet, so the command that will run is unknown.
		// Keep a working command tree and let the root hook raise this where it matters.
		var parseErr flagError
		if errors.As(err, &parseErr) {
			c.flagErr = err
		} else {
			c.configErr = err
		}
		c.resolveWithoutConfig()
	}

	c.addSubcommands()
}

// resolveInfo obtains the project version and the plugin keys, and resolves the plugin chain.
func (c *CLI) resolveInfo(readConfig bool) error {
	var uve config.UnsupportedVersionError

	// Get project version and plugin keys.
	switch err := c.getInfo(readConfig); {
	case err == nil:
	case errors.As(err, &uve) && uve.Version.Compare(config.Version{Number: 3, Stage: stage.Alpha}) == 0:
		// Use the stable project version when it is registered.
		stableVersion := config.Version{
			Number: uve.Version.Number,
		}
		if !config.IsRegistered(stableVersion) {
			// stable version not registered, let's bail out
			return err
		}
		c.projectVersion = stableVersion
	default:
		return err
	}

	// Resolve plugins for project version and plugin keys.
	return c.resolvePlugins()
}

// resolveSkippedConfig reads the project configuration that was left unread while the command tree
// was built. The tree cannot be built again, so a plugin chain that differs from the one it was
// built with is reported as an error instead of scaffolding with the wrong plugins.
func (c *CLI) resolveSkippedConfig() error {
	builtWith := slices.Clone(c.pluginKeys)
	builtForVersion := c.projectVersion
	c.forgetPluginChain()

	if err := c.resolveInfo(true); err != nil {
		return err
	}

	if !slices.Equal(c.pluginKeys, builtWith) || c.projectVersion.Compare(builtForVersion) != 0 {
		return fmt.Errorf("the command tree was built for the plugin chain %q but the project requires %q: "+
			"run the program with the arguments of the command that Command().SetArgs runs",
			strings.Join(builtWith, ","), strings.Join(c.pluginKeys, ","))
	}

	return nil
}

// resolveWithoutConfig builds a fallback plugin chain without reading the project configuration.
// It uses valid flag values when possible and otherwise uses the CLI defaults.
func (c *CLI) resolveWithoutConfig() {
	const withFlags, withoutFlags = true, false

	if c.resolveDefaultPlugins(withFlags) == nil {
		return
	}

	_ = c.resolveDefaultPlugins(withoutFlags)
}

// resolveDefaultPlugins resolves the plugin chain from the CLI defaults, optionally letting the
// flags override them. The chain resolved so far is discarded first, so a failed attempt leaves
// nothing behind for the next one.
func (c *CLI) resolveDefaultPlugins(withFlags bool) error {
	c.forgetPluginChain()

	if withFlags {
		if err := c.getInfoFromFlags(false); err != nil {
			return err
		}
	}
	c.getInfoFromDefaults()

	if err := c.resolvePlugins(); err != nil {
		c.resolvedPlugins = nil
		return err
	}

	return nil
}

// forgetPluginChain clears the selected plugin keys, project version, and resolved plugins.
func (c *CLI) forgetPluginChain() {
	c.pluginKeys = nil
	c.projectVersion = config.Version{}
	c.resolvedPlugins = nil
}

// flagError stores a command-line parsing error.
type flagError struct{ err error }

// Error returns the command-line parsing error as a string.
func (e flagError) Error() string { return e.err.Error() }

// Unwrap returns the command-line parsing error for errors.Is and errors.As.
func (e flagError) Unwrap() error { return e.err }

// getInfo obtains the plugin keys and project version while resolving conflicts between the project
// configuration and flags.
func (c *CLI) getInfo(readConfig bool) error {
	// A missing PROJECT file is not an error here. Commands that require it check for it later.
	hasConfigFile := readConfig
	var configErr error
	if readConfig {
		switch err := c.getInfoFromConfigFile(); {
		case err == nil:
		case errors.Is(err, os.ErrNotExist):
			hasConfigFile = false
		default:
			// The flags are still read, so that a malformed command line is reported instead of a
			// configuration the command may not even consume.
			configErr = err
			hasConfigFile = false
		}
	}

	// We can't early return here in case a project configuration file was found because
	// this command call may override the project plugins.

	// Get project version and plugin info from flags
	if err := c.getInfoFromFlags(hasConfigFile); err != nil {
		return flagError{err}
	}
	if configErr != nil {
		return configErr
	}

	// Get project version and plugin info from defaults
	c.getInfoFromDefaults()

	return nil
}

// getInfoFromConfigFile obtains the project version and plugin keys from the project config file.
func (c *CLI) getInfoFromConfigFile() error {
	// Read the project configuration file
	cfg := yamlstore.New(c.fs)

	// Workaround for https://github.com/kubernetes-sigs/kubebuilder/issues/4433
	//
	// This allows the `kubebuilder alpha generate` command to work with old projects
	// that use plugin versions no longer supported (like go.kubebuilder.io/v3).
	//
	// We read the PROJECT file into memory and update the plugin version (e.g. from v3 to v4)
	// before the CLI tries to load it. This avoids errors during config loading
	// and lets users migrate their project layout from go/v3 to go/v4.

	if isAlphaGenerateCommand(c.args) {
		// Only a regular file can be patched. Anything else at the path is classified and reported
		// by Load, and a named pipe must never be opened: reading it blocks until something writes.
		if isRegularPathNoFollow(c.fs.FS, yamlstore.DefaultPath) {
			if err := patchProjectFileInMemoryIfNeeded(c.fs.FS, yamlstore.DefaultPath); err != nil {
				return err
			}
		}
	}

	if err := cfg.Load(); err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	return c.getInfoFromConfig(cfg.Config())
}

// isRegularPathNoFollow reports whether path is a regular file without following a final symbolic
// link. If the filesystem cannot make that distinction, it is not safe to patch the path.
func isRegularPathNoFollow(fs afero.Fs, path string) bool {
	lstater, ok := fs.(afero.Lstater)
	if !ok {
		return false
	}

	info, _, err := lstater.LstatIfPossible(path)
	return err == nil && info.Mode().IsRegular()
}

// positionalArgs returns arguments that are not flags or flag values.
func positionalArgs(args []string) []string {
	positional := []string{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		// A flag in --flag=value form consumes no additional argument.
		if strings.Contains(arg, "=") {
			continue
		}

		// A flag consumes the following argument only when it is not another flag.
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
		}
	}

	return positional
}

// hasOnlyRootFlags returns true when args consists solely of root flags and their values.
func hasOnlyRootFlags(args []string) bool {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case pluginsFlagArg, "--" + projectVersionFlag, helpFlagArg, helpShorthandArg:
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
		default:
			if strings.HasPrefix(args[i], pluginsFlagArg+"=") ||
				strings.HasPrefix(args[i], "--"+projectVersionFlag+"=") ||
				strings.HasPrefix(args[i], helpFlagArg+"=") ||
				strings.HasPrefix(args[i], helpShorthandArg+"=") {
				continue
			}
			return false
		}
	}

	return true
}

// isAlphaGenerateCommand reports whether args invoke `kubebuilder alpha generate`.
func isAlphaGenerateCommand(args []string) bool {
	positional := positionalArgs(args)

	// Check for `alpha generate` in positional arguments
	for i := 0; i < len(positional)-1; i++ {
		if positional[i] == alphaCommand && positional[i+1] == generateSubcommand {
			return true
		}
	}

	return false
}

// subcommandsWithoutConfig are the subcommands that run without a project configuration file.
var subcommandsWithoutConfig = []string{
	kubebuilderSubcommandHelp,
	kubebuilderSubcommandVersion,
	kubebuilderSubcommandCompletion,
	cobra.ShellCompRequestCmd,
	cobra.ShellCompNoDescRequestCmd,
}

// isSubcommandWithoutConfig returns true for invocations that do not need the PROJECT file to build
// their command tree.
func isSubcommandWithoutConfig(args []string) bool {
	positional := positionalArgs(args)
	// With no subcommand, Cobra displays root help. Configuration is not required.
	if len(positional) == 0 {
		return len(args) == 0 || hasOnlyRootFlags(args)
	}

	// The help command consumes the remaining arguments as the command whose help is requested.
	// A plugin-backed command needs the project chain to build accurate help, while help for a
	// context-free command does not.
	if positional[0] == kubebuilderSubcommandHelp {
		return len(positional) == 1 || isSubcommandPathWithoutConfig(positional[1:])
	}

	return isSubcommandPathWithoutConfig(positional)
}

// isSubcommandPathWithoutConfig returns true for a subcommand path that never consumes the PROJECT
// file. The path holds the subcommand names leading to the command, such as "alpha" and "generate".
// The root command displays help, so it does not require the configuration either.
func isSubcommandPathWithoutConfig(path []string) bool {
	if len(path) == 0 {
		return true
	}

	return slices.Contains(subcommandsWithoutConfig, path[0])
}

// patchProjectFileInMemoryIfNeeded updates deprecated plugin keys before the PROJECT file is loaded,
// so that users can run `kubebuilder alpha generate` with older plugin layouts.
//
// See: https://github.com/kubernetes-sigs/kubebuilder/issues/4433
//
// This lets the CLI load the PROJECT file without failing on unsupported plugin versions.
func patchProjectFileInMemoryIfNeeded(fs afero.Fs, path string) error {
	type pluginReplacement struct {
		Old string
		New string
	}

	replacements := []pluginReplacement{
		{pluginGoKubebuilderV2, pluginGoKubebuilderV4},
		{pluginGoKubebuilderV3, pluginGoKubebuilderV4},
		{pluginGoKubebuilderV3 + "-alpha", pluginGoKubebuilderV4},
	}

	content, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil
	}

	original := string(content)
	modified := original

	for _, rep := range replacements {
		if strings.Contains(modified, rep.Old) {
			modified = strings.ReplaceAll(modified, rep.Old, rep.New)
			log.Warn("Project is using an old and unsupported plugin layout",
				"old_layout", rep.Old,
				"new_layout", rep.New,
				"note", "Replace in memory to allow `alpha generate` to work.",
			)
		}
	}

	if modified != original {
		err := afero.WriteFile(fs, path, []byte(modified), machinery.DefaultFilePermission)
		if err != nil {
			return fmt.Errorf("failed to write patched PROJECT file: %w", err)
		}
	}

	return nil
}

// getInfoFromConfig obtains the project version and plugin keys from the project configuration.
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
	// Check if --plugins is followed by --help or -h to avoid parsing help as a plugin value
	// This fixes: kubebuilder init --plugins --help
	for i := 0; i < len(c.args)-1; i++ {
		if c.args[i] == pluginsFlagArg || c.args[i] == pluginsFlagArg+"=" {
			nextArg := c.args[i+1]
			if isHelpFlag(nextArg) {
				// Help was requested, return early to let Cobra handle it
				return nil
			}
		}
	}

	// Partially parse the command line arguments
	fs := pflag.NewFlagSet("base", pflag.ContinueOnError)

	// The global flags are declared again instead of taken from the root command, which holds the
	// values Cobra parses. A shared value appends what it already holds, so reading the arguments
	// more than once would grow the plugin chain.
	fs.StringSlice(pluginsFlag, nil, pluginsFlagDescription)

	// If we were unable to load the project configuration, we should also accept the project version flag
	var projectVersionStr string
	if !hasConfigFile {
		fs.StringVar(&projectVersionStr, projectVersionFlag, "", "project version")
	}

	// FlagSet special cases --help and -h, so we need to create a dummy flag with these 2 values to prevent the default
	// behavior (printing the usage of this FlagSet) as we want to print the usage message of the underlying command.
	fs.BoolP("help", "h", false, fmt.Sprintf("help for %s", c.commandName))

	// Omit unknown flags to avoid parsing errors
	fs.ParseErrorsAllowlist = pflag.ParseErrorsAllowlist{UnknownFlags: true}

	// Parse the arguments
	if err := fs.Parse(c.args); err != nil {
		return fmt.Errorf("could not parse flags: %w", err)
	}

	// If any plugin key was provided, replace those from the project configuration file
	if pluginKeys, err := fs.GetStringSlice(pluginsFlag); err != nil {
		return fmt.Errorf("invalid flag %q: %w", pluginsFlag, err)
	} else if len(pluginKeys) != 0 {
		// Filter out help flags that may have been incorrectly parsed as plugin values
		// This fixes the issue where "kubebuilder edit --plugins --help" treats --help as a plugin
		validPluginKeys := make([]string, 0, len(pluginKeys))
		helpRequested := false
		for _, key := range pluginKeys {
			key = strings.TrimSpace(key)
			// Skip help flags
			if isHelpFlag(key) {
				helpRequested = true
				continue
			}
			validPluginKeys = append(validPluginKeys, key)
		}

		// If help was requested via --plugins flag, set the help flag to trigger Cobra's help display
		// This prevents command execution and shows help instead
		if helpRequested {
			if err := fs.Set("help", "true"); err == nil {
				return nil
			}
			// If setting help flag fails, still return nil to avoid validation errors
			return nil
		}

		// Validate the remaining plugin keys
		for i, key := range validPluginKeys {
			if err := plugin.ValidateKey(key); err != nil {
				return fmt.Errorf("invalid plugin %q found in flags: %w", validPluginKeys[i], err)
			}
		}

		c.pluginKeys = validPluginKeys
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
	"https://kubebuilder.io/plugins/plugins-versioning)"

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
				return fmt.Errorf("error parsing input plugin version from key %q: %w", pluginKey, err)
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

	// kubebuilder delete (plugin delete only)
	c.cmd.AddCommand(c.newDeleteCmd())

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
		if p == nil {
			continue
		}
		if deprecated, ok := p.(plugin.Deprecated); ok && len(deprecated.DeprecationWarning()) > 0 {
			_, _ = fmt.Fprintf(os.Stderr, noticeColor, fmt.Sprintf(deprecationFmt, deprecated.DeprecationWarning()))
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
	if err := c.cmd.Execute(); err != nil {
		// Don't return error if help was displayed (from --plugins --help pattern)
		if err == errHelpDisplayed {
			return nil
		}
		return fmt.Errorf("error executing command: %w", err)
	}

	return nil
}

// Command returns the underlying root command.
func (c CLI) Command() *cobra.Command {
	return c.cmd
}
