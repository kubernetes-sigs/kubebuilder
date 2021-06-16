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

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

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
