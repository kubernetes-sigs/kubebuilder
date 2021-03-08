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

// Option is a function that can configure the cli
type Option func(*cli) error

// WithCommandName is an Option that sets the cli's root command name.
func WithCommandName(name string) Option {
	return func(c *cli) error {
		c.commandName = name
		return nil
	}
}

// WithVersion is an Option that defines the version string of the cli.
func WithVersion(version string) Option {
	return func(c *cli) error {
		c.version = version
		return nil
	}
}

// WithDefaultProjectVersion is an Option that sets the cli's default project version.
// Setting an unknown version will result in an error.
func WithDefaultProjectVersion(version config.Version) Option {
	return func(c *cli) error {
		if err := version.Validate(); err != nil {
			return fmt.Errorf("broken pre-set default project version %q: %v", version, err)
		}
		c.defaultProjectVersion = version
		return nil
	}
}

// WithDefaultPlugins is an Option that sets the cli's default plugins.
func WithDefaultPlugins(plugins ...plugin.Plugin) Option {
	return func(c *cli) error {
		if len(plugins) == 0 {
			return fmt.Errorf("empty set of plugins provided")
		}
		for _, p := range plugins {
			if err := plugin.Validate(p); err != nil {
				return fmt.Errorf("broken pre-set default plugin %q: %v", plugin.KeyFor(p), err)
			}
			c.defaultPlugins = append(c.defaultPlugins, plugin.KeyFor(p))
		}
		return nil
	}
}

// WithPlugins is an Option that sets the cli's plugins.
func WithPlugins(plugins ...plugin.Plugin) Option {
	return func(c *cli) error {
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

// WithExtraCommands is an Option that adds extra subcommands to the cli.
// Adding extra commands that duplicate existing commands results in an error.
func WithExtraCommands(cmds ...*cobra.Command) Option {
	return func(c *cli) error {
		c.extraCommands = append(c.extraCommands, cmds...)
		return nil
	}
}

// WithCompletion is an Option that adds the completion subcommand.
func WithCompletion() Option {
	return func(c *cli) error {
		c.completionCommand = true
		return nil
	}
}
