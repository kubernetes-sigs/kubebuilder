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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	yamlstore "sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

// noResolvedPluginError is returned by subcommands that require a plugin when none was resolved.
type noResolvedPluginError struct{}

// Error implements error interface.
func (e noResolvedPluginError) Error() string {
	return "no resolved plugin, please verify the project version and plugins specified in flags or configuration file"
}

// noAvailablePluginError is returned by subcommands that require a plugin when none of their specific type was found.
type noAvailablePluginError struct {
	subcommand string
}

// Error implements error interface.
func (e noAvailablePluginError) Error() string {
	return fmt.Sprintf("resolved plugins do not provide any %s subcommand", e.subcommand)
}

// cmdErr updates a cobra command to output error information when executed
// or used with the help flag.
func cmdErr(cmd *cobra.Command, err error) {
	cmd.Long = fmt.Sprintf("%s\nNote: %v", cmd.Long, err)
	cmd.RunE = errCmdFunc(err)
}

// errCmdFunc returns a cobra RunE function that returns the provided error
func errCmdFunc(err error) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		return err
	}
}

// keySubcommandTuple pairs a plugin key with its subcommand.
// key is the plugin's own key, configKey is the bundle key (if wrapped in a bundle).
type keySubcommandTuple struct {
	key        string
	configKey  string
	subcommand plugin.Subcommand

	// skip marks subcommands that should be skipped after a plugin.ExitError.
	skip bool
}

type pluginChainSetter interface {
	SetPluginChain([]string)
}

// filterSubcommands returns plugin keys and subcommands from resolved plugins.
func (c *CLI) filterSubcommands(
	filter func(plugin.Plugin) bool,
	extract func(plugin.Plugin) plugin.Subcommand,
) []keySubcommandTuple {
	tuples := make([]keySubcommandTuple, 0, len(c.resolvedPlugins))
	for _, p := range c.resolvedPlugins {
		tuples = append(tuples, collectSubcommands(p, plugin.KeyFor(p), filter, extract)...)
	}
	return tuples
}

func collectSubcommands(
	p plugin.Plugin,
	configKey string,
	filter func(plugin.Plugin) bool,
	extract func(plugin.Plugin) plugin.Subcommand,
) []keySubcommandTuple {
	if bundle, isBundle := p.(plugin.Bundle); isBundle {
		collected := make([]keySubcommandTuple, 0, len(bundle.Plugins()))
		for _, nested := range bundle.Plugins() {
			collected = append(collected, collectSubcommands(nested, configKey, filter, extract)...)
		}
		return collected
	}

	if !filter(p) {
		return nil
	}

	return []keySubcommandTuple{{
		key:        plugin.KeyFor(p),
		configKey:  configKey,
		subcommand: extract(p),
	}}
}

// applySubcommandHooks runs the initialization hooks and wires pre-run, run, and post-run for the command.
// Used by init, create api, create webhook, and edit. When several plugins define the same flag,
// one flag is shown and its value is synced to all plugins after parse.
func (c *CLI) applySubcommandHooks(
	cmd *cobra.Command,
	subcommands []keySubcommandTuple,
	errorMessage string,
	createConfig bool,
) {
	commandPluginChain := make([]string, len(subcommands))
	for i, tuple := range subcommands {
		commandPluginChain[i] = tuple.key
	}
	for _, tuple := range subcommands {
		if setter, ok := tuple.subcommand.(pluginChainSetter); ok {
			setter.SetPluginChain(commandPluginChain)
		}
	}

	// In case we create a new project configuration we need to compute the plugin chain.
	pluginChain := make([]string, 0, len(c.resolvedPlugins))
	if createConfig {
		// We extract the plugin keys again instead of using the ones obtained when filtering subcommands
		// as these plugins are unbundled but we want to keep bundle names in the plugin chain.
		for _, p := range c.resolvedPlugins {
			pluginChain = append(pluginChain, plugin.KeyFor(p))
		}
	}

	result, err := initializationHooks(cmd, subcommands, c.metadata())
	if err != nil {
		cmdErr(cmd, err)
		return
	}

	factory := executionHooksFactory{
		fs:                  c.fs,
		store:               yamlstore.New(c.fs),
		subcommands:         subcommands,
		errorMessage:        errorMessage,
		projectVersion:      c.projectVersion,
		pluginChain:         pluginChain,
		cliVersion:          c.cliVersion,
		duplicateFlagValues: result.duplicateFlagValues,
	}
	cmd.PreRunE = factory.preRunEFunc(result.options, createConfig)
	cmd.RunE = factory.runEFunc()
	cmd.PostRunE = factory.postRunEFunc()
}

// appendPluginTable appends a filtered plugin table to the command's Long description.
// For subcommands, it excludes the default scaffold and its component plugins.
func (c *CLI) appendPluginTable(cmd *cobra.Command, filter func(plugin.Plugin) bool, title string) {
	pluginTable := c.getPluginTableFilteredForSubcommand(filter)
	cmd.Long = fmt.Sprintf("%s\n%s:\n\n%s\n", cmd.Long, title, pluginTable)
}

// initHooksResult holds the result of initializationHooks: resource options and
// duplicate-flag values to sync after parse.
type initHooksResult struct {
	options             *resourceOptions
	duplicateFlagValues map[string][]pflag.Value
}

// mergeFlagSetInto merges flags from src into dest using AddFlagSet. If a flag name already exists,
// the flag is not added again; its Value is stored in duplicateValues for later sync and the existing
// Usage is extended. Returns an error if the same flag name is used with a different value type.
func mergeFlagSetInto(
	dest *pflag.FlagSet,
	src *pflag.FlagSet,
	duplicateValues map[string][]pflag.Value,
	pluginKey string,
	firstPluginByFlag map[string]string,
) error {
	destNames := make(map[string]struct{})
	dest.VisitAll(func(f *pflag.Flag) {
		destNames[f.Name] = struct{}{}
	})
	dest.AddFlagSet(src)

	var err error
	src.VisitAll(func(flag *pflag.Flag) {
		if err != nil {
			return
		}
		existing := dest.Lookup(flag.Name)
		if _, wasInDest := destNames[flag.Name]; !wasInDest {
			firstPluginByFlag[flag.Name] = pluginKey
			existing.Usage = "For plugin (" + pluginKey + "): " + strings.TrimSpace(flag.Usage)
			return
		}
		if existing.Value.Type() != flag.Value.Type() {
			firstKey := firstPluginByFlag[flag.Name]
			err = fmt.Errorf(
				"plugins %q and %q use the same flag name %q but expect different value types: one %s, other %s",
				firstKey, pluginKey, flag.Name, existing.Value.Type(), flag.Value.Type(),
			)
			return
		}
		duplicateValues[flag.Name] = append(duplicateValues[flag.Name], flag.Value)
		existing.Usage += " AND for plugin (" + pluginKey + "): " + strings.TrimSpace(flag.Usage)
	})
	return err
}

// syncDuplicateFlags copies the parsed value of each flag to all duplicate Values from merge.
// Call after the command has parsed flags (e.g. at the start of PreRunE).
func syncDuplicateFlags(flags *pflag.FlagSet, duplicateValues map[string][]pflag.Value) {
	for name, values := range duplicateValues {
		parsed := flags.Lookup(name)
		if parsed == nil {
			continue
		}
		srcVal := parsed.Value.String()
		for _, v := range values {
			_ = v.Set(srcVal)
		}
	}
}

// initializationHooks runs update-metadata and bind-flags hooks. When multiple plugins bind the same
// flag, one flag is used and its value is synced to all after parse; usage text is aggregated.
// Returns an error if the same flag name is used with different value types (e.g. bool vs string).
func initializationHooks(
	cmd *cobra.Command,
	subcommands []keySubcommandTuple,
	meta plugin.CLIMetadata,
) (*initHooksResult, error) {
	// Update metadata hook.
	subcmdMeta := plugin.SubcommandMetadata{
		Description: cmd.Long,
		Examples:    cmd.Example,
	}
	for _, tuple := range subcommands {
		if subcommand, updatesMetadata := tuple.subcommand.(plugin.UpdatesMetadata); updatesMetadata {
			subcommand.UpdateMetadata(meta, &subcmdMeta)
		}
	}
	cmd.Long = subcmdMeta.Description
	cmd.Example = subcmdMeta.Examples

	// Before binding specific plugin flags, bind common ones.
	requiresResource := false
	for _, tuple := range subcommands {
		if _, requiresResource = tuple.subcommand.(plugin.RequiresResource); requiresResource {
			break
		}
	}
	var options *resourceOptions
	if requiresResource {
		options = bindResourceFlags(cmd.Flags())
	}

	// Bind flags hook: each plugin binds to a temporary FlagSet, then we merge into the command so
	// duplicate names do not panic; values are synced after parse and help text is aggregated.
	duplicateValues := make(map[string][]pflag.Value)
	firstPluginByFlag := make(map[string]string)
	for _, tuple := range subcommands {
		if subcommand, hasFlags := tuple.subcommand.(plugin.HasFlags); hasFlags {
			tmpSet := pflag.NewFlagSet(cmd.Name(), pflag.ExitOnError)
			subcommand.BindFlags(tmpSet)
			if err := mergeFlagSetInto(cmd.Flags(), tmpSet, duplicateValues, tuple.key, firstPluginByFlag); err != nil {
				return nil, err
			}
		}
	}

	return &initHooksResult{options: options, duplicateFlagValues: duplicateValues}, nil
}

type executionHooksFactory struct {
	// fs is the filesystem abstraction to scaffold files to.
	fs machinery.Filesystem
	// store is the backend used to load/save the project configuration.
	store store.Store
	// subcommands are the tuples representing the set of subcommands provided by the resolved plugins.
	subcommands []keySubcommandTuple
	// errorMessage is prepended to returned errors.
	errorMessage string
	// projectVersion is the project version that will be used to create new project configurations.
	// It is only used for initialization.
	projectVersion config.Version
	// pluginChain is the plugin chain configured for this project.
	pluginChain []string
	// cliVersion is the version of the CLI.
	cliVersion string
	// duplicateFlagValues maps flag names to Values to sync from the parsed flag in PreRunE.
	duplicateFlagValues map[string][]pflag.Value
}

func (factory *executionHooksFactory) forEach(cb func(subcommand plugin.Subcommand) error, errorMessage string) error {
	for i, tuple := range factory.subcommands {
		if tuple.skip {
			continue
		}

		err := factory.withPluginChain(tuple, func() error {
			return cb(tuple.subcommand)
		})

		var exitError plugin.ExitError
		switch {
		case err == nil:
			// No error do nothing
		case errors.As(err, &exitError):
			// Exit errors imply that no further hooks of this subcommand should be called, so we flag it to be skipped
			factory.subcommands[i].skip = true
			fmt.Printf("skipping remaining hooks of %q: %s\n", tuple.key, exitError.Reason)
		default:
			// Any other error, wrap it
			return fmt.Errorf("%s: %s %q: %w", factory.errorMessage, errorMessage, tuple.key, err)
		}
	}

	return nil
}

func (factory *executionHooksFactory) withPluginChain(tuple keySubcommandTuple, cb func() error) (err error) {
	if tuple.configKey == "" {
		return cb()
	}

	cfg := factory.store.Config()
	if cfg == nil {
		return cb()
	}

	// Temporarily move configKey to the front so GetPluginKeyForConfig finds it first.
	// This ensures each bundled plugin saves config under the right key.
	original := append([]string(nil), cfg.GetPluginChain()...)
	newChain := moveKeyToFront(original, tuple.configKey)
	changed := !equalStringSlices(original, newChain)
	if changed {
		if setErr := cfg.SetPluginChain(newChain); setErr != nil {
			return fmt.Errorf("failed to set plugin chain for %q: %w", tuple.configKey, setErr)
		}
		defer func() {
			if resetErr := cfg.SetPluginChain(original); resetErr != nil && err == nil {
				err = fmt.Errorf("failed to reset plugin chain: %w", resetErr)
			}
		}()
	}

	return cb()
}

func moveKeyToFront(chain []string, key string) []string {
	if len(chain) == 0 {
		return []string{key}
	}

	if chain[0] == key {
		return chain
	}

	newChain := make([]string, 0, len(chain)+1)
	newChain = append(newChain, key)
	for _, existing := range chain {
		if existing == key {
			continue
		}
		newChain = append(newChain, existing)
	}
	return newChain
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// preRunEFunc returns a cobra RunE function that loads the configuration, creates the resource,
// and executes inject config, inject resource, and pre-scaffold hooks.
func (factory *executionHooksFactory) preRunEFunc(
	options *resourceOptions,
	createConfig bool,
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if len(factory.duplicateFlagValues) > 0 {
			syncDuplicateFlags(cmd.Flags(), factory.duplicateFlagValues)
		}
		if createConfig {
			// Check if a project configuration is already present.
			if err := factory.store.Load(); err == nil || !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("%s: already initialized", factory.errorMessage)
			}

			// Initialize the project configuration.
			if err := factory.store.New(factory.projectVersion); err != nil {
				return fmt.Errorf("%s: error initializing project configuration: %w", factory.errorMessage, err)
			}
		} else {
			// Load the project configuration.
			if err := factory.store.Load(); os.IsNotExist(err) {
				return fmt.Errorf("%s: failed to find configuration file, project must be initialized",
					factory.errorMessage)
			} else if err != nil {
				return fmt.Errorf("%s: failed to load configuration file: %w", factory.errorMessage, err)
			}
		}
		cfg := factory.store.Config()

		// Set the CLI version if creating a new project configuration.
		if createConfig {
			_ = cfg.SetCliVersion(factory.cliVersion)
		}

		// Set the pluginChain field.
		if len(factory.pluginChain) != 0 {
			_ = cfg.SetPluginChain(factory.pluginChain)
		}

		// Create the resource if non-nil options provided
		var res *resource.Resource
		if options != nil {
			// TODO: offer a flag instead of hard-coding project-wide domain
			options.Domain = cfg.GetDomain()
			if err := options.validate(); err != nil {
				return fmt.Errorf("%s: failed to create resource: %w", factory.errorMessage, err)
			}
			res = options.newResource()
		}

		// Inject config hook.
		if err := factory.forEach(func(subcommand plugin.Subcommand) error {
			if subcommand, requiresConfig := subcommand.(plugin.RequiresConfig); requiresConfig {
				return subcommand.InjectConfig(cfg)
			}
			return nil
		}, "unable to inject the configuration to"); err != nil {
			return err
		}

		if res != nil {
			// Inject resource hook.
			if err := factory.forEach(func(subcommand plugin.Subcommand) error {
				if subcommand, requiresResource := subcommand.(plugin.RequiresResource); requiresResource {
					return subcommand.InjectResource(res)
				}
				return nil
			}, "unable to inject the resource to"); err != nil {
				return err
			}

			if err := res.Validate(); err != nil {
				return fmt.Errorf("%s: created invalid resource: %w", factory.errorMessage, err)
			}
		}

		// Pre-scaffold hook.
		//nolint:revive
		if err := factory.forEach(func(subcommand plugin.Subcommand) error {
			if subcommand, hasPreScaffold := subcommand.(plugin.HasPreScaffold); hasPreScaffold {
				return subcommand.PreScaffold(factory.fs)
			}
			return nil
		}, "unable to run pre-scaffold tasks of"); err != nil {
			return err
		}

		return nil
	}
}

// runEFunc returns a cobra RunE function that executes the scaffold hook.
func (factory *executionHooksFactory) runEFunc() func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		// Scaffold hook.
		//nolint:revive
		if err := factory.forEach(func(subcommand plugin.Subcommand) error {
			return subcommand.Scaffold(factory.fs)
		}, "unable to scaffold with"); err != nil {
			return err
		}

		return nil
	}
}

// postRunEFunc returns a cobra RunE function that saves the configuration
// and executes the post-scaffold hook.
func (factory *executionHooksFactory) postRunEFunc() func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		if err := factory.store.Save(); err != nil {
			return fmt.Errorf("%s: failed to save configuration file: %w", factory.errorMessage, err)
		}

		// Post-scaffold hook.
		//nolint:revive
		if err := factory.forEach(func(subcommand plugin.Subcommand) error {
			if subcommand, hasPostScaffold := subcommand.(plugin.HasPostScaffold); hasPostScaffold {
				return subcommand.PostScaffold()
			}
			return nil
		}, "unable to run post-scaffold tasks of"); err != nil {
			return err
		}

		return nil
	}
}
