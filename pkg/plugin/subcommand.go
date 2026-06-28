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
	"github.com/spf13/cobra"
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

// MarksRequiredFlags is an optional interface for subcommands that need to
// mark flags as required after they are bound. This allows plugin-specific
// required-flag logic that varies between subcommands (e.g., --version and
// --kind are required for create api but not for create webhook standalone mode).
type MarksRequiredFlags interface {
	// MarkRequiredFlags marks specific flags as required on the given command.
	MarkRequiredFlags(*cobra.Command) error
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

// RequiresStandaloneWebhook is an optional interface for subcommands that support
// multi-GVK webhooks (webhooks not tied to a single API resource).
type RequiresStandaloneWebhook interface {
	// InjectStandaloneWebhook injects the standalone webhook configuration into the subcommand.
	InjectStandaloneWebhook(*resource.StandaloneWebhook) error
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

// EditSubcommand is an interface that represents an `edit` subcommand.
type EditSubcommand interface {
	Subcommand
}
