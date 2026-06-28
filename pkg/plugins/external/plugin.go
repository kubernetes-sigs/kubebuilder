/*
Copyright 2021 The Kubernetes Authors.

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

package external

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var (
	_ plugin.Full   = Plugin{}
	_ plugin.Delete = Plugin{}
)

// Plugin implements the plugin.Full interface and plugin.Delete.
type Plugin struct {
	PName                     string
	PVersion                  plugin.Version
	PSupportedProjectVersions []config.Version

	// PSupportsDelete declares that the external binary supports the "delete" command.
	// Set this to true when loading the plugin if the binary handles Command:"delete".
	// When false, GetDeleteSubcommand returns a subcommand that fails with a clear error
	// rather than calling the binary.
	PSupportsDelete bool

	Path string
	Args []string
}

// Name returns the name of the plugin
func (p Plugin) Name() string { return p.PName }

// Version returns the version of the plugin
func (p Plugin) Version() plugin.Version { return p.PVersion }

// SupportedProjectVersions returns an array with all project versions supported by the plugin
func (p Plugin) SupportedProjectVersions() []config.Version { return p.PSupportedProjectVersions }

// GetInitSubcommand will return the subcommand which is responsible for initializing and common scaffolding
func (p Plugin) GetInitSubcommand() plugin.InitSubcommand {
	return &initSubcommand{
		Path: p.Path,
		Args: p.Args,
	}
}

// GetCreateAPISubcommand will return the subcommand which is responsible for scaffolding apis
func (p Plugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand {
	return &createAPISubcommand{
		Path: p.Path,
		Args: p.Args,
	}
}

// GetCreateWebhookSubcommand will return the subcommand which is responsible for scaffolding webhooks
func (p Plugin) GetCreateWebhookSubcommand() plugin.CreateWebhookSubcommand {
	return &createWebhookSubcommand{
		Path: p.Path,
		Args: p.Args,
	}
}

// GetEditSubcommand will return the subcommand which is responsible for editing the scaffold of the project
func (p Plugin) GetEditSubcommand() plugin.EditSubcommand {
	return &editSubcommand{
		Path: p.Path,
		Args: p.Args,
	}
}

// GetDeleteSubcommand returns the subcommand that removes files scaffolded by this external plugin.
// The returned subcommand calls the binary with Command:"delete".
// Scaffold fails immediately when PSupportsDelete is false — set it to true and implement the
// "delete" command in the binary to enable this feature.
func (p Plugin) GetDeleteSubcommand() plugin.DeleteSubcommand {
	return &deleteSubcommand{
		Path:           p.Path,
		Args:           p.Args,
		supportsDelete: p.PSupportsDelete,
	}
}

// DeprecationWarning define the deprecation message or return empty when plugin is not deprecated
func (p Plugin) DeprecationWarning() string {
	return ""
}
