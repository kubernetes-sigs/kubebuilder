# Extensible CLI and Scaffolding Plugins - Phase 1.5

Continuation of [Extensible CLI and Scaffolding Plugins](./extensible-cli-and-scaffolding-plugins-phase-1.md).

## Goal

The goal of this phase is to achieve one of the goals proposed for Phase 2: chaining plugins.
Phase 2 includes several other challenging goals, but being able to chain plugins will be beneficial
for third-party developers that are using kubebuilder as a library.

## Table of contents
- [Goal](#goal)
- [Motivation](#motivation)
- [Proposal](#proposal)
- [Implementation](#implementation)

## Motivation

There are several cases of plugins that want to maintain most of the go plugin functionality and add
certain features on top of it, both inside and outside kubebuilder repository:
- [Addon pattern](../plugins/addon)
- [Operator SDK](https://github.com/operator-framework/operator-sdk/tree/master/internal/plugins/golang)

This behavior fits perfectly under Phase 1.5, where plugins could be chained. However, as this feature is
not available, the adopted temporal solution is to wrap the base go plugin and perform additional actions
after its `Run` method has been executed. This solution faces several issues:

- Wrapper plugins are unable to access the data of the wrapped plugins, as they weren't designed for this
  purpose, and therefore, most of its internal data is non-exported. An example of this inaccessible data
  would be the `Resource` objects created inside the `create api` and `create webhook` commands.
- Wrapper plugins are dependent on their wrapped plugins, and therefore can't be used for other plugins.
- Under the hood, subcommands implement a second hidden interface: `RunOptions`, which further accentuates
  these issues.

Plugin chaining solves the aforementioned problems but the current plugin API, and more specifically the
`Subcommand` interface, does not support plugin chaining.

- The `RunOptions` interface implemented under the hood is not part of the plugin API, and therefore
  the cli is not able to run post-scaffold logic (implemented in `RunOptions.PostScaffold` method) after
  all the plugins have scaffolded their part.
- `Resource`-related commands can't bind flags like `--group`, `--version` or `--kind` in each plugin,
  it must be created outside the plugins and then injected into them similar to the approach followed
  currently for `Config` objects.

## Proposal

Design a Plugin API that combines the current [`Subcommand`](../pkg/plugin/interfaces.go) and
[`RunOptions`](../pkg/plugins/internal/cmdutil/cmdutil.go) interfaces and enables plugin-chaining.
The new `Subcommand` methods can be split in two different categories:
- Initialization methods
- Execution methods

Additionally, some of these methods may be optional, in which case a non-implemented method will be skipped
when it should be called and consider it succeeded. This also allows to create some methods specific for
a certain subcommand call (e.g.: `Resource`-related methods for the `edit` subcommand are not needed).

Different ordering guarantees can be considered:
- Method order guarantee: a method for a plugin will be called after its previous methods succeeded.
- Steps order guarantee: methods will be called when all plugins have finished the previous method.
- Plugin order guarantee: same method for each plugin will be called in the order specified
  by the plugin position at the plugin chain.

All of the methods will offer plugin order guarantee, as they all modify/update some item so the order
of plugins is important. Execution methods need to guarantee step order, as the items that are being modified
in each step (config, resource, and filesystem) are also needed in the following steps. This is not true for
initialization methods that modify items (metadata and flagset) that are only used in their own methods,
so they only need to guarantee method order.

Execution methods will be able to return an error. A specific error can be returned to specify that
no further methods of this plugin should be called, but that the scaffold process should be continued.
This enables plugins to exit early, e.g., a plugin that scaffolds some files only for cluster-scoped
resources can detect if the resource is cluster-scoped at one of the first execution steps, and
therefore, use this error to tell the CLI that no further execution step should be called for itself.

### Initialization methods

#### Update metadata
This method will be used for two purposes. It provides CLI-related metadata to the Subcommand (e.g., 
command name) and update the subcommands metadata such as the description or examples.

- Required/optional
  - [ ] Required
  - [x] Optional
- Subcommands
  - [x] Init
  - [x] Edit
  - [x] Create API
  - [x] Create webhook

#### Bind flags
This method will allow subcommands to define specific flags.

- Required/optional
  - [ ] Required
  - [x] Optional
- Subcommands
  - [x] Init
  - [x] Edit
  - [x] Create API
  - [x] Create webhook

### Execution methods

#### Inject configuration
This method will be used to inject the `Config` object that the plugin can modify at will.
The CLI will create/load/save this configuration object.

- Required/optional
  - [ ] Required
  - [x] Optional
- Subcommands
  - [x] Init
  - [x] Edit
  - [x] Create API
  - [x] Create webhook

#### Inject resource
This method will be used to inject the `Resource` object.

- Required/optional
  - [x] Required
  - [ ] Optional
- Subcommands
  - [ ] Init
  - [ ] Edit
  - [x] Create API
  - [x] Create webhook

#### Pre-scaffold
This method will be used to take actions before the main scaffolding is performed, e.g. validations.

NOTE: a filesystem abstraction will be passed to this method that must be used for scaffolding.

- Required/optional
  - [ ] Required
  - [x] Optional
- Subcommands
  - [x] Init
  - [x] Edit
  - [x] Create API
  - [x] Create webhook

#### Scaffold
This method will be used to perform the main scaffolding.

NOTE: a filesystem abstraction will be passed to this method that must be used for scaffolding.

- Required/optional
  - [x] Required
  - [ ] Optional
- Subcommands
  - [x] Init
  - [x] Edit
  - [x] Create API
  - [x] Create webhook

#### Post-scaffold
This method will be used to take actions after the main scaffolding is performed, e.g. cleanup.

NOTE: a filesystem abstraction will **NOT** be passed to this method, as post-scaffold task do not require it.
In case some post-scaffold task requires a filesystem abstraction, it could be added.

- Required/optional
  - [ ] Required
  - [x] Optional
- Subcommands
  - [x] Init
  - [x] Edit
  - [x] Create API
  - [x] Create webhook

## Implementation

The following types are used as input/output values of the described methods:
```go
// CLIMetadata is the runtime meta-data of the CLI
type CLIMetadata struct {
	// CommandName is the root command name.
	CommandName string
}

// SubcommandMetadata is the runtime meta-data for a subcommand
type SubcommandMetadata struct {
	// Description is a description of what this subcommand does. It is used to display help.
	Description string
	// Examples are one or more examples of the command-line usage of this subcommand. It is used to display help.
	Examples string
}

type ExitError struct {
	Plugin string
	Reason string
}

func (e ExitError) Error() string {
	return fmt.Sprintf("plugin %s exit early: %s", e.Plugin, e.Reason)
}
```

The described methods are implemented through the use of the following interfaces.
```go
type RequiresCLIMetadata interface {
	InjectCLIMetadata(CLIMetadata)
}

type UpdatesSubcommandMetadata interface {
	UpdateSubcommandMetadata(*SubcommandMetadata)
}

type HasFlags interface {
	BindFlags(*pflag.FlagSet)
}

type RequiresConfig interface {
	InjectConfig(config.Config) error
}

type RequiresResource interface {
	InjectResource(*resource.Resource) error
}

type HasPreScaffold interface {
	PreScaffold(afero.Fs) error
}

type Scaffolder interface {
	Scaffold(afero.Fs) error
}

type HasPostScaffold interface {
	PostScaffold() error
}
```

Additional interfaces define the required method for each type of plugin:
```go
// InitSubcommand is the specific interface for subcommands returned by init plugins.
type InitSubcommand interface {
	Scaffolder
}

// EditSubcommand is the specific interface for subcommands returned by edit plugins.
type EditSubcommand interface {
	Scaffolder
}

// CreateAPISubcommand is the specific interface for subcommands returned by create API plugins.
type CreateAPISubcommand interface {
	RequiresResource
	Scaffolder
}

// CreateWebhookSubcommand is the specific interface for subcommands returned by create webhook plugins.
type CreateWebhookSubcommand interface {
	RequiresResource
	Scaffolder
}
```
