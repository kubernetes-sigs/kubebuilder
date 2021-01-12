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

package plugin

import (
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
)

// Plugin is an interface that defines the common base for all plugins
type Plugin interface {
	// Name returns a DNS1123 label string identifying the plugin uniquely. This name should be fully-qualified,
	// i.e. have a short prefix describing the plugin type (like a language) followed by a domain.
	// For example, Kubebuilder's main plugin would return "go.kubebuilder.io".
	Name() string
	// Version returns the plugin's version.
	//
	// NOTE: this version is different from config version.
	Version() Version
	// SupportedProjectVersions lists all project configuration versions this plugin supports.
	// The returned slice cannot be empty.
	SupportedProjectVersions() []config.Version
}

// Deprecated is an interface that defines the messages for plugins that are deprecated.
type Deprecated interface {
	// DeprecationWarning returns a string indicating a plugin is deprecated.
	DeprecationWarning() string
}

// Subcommand is an interface that defines the common base for subcommands returned by plugins
type Subcommand interface {
	// UpdateMetadata injects CLI meta-data into the Subcommand and returns the Subcommand's metadata for the CLI.
	// Any zero field in the returned object will use the default value.
	UpdateMetadata(CLIMetadata) CommandMetadata
	// BindFlags binds the subcommand's flags to the CLI. This allows each subcommand to define its own
	// command line flags.
	BindFlags(*pflag.FlagSet)
	// Run runs the subcommand.
	Run() error
	// InjectConfig passes a config to a plugin. The plugin may modify the config.
	// Initializing, loading, and saving the config is managed by the cli package.
	InjectConfig(config.Config)
}

// Init is an interface for plugins that provide an `init` subcommand
type Init interface {
	Plugin
	// GetInitSubcommand returns the underlying InitSubcommand interface.
	GetInitSubcommand() InitSubcommand
}

// InitSubcommand is an interface that represents an `init` subcommand
type InitSubcommand interface {
	Subcommand
}

// CreateAPI is an interface for plugins that provide a `create api` subcommand
type CreateAPI interface {
	Plugin
	// GetCreateAPISubcommand returns the underlying CreateAPISubcommand interface.
	GetCreateAPISubcommand() CreateAPISubcommand
}

// CreateAPISubcommand is an interface that represents a `create api` subcommand
type CreateAPISubcommand interface {
	Subcommand
}

// CreateWebhook is an interface for plugins that provide a `create webhook` subcommand
type CreateWebhook interface {
	Plugin
	// GetCreateWebhookSubcommand returns the underlying CreateWebhookSubcommand interface.
	GetCreateWebhookSubcommand() CreateWebhookSubcommand
}

// CreateWebhookSubcommand is an interface that represents a `create wekbhook` subcommand
type CreateWebhookSubcommand interface {
	Subcommand
}

// Edit is an interface for plugins that provide a `edit` subcommand
type Edit interface {
	Plugin
	// GetEditSubcommand returns the underlying EditSubcommand interface.
	GetEditSubcommand() EditSubcommand
}

// EditSubcommand is an interface that represents an `edit` subcommand
type EditSubcommand interface {
	Subcommand
}

// Full is an interface for plugins that provide `init`, `create api`, `create webhook` and `edit` subcommands
type Full interface {
	Init
	CreateAPI
	CreateWebhook
	Edit
}
