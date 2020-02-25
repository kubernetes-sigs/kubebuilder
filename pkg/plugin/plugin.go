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
	"path"
	"strings"

	"github.com/spf13/pflag"
)

type Base interface {
	// Name returns a DNS1123 label string defining the plugin type.
	// For example, Kubebuilder's main plugin would return "go".
	//
	// TODO: fully-qualified automatic append and comparison.
	Name() string
	// Version returns the plugin's semantic version, ex. "v1.2.3".
	//
	// Note: this version is different from config version.
	Version() string
	// SupportedProjectVersions lists all project configuration versions this
	// plugin supports, ex. []string{"2", "3"}. The returned slice cannot be empty.
	SupportedProjectVersions() []string
}

// Key returns a Base plugin's unique identifying string.
func Key(p Base) string {
	return path.Join(p.Name(), "v"+strings.TrimLeft(p.Version(), "v"))
}

type Deprecated interface {
	// DeprecationWarning returns a string indicating a plugin is deprecated.
	DeprecationWarning() string
}

type GenericSubcommand interface {
	// UpdateContext updates a Context with command-specific help text, like description and examples.
	// Can be a no-op if default help text is desired.
	UpdateContext(*Context)
	// BindFlags binds the plugin's flags to the CLI. This allows each plugin to define its own
	// command line flags for the kubebuilder subcommand.
	BindFlags(*pflag.FlagSet)
	// Run runs the subcommand.
	Run() error
}

type Context struct {
	// CommandName sets the command name for a plugin.
	CommandName string
	// Description is a description of what this subcommand does. It is used to display help.
	Description string
	// Examples are one or more examples of the command-line usage
	// of this plugin's project subcommand support. It is used to display help.
	Examples string
}

type InitPluginGetter interface {
	Base
	// GetInitPlugin returns the underlying Init interface.
	GetInitPlugin() Init
}

type Init interface {
	GenericSubcommand
	// SetVersion injects the version a project is initialized with into the
	// plugin.
	SetVersion(string)
}

type CreateAPIPluginGetter interface {
	Base
	// GetCreateAPIPlugin returns the underlying CreateAPI interface.
	GetCreateAPIPlugin() CreateAPI
}

type CreateAPI interface {
	GenericSubcommand
}

type CreateWebhookPluginGetter interface {
	Base
	// GetCreateWebhookPlugin returns the underlying CreateWebhook interface.
	GetCreateWebhookPlugin() CreateWebhook
}

type CreateWebhook interface {
	GenericSubcommand
}
