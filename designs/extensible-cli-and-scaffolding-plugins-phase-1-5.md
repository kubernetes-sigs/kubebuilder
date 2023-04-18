| Authors       | Creation Date | Status      | Extra                                                           |
|---------------|---------------|-------------|-----------------------------------------------------------------|
| @adirio | Mar 9, 2021  | Implemented | [Plugins doc](https://book.kubebuilder.io/plugins/plugins.html) |

# Extensible CLI and Scaffolding Plugins - Phase 1.5

Continuation of [Extensible CLI and Scaffolding Plugins](./extensible-cli-and-scaffolding-plugins-phase-1.md).

## Goal

The goal of this phase is to achieve one of the goals proposed for Phase 2: chaining plugins.
Phase 2 includes several other challenging goals, but being able to chain plugins will be beneficial
for third-party developers that are using kubebuilder as a library.

The other main goal of phase 2, discovering and using external plugins, is out of the scope of this phase,
and will be tackled when phase 2 is implemented.

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
The new `Subcommand` hooks can be split in two different categories:
- Initialization hooks
- Execution hooks

Initialization hooks are run during the dynamic creation of the CLI, which means that they are able to
modify the CLI, e.g. providing descriptions and examples for subcommands or binding flags.
Execution hooks are run after the CLI is created, and therefore cannot modify the CLI. On the other hand,
as they are run during the CLI execution, they have access to user-provided flag values, project configuration,
the new API resource or the filesystem abstraction, as opposed to the initialization hooks.

Additionally, some of these hooks may be optional, in which case a non-implemented hook will be skipped
when it should be called and consider it succeeded. This also allows to create some hooks specific for
a certain subcommand call (e.g.: `Resource`-related hooks for the `edit` subcommand are not needed).

Different ordering guarantees can be considered:
- Hook order guarantee: a hook for a plugin will be called after its previous hooks succeeded.
- Steps order guarantee: hooks will be called when all plugins have finished the previous hook.
- Plugin order guarantee: same hook for each plugin will be called in the order specified
  by the plugin position at the plugin chain.

All of the hooks will offer plugin order guarantee, as they all modify/update some item so the order
of plugins is important. Execution hooks need to guarantee step order, as the items that are being modified
in each step (config, resource, and filesystem) are also needed in the following steps. This is not true for
initialization hooks that modify items (metadata and flagset) that are only used in their own methods,
so they only need to guarantee hook order.

Execution hooks will be able to return an error. A specific error can be returned to specify that
no further hooks of this plugin should be called, but that the scaffold process should be continued.
This enables plugins to exit early, e.g., a plugin that scaffolds some files only for cluster-scoped
resources can detect if the resource is cluster-scoped at one of the first execution steps, and
therefore, use this error to tell the CLI that no further execution step should be called for itself.

### Initialization hooks

#### Update metadata
This hook will be used for two purposes. It provides CLI-related metadata to the Subcommand (e.g., 
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
This hook will allow subcommands to define specific flags.

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
This hook will be used to inject the `Config` object that the plugin can modify at will.
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
This hook will be used to inject the `Resource` object created by the CLI.

- Required/optional
  - [x] Required
  - [ ] Optional
- Subcommands
  - [ ] Init
  - [ ] Edit
  - [x] Create API
  - [x] Create webhook

#### Pre-scaffold
This hook will be used to take actions before the main scaffolding is performed, e.g. validations.

NOTE: a filesystem abstraction will be passed to this hook, but it should not be used for scaffolding.

- Required/optional
  - [ ] Required
  - [x] Optional
- Subcommands
  - [x] Init
  - [x] Edit
  - [x] Create API
  - [x] Create webhook

#### Scaffold
This hook will be used to perform the main scaffolding.

NOTE: a filesystem abstraction will be passed to this hook that must be used for scaffolding.

- Required/optional
  - [x] Required
  - [ ] Optional
- Subcommands
  - [x] Init
  - [x] Edit
  - [x] Create API
  - [x] Create webhook

#### Post-scaffold
This hook will be used to take actions after the main scaffolding is performed, e.g. cleanup.

NOTE: a filesystem abstraction will **NOT** be passed to this hook, as post-scaffold task do not require it.
In case some post-scaffold task requires a filesystem abstraction, it could be added.

NOTE 2: the project configuration is saved by the CLI before calling this hook, so changes done to the
configuration at this hook will not be persisted.

- Required/optional
  - [ ] Required
  - [x] Optional
- Subcommands
  - [x] Init
  - [x] Edit
  - [x] Create API
  - [x] Create webhook
  
### Override plugins for single subcommand calls

Defining plugins at initialization and using them for every command call will solve most of the cases.
However, there are some cases where a plugin may be wanted just for a certain subcommand call. For
example, a project with multiple controllers may want to follow the declarative pattern in only one of
their controllers. The other case is also relevant, a project where most of the controllers follow the
declarative pattern may need a single controller not to follow it.

In order to achieve this, the `--plugins` flag will be allowed in every command call, overriding the
value used in its corresponging project initialization call.

### Plugin chain persistence

Currently, the project configuration v3 offers two mechanisms for storing plugin-related information.

- A layout field (`string`) that is used for plugin resolution on initialized projects.
- A plugin field (`map[string]interface{}`) that is used for plugin configuration raw storage.

Plugin resolution uses the `layout` field to resolve plugins. In this phase, it has to store a plugin
chain and not a single plugin. As this value is stored as a string, comma-separated representation can
be used to represent a chain of plugins instead.

NOTE: commas are not allowed in the plugin key.

While the `plugin` field may seem like a better fit to store the plugin chain, as it can already
contain multiple values, there are several issues with this alternative approach:
- A map does not provide any order guarantee, and the plugin chain order is relevant.
- Some plugins do not store plugin-specific configuration information, e.g. the `go`-plugins. So
  the absence of a plugin key doesn't mean that the plugin is not part of the plugin chain.
- The desire of running a different set of plugins for a single subcommand call has already been
  mentioned. Some of these out-of-chain plugins may need to store plugin-specific configuration,
  so the presence of a plugin doesn't mean that is part of the plugin chain.

The next project configuration version could consider this new requirements to define the
names/types of these two fields.

### Plugin bundle

As a side-effect of plugin chaining, the user experience may suffer if they need to provide
several plugin keys for the `--plugins` flag. Additionally, this would also mean a user-facing
important breaking change.

In order to solve this issue, a plugin bundle concept will be introduced. A plugin bundle
behaves as a plugin:
- It has a name: provided at creation.
- It has a version: provided at creation.
- It has a list of supported project versions: computed from the common supported project
  versions of all the plugins in the bundled.

Instead of implementing the optional getter methods that return a subcommand, it offers a way
to retrieve the list of bundled plugins. This process will be done after plugin resolution.

This way, CLIs will be able to define bundles, which will be used in the user-facing API and
the plugin resolution process, but later they will be treated as separate plugins offering
the maintainability and separation of concerns advantages that smaller plugins have in
comparison with bigger monolithic plugins.

## Implementation

The following types are used as input/output values of the described hooks:
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

The described hooks are implemented through the use of the following interfaces.
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
	PreScaffold(machinery.Filesystem) error
}

type Scaffolder interface {
	Scaffold(machinery.Filesystem) error
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

An additional interface defines the bundle method to return the wrapped plugins:
```go
type Bundle interface {
	Plugin
	Plugins() []Plugin
}
```
