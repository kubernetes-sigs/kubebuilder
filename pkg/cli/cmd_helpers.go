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

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/config/store"
	yamlstore "sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
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

// keySubcommandTuple represents a pairing of the key of a plugin with a plugin.Subcommand.
type keySubcommandTuple struct {
	key        string
	subcommand plugin.Subcommand

	// skip will be used to flag subcommands that should be skipped after any hook returned a plugin.ExitError.
	skip bool
}

// filterSubcommands returns a list of plugin keys and subcommands from a filtered list of resolved plugins.
func (c *CLI) filterSubcommands(
	filter func(plugin.Plugin) bool,
	extract func(plugin.Plugin) plugin.Subcommand,
) []keySubcommandTuple {
	// Unbundle plugins
	plugins := make([]plugin.Plugin, 0, len(c.resolvedPlugins))
	for _, p := range c.resolvedPlugins {
		if bundle, isBundle := p.(plugin.Bundle); isBundle {
			plugins = append(plugins, bundle.Plugins()...)
		} else {
			plugins = append(plugins, p)
		}
	}

	tuples := make([]keySubcommandTuple, 0, len(plugins))
	for _, p := range plugins {
		if filter(p) {
			tuples = append(tuples, keySubcommandTuple{
				key:        plugin.KeyFor(p),
				subcommand: extract(p),
			})
		}
	}
	return tuples
}

// applySubcommandHooks runs the initialization hooks and configures the commands pre-run,
// run, and post-run hooks with the appropriate execution hooks.
func (c *CLI) applySubcommandHooks(
	cmd *cobra.Command,
	subcommands []keySubcommandTuple,
	errorMessage string,
	createConfig bool,
) {
	// In case we create a new project configuration we need to compute the plugin chain.
	pluginChain := make([]string, 0, len(c.resolvedPlugins))
	if createConfig {
		// We extract the plugin keys again instead of using the ones obtained when filtering subcommands
		// as these plugins are unbundled but we want to keep bundle names in the plugin chain.
		for _, p := range c.resolvedPlugins {
			pluginChain = append(pluginChain, plugin.KeyFor(p))
		}
	}

	options := initializationHooks(cmd, subcommands, c.metadata())

	factory := executionHooksFactory{
		fs:             c.fs,
		store:          yamlstore.New(c.fs),
		subcommands:    subcommands,
		errorMessage:   errorMessage,
		projectVersion: c.projectVersion,
		pluginChain:    pluginChain,
	}
	cmd.PreRunE = factory.preRunEFunc(options, createConfig)
	cmd.RunE = factory.runEFunc()
	cmd.PostRunE = factory.postRunEFunc()
}

// initializationHooks executes update metadata and bind flags plugin hooks.
func initializationHooks(
	cmd *cobra.Command,
	subcommands []keySubcommandTuple,
	meta plugin.CLIMetadata,
) *resourceOptions {
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

	// Bind flags hook.
	for _, tuple := range subcommands {
		if subcommand, hasFlags := tuple.subcommand.(plugin.HasFlags); hasFlags {
			subcommand.BindFlags(cmd.Flags())
		}
	}

	return options
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
}

func (factory *executionHooksFactory) forEach(cb func(subcommand plugin.Subcommand) error, errorMessage string) error {
	for i, tuple := range factory.subcommands {
		if tuple.skip {
			continue
		}

		err := cb(tuple.subcommand)

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

// preRunEFunc returns a cobra RunE function that loads the configuration, creates the resource,
// and executes inject config, inject resource, and pre-scaffold hooks.
func (factory *executionHooksFactory) preRunEFunc(
	options *resourceOptions,
	createConfig bool,
) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
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
				return fmt.Errorf("%s: unable to find configuration file, project must be initialized",
					factory.errorMessage)
			} else if err != nil {
				return fmt.Errorf("%s: unable to load configuration file: %w", factory.errorMessage, err)
			}
		}
		cfg := factory.store.Config()

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
				return fmt.Errorf("%s: unable to create resource: %w", factory.errorMessage, err)
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
		// nolint:revive
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
		// nolint:revive
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
			return fmt.Errorf("%s: unable to save configuration file: %w", factory.errorMessage, err)
		}

		// Post-scaffold hook.
		// nolint:revive
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
