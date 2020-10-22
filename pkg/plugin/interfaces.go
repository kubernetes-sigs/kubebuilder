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

	"sigs.k8s.io/kubebuilder/pkg/model/config"
)

// Base is an interface that defines the common base for all plugins
type Base interface {
	// Name returns a DNS1123 label string identifying the plugin uniquely. This name should be fully-qualified,
	// i.e. have a short prefix describing the plugin type (like a language) followed by a domain.
	// For example, Kubebuilder's main plugin would return "go.kubebuilder.io".
	Name() string
	// Version returns the plugin's version, which contains an integer and an optional stability of "alpha" or "beta".
	//
	// Note: this version is different from config version.
	Version() Version
	// SupportedProjectVersions lists all project configuration versions this
	// plugin supports, ex. []string{"2", "3"}. The returned slice cannot be empty.
	SupportedProjectVersions() []string
}

// Deprecated is an interface that defines the messages for plugins that are deprecated.
type Deprecated interface {
	// DeprecationWarning returns a string indicating a plugin is deprecated.
	DeprecationWarning() string
}

// GenericSubcommand is an interface that defines the plugins operations
type GenericSubcommand interface {
	// UpdateContext updates a Context with command-specific help text, like description and examples.
	// Can be a no-op if default help text is desired.
	UpdateContext(*Context)
	// BindFlags binds the plugin's flags to the CLI. This allows each plugin to define its own
	// command line flags for the kubebuilder subcommand.
	BindFlags(*pflag.FlagSet)
	// Run runs the subcommand.
	Run() error
	// InjectConfig passes a config to a plugin. The plugin may modify the
	// config. Initializing, loading, and saving the config is managed by the
	// cli package.
	InjectConfig(*config.Config)
}

// Context is the runtime context for a plugin.
type Context struct {
	// CommandName sets the command name for a plugin.
	CommandName string
	// Description is a description of what this subcommand does. It is used to display help.
	Description string
	// Examples are one or more examples of the command-line usage
	// of this plugin's project subcommand support. It is used to display help.
	Examples string
}

// InitPluginGetter is an interface that defines gets an Init plugin
type InitPluginGetter interface {
	Base
	// GetInitPlugin returns the underlying Init interface.
	GetInitPlugin() Init
}

// Init is an interface that represents an `init` command
type Init interface {
	GenericSubcommand
}

// CreateAPIPluginGetter is an interface that defines gets an Create API plugin
type CreateAPIPluginGetter interface {
	Base
	// GetCreateAPIPlugin returns the underlying CreateAPI interface.
	GetCreateAPIPlugin() CreateAPI
}

// CreateAPI is an interface that represents an `create api` command
type CreateAPI interface {
	GenericSubcommand
}

// CreateWebhookPluginGetter is an interface that defines gets an Create WebHook plugin
type CreateWebhookPluginGetter interface {
	Base
	// GetCreateWebhookPlugin returns the underlying CreateWebhook interface.
	GetCreateWebhookPlugin() CreateWebhook
}

// CreateWebhook is an interface that represents an `create wekbhook` command
type CreateWebhook interface {
	GenericSubcommand
}
