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

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// UpdatesMetadata is an interface that implements the optional metadata update method.
type UpdatesMetadata interface {
	// UpdateMetadata updates the subcommand metadata.
	UpdateMetadata(CLIMetadata, *SubcommandMetadata)
}

// HasFlags is an interface that implements the optional bind flags method.
type HasFlags interface {
	// BindFlags binds flags to the CLI subcommand.
	BindFlags(*pflag.FlagSet)
}

// RequiresConfig is an interface that implements the optional inject config method.
type RequiresConfig interface {
	// InjectConfig injects the configuration to a subcommand.
	InjectConfig(config.Config) error
}

// RequiresResource is an interface that implements the required inject resource method.
type RequiresResource interface {
	// InjectResource injects the resource model to a subcommand.
	InjectResource(*resource.Resource) error
}

// HasPreScaffold is an interface that implements the optional pre-scaffold method.
type HasPreScaffold interface {
	// PreScaffold executes tasks before the main scaffolding.
	PreScaffold(machinery.Filesystem) error
}

// Scaffolder is an interface that implements the required scaffold method.
type Scaffolder interface {
	// Scaffold implements the main scaffolding.
	Scaffold(machinery.Filesystem) error
}

// HasPostScaffold is an interface that implements the optional post-scaffold method.
type HasPostScaffold interface {
	// PostScaffold executes tasks after the main scaffolding.
	PostScaffold() error
}

// HasPluginChain is an interface that implements the optional plugin chain injection method.
// This allows subcommands to receive the full chain of plugins being executed in the current command,
// enabling cross-plugin coordination and validation.
//
// The plugin chain is automatically injected by the CLI before subcommand execution.
// Plugins should implement this interface if they need to:
//   - Validate that required companion plugins are present in the chain
//   - Coordinate behavior with other plugins in the execution sequence
//   - Check plugin execution order
type HasPluginChain interface {
	// SetPluginChain injects the current plugin chain into the subcommand.
	// The chain represents the ordered list of plugin keys being executed for this command.
	//
	// Example chain:
	// ["go.kubebuilder.io/v4", "kustomize.common.kubebuilder.io/v2", "deploy-image.go.kubebuilder.io/v1-alpha"]
	SetPluginChain(chain []string)
}

// Subcommand is a base interface for all subcommands.
type Subcommand interface {
	Scaffolder
}

// InitSubcommand is an interface that represents an `init` subcommand.
type InitSubcommand interface {
	Subcommand
}

// CreateAPISubcommand is an interface that represents a `create api` subcommand.
type CreateAPISubcommand interface {
	Subcommand
	RequiresResource
}

// CreateWebhookSubcommand is an interface that represents a `create wekbhook` subcommand.
type CreateWebhookSubcommand interface {
	Subcommand
	RequiresResource
}

// DeleteAPISubcommand is an interface that represents a `delete api` subcommand.
type DeleteAPISubcommand interface {
	Subcommand
	RequiresResource
}

// DeleteWebhookSubcommand is an interface that represents a `delete webhook` subcommand.
type DeleteWebhookSubcommand interface {
	Subcommand
	RequiresResource
}

// EditSubcommand is an interface that represents an `edit` subcommand.
type EditSubcommand interface {
	Subcommand
}
