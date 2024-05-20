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
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
)

// Plugin is an interface that defines the common base for all plugins.
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

// Init is an interface for plugins that provide an `init` subcommand.
type Init interface {
	Plugin
	// GetInitSubcommand returns the underlying InitSubcommand interface.
	GetInitSubcommand() InitSubcommand
}

// CreateAPI is an interface for plugins that provide a `create api` subcommand.
type CreateAPI interface {
	Plugin
	// GetCreateAPISubcommand returns the underlying CreateAPISubcommand interface.
	GetCreateAPISubcommand() CreateAPISubcommand
}

// CreateWebhook is an interface for plugins that provide a `create webhook` subcommand.
type CreateWebhook interface {
	Plugin
	// GetCreateWebhookSubcommand returns the underlying CreateWebhookSubcommand interface.
	GetCreateWebhookSubcommand() CreateWebhookSubcommand
}

// Edit is an interface for plugins that provide a `edit` subcommand.
type Edit interface {
	Plugin
	// GetEditSubcommand returns the underlying EditSubcommand interface.
	GetEditSubcommand() EditSubcommand
}

// Full is an interface for plugins that provide `init`, `create api`, `create webhook` and `edit` subcommands.
type Full interface {
	Init
	CreateAPI
	CreateWebhook
	Edit
}

// Bundle allows to group plugins under a single key.
type Bundle interface {
	Plugin
	// Plugins returns a list of the bundled plugins.
	// The returned list should be flattened, i.e., no plugin bundles should be part of this list.
	Plugins() []Plugin
}
