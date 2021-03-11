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

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/config/store"
	yamlstore "sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
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

// filterSubcommands returns a list of plugin keys and subcommands from a filtered list of resolved plugins.
func (c CLI) filterSubcommands(
	filter func(plugin.Plugin) bool,
	extract func(plugin.Plugin) plugin.Subcommand,
) ([]string, *[]plugin.Subcommand) {
	// Unbundle plugins
	plugins := make([]plugin.Plugin, 0, len(c.resolvedPlugins))
	for _, p := range c.resolvedPlugins {
		if bundle, isBundle := p.(plugin.Bundle); isBundle {
			plugins = append(plugins, bundle.Plugins()...)
		} else {
			plugins = append(plugins, p)
		}
	}

	pluginKeys := make([]string, 0, len(plugins))
	subcommands := make([]plugin.Subcommand, 0, len(plugins))
	for _, p := range plugins {
		if filter(p) {
			pluginKeys = append(pluginKeys, plugin.KeyFor(p))
			subcommands = append(subcommands, extract(p))
		}
	}
	return pluginKeys, &subcommands
}

// initializationMethods
func (c CLI) initializationMethods(cmd *cobra.Command, subcommands *[]plugin.Subcommand) *resourceOptions {
	// Update metadata method.
	meta := plugin.SubcommandMetadata{
		Description: cmd.Long,
		Examples:    cmd.Example,
	}
	for _, subcommand := range *subcommands {
		if subcmd, updatesMetadata := subcommand.(plugin.UpdatesMetadata); updatesMetadata {
			subcmd.UpdateMetadata(c.metadata(), &meta)
		}
	}
	cmd.Long = meta.Description
	cmd.Example = meta.Examples

	// Before binding specific plugin flags, bind common ones
	requiresResource := false
	for _, subcommand := range *subcommands {
		if _, requiresResource = subcommand.(plugin.RequiresResource); requiresResource {
			break
		}
	}
	var options *resourceOptions
	if requiresResource {
		options = bindResourceFlags(cmd.Flags())
	}

	// Bind flags method.
	for _, subcommand := range *subcommands {
		if subcmd, hasFlags := subcommand.(plugin.HasFlags); hasFlags {
			subcmd.BindFlags(cmd.Flags())
		}
	}

	return options
}

// executionMethodsFuncs returns cobra RunE functions for PreRunE, RunE, PostRunE cobra hooks.
func (c CLI) executionMethodsFuncs(
	pluginKeys []string,
	subcommands *[]plugin.Subcommand,
	options *resourceOptions,
	msg string,
) (
	func(*cobra.Command, []string) error,
	func(*cobra.Command, []string) error,
	func(*cobra.Command, []string) error,
) {
	cfg := yamlstore.New(c.fs)
	return executionMethodsPreRunEFunc(pluginKeys, subcommands, cfg, options, c.fs, msg),
		executionMethodsRunEFunc(pluginKeys, subcommands, c.fs, msg),
		executionMethodsPostRunEFunc(pluginKeys, subcommands, cfg, msg)
}

// executionMethodsPreRunEFunc returns a cobra RunE function that loads the configuration
// and executes inject config, inject resource and pre-scaffold methods.
func executionMethodsPreRunEFunc(
	pluginKeys []string,
	subcommands *[]plugin.Subcommand,
	cfg store.Store,
	options *resourceOptions,
	fs afero.Fs,
	msg string,
) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		if err := cfg.Load(); os.IsNotExist(err) {
			return fmt.Errorf("%s: unable to find configuration file, project must be initialized", msg)
		} else if err != nil {
			return fmt.Errorf("%s: unable to load configuration file: %w", msg, err)
		}

		var res *resource.Resource
		if options != nil {
			options.Domain = cfg.Config().GetDomain() // TODO: offer a flag instead of hard-coding project-wide domain
			if err := options.validate(); err != nil {
				return fmt.Errorf("%s: unable to create resource: %w", msg, err)
			}
			res = options.newResource()
		}

		// Inject config method.
		subcommandsCopy := make([]plugin.Subcommand, len(*subcommands))
		copy(subcommandsCopy, *subcommands)
		for i, subcommand := range subcommandsCopy {
			if subcmd, requiresConfig := subcommand.(plugin.RequiresConfig); requiresConfig {
				if err := subcmd.InjectConfig(cfg.Config()); err != nil {
					var exitError plugin.ExitError
					if errors.As(err, &exitError) {
						fmt.Printf("skipping %q: %s\n", pluginKeys[i], exitError.Reason)
						*subcommands = append((*subcommands)[:i], (*subcommands)[i+1:]...)
					} else {
						return fmt.Errorf("%s: unable to inject the configuration to %q: %w", msg, pluginKeys[i], err)
					}
				}
			}
		}

		// Inject resource method.
		if res != nil {
			subcommandsCopy = make([]plugin.Subcommand, len(*subcommands))
			copy(subcommandsCopy, *subcommands)
			for i, subcommand := range subcommandsCopy {
				if subcmd, requiresResource := subcommand.(plugin.RequiresResource); requiresResource {
					if err := subcmd.InjectResource(res); err != nil {
						var exitError plugin.ExitError
						if errors.As(err, &exitError) {
							fmt.Printf("skipping %q: %s\n", pluginKeys[i], exitError.Reason)
							*subcommands = append((*subcommands)[:i], (*subcommands)[i+1:]...)
						} else {
							return fmt.Errorf("%s: unable to inject the resource to %q: %w", msg, pluginKeys[i], err)
						}
					}
				}
			}
			if err := res.Validate(); err != nil {
				return fmt.Errorf("%s: created invalid resource: %w", msg, err)
			}
		}

		// Pre-scaffold method.
		subcommandsCopy = make([]plugin.Subcommand, len(*subcommands))
		copy(subcommandsCopy, *subcommands)
		for i, subcommand := range subcommandsCopy {
			if subcmd, hasPreScaffold := subcommand.(plugin.HasPreScaffold); hasPreScaffold {
				if err := subcmd.PreScaffold(fs); err != nil {
					var exitError plugin.ExitError
					if errors.As(err, &exitError) {
						fmt.Printf("skipping %q: %s\n", pluginKeys[i], exitError.Reason)
						*subcommands = append((*subcommands)[:i], (*subcommands)[i+1:]...)
					} else {
						return fmt.Errorf("%s: unable to run pre-scaffold tasks of %q: %w", msg, pluginKeys[i], err)
					}
				}
			}
		}

		return nil
	}
}

// executionMethodsRunEFunc returns a cobra RunE function that executes the scaffold method.
func executionMethodsRunEFunc(
	pluginKeys []string,
	subcommands *[]plugin.Subcommand,
	fs afero.Fs,
	msg string,
) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		// Scaffold method.
		subcommandsCopy := make([]plugin.Subcommand, len(*subcommands))
		copy(subcommandsCopy, *subcommands)
		for i, subcommand := range subcommandsCopy {
			if err := subcommand.Scaffold(fs); err != nil {
				var exitError plugin.ExitError
				if errors.As(err, &exitError) {
					fmt.Printf("skipping %q: %s\n", pluginKeys[i], exitError.Reason)
					*subcommands = append((*subcommands)[:i], (*subcommands)[i+1:]...)
				} else {
					return fmt.Errorf("%s: unable to scaffold with %q: %v", msg, pluginKeys[i], err)
				}
			}
		}

		return nil
	}
}

// executionMethodsPostRunEFunc returns a cobra RunE function that executes the post-scaffold method
// and saves the configuration.
func executionMethodsPostRunEFunc(
	pluginKeys []string,
	subcommands *[]plugin.Subcommand,
	cfg store.Store,
	msg string,
) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		err := cfg.Save()
		if err != nil {
			return fmt.Errorf("%s: unable to save configuration file: %w", msg, err)
		}

		// Post-scaffold method.
		subcommandsCopy := make([]plugin.Subcommand, len(*subcommands))
		copy(subcommandsCopy, *subcommands)
		for i, subcommand := range subcommandsCopy {
			if subcmd, hasPostScaffold := subcommand.(plugin.HasPostScaffold); hasPostScaffold {
				if err := subcmd.PostScaffold(); err != nil {
					var exitError plugin.ExitError
					if errors.As(err, &exitError) {
						fmt.Printf("skipping %q: %s\n", pluginKeys[i], exitError.Reason)
						*subcommands = append((*subcommands)[:i], (*subcommands)[i+1:]...)
					} else {
						return fmt.Errorf("%s: unable to run post-scaffold tasks of %q: %w", msg, pluginKeys[i], err)
					}
				}
			}
		}

		return nil
	}
}
