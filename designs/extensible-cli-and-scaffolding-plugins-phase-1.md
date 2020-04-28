# Extensible CLI and Scaffolding Plugins

## Overview

I would like for Kubebuilder to become more extensible, such that it could be imported and used as a library in other projects. Specifically, I'm looking for a way to use Kubebuilder's existing CLI and scaffolding for Go projects, but to also be able to augment the Kubebuilder project structure with other custom project types so that I can support the Kubebuilder workflow with non-Go operators (e.g. operator-sdk's Ansible and Helm-based operators).

The idea is for Kubebuilder to define one or more plugin interfaces that can be used to drive what the `init`, `create api` and `create webhooks` subcommands do and to add a new `cli` package that other projects can use to integrate out-of-tree plugins with the Kubebuilder CLI in their own projects.

## Related issues and PRs

* [#1148](https://github.com/kubernetes-sigs/kubebuilder/pull/1148)
* [#1171](https://github.com/kubernetes-sigs/kubebuilder/pull/1171)
* Possibly [#1218](https://github.com/kubernetes-sigs/kubebuilder/issues/1218)

## Prototype implementation

Barebones plugin refactor: https://github.com/joelanford/kubebuilder-exp
Kubebuilder feature branch: https://github.com/kubernetes-sigs/kubebuilder/tree/feature/plugins-part-2-electric-boogaloo

## Plugin interfaces

### Required

Each plugin would minimally be required to implement the `Plugin` interface.

```go
type Plugin interface {
    // Version returns the plugin's semantic version, ex. "v1.2.3".
    //
    // Note: this version is different from config version.
    Version() string
    // Name returns a DNS1123 label string defining the plugin type.
    // For example, Kubebuilder's main plugin would return "go".
    //
    // Plugin names can be fully-qualified, and non-fully-qualified names are
    // prepended to ".kubebuilder.io" to prevent conflicts.
    Name() string
    // SupportedProjectVersions lists all project configuration versions this
    // plugin supports, ex. []string{"2", "3"}. The returned slice cannot be empty.
    SupportedProjectVersions() []string
}
```

#### Plugin naming

Plugin names (returned by `Name()`) must be DNS1123 labels. The returned name
may be fully qualified (fq), ex. `go.kubebuilder.io`, or not but internally will
always be fq by either appending `.kubebuilder.io` to the name or using an
existing qualifier defined by the plugin. FQ names prevent conflicts between
plugin names; the plugin runner will ask the user to add a name qualifier to
a conflicting plugin.

### Optional

Next, a plugin could optionally implement further interfaces to declare its support for specific Kubebuilder subcommands. For example:
* `InitPlugin` - to initialize new projects
* `CreateAPIPlugin` - to create APIs (and possibly controllers) for existing projects
* `CreateWebhookPlugin` - to create webhooks for existing projects

Each of these interfaces would follow the same pattern (see the `InitPlugin` interface example below).

```go
type InitPluginGetter interface {
    Plugin
    // GetInitPlugin returns the underlying InitPlugin interface.
    GetInitPlugin() InitPlugin
}

type InitPlugin interface {
    GenericSubcommand
}
```

Each specialized plugin interface can leverage a generic subcommand interface, which prevents duplication of methods while permitting type checking and interface flexibility. A plugin context can be used to preserve default help text in case a plugin does not implement its own.

```go
type GenericSubcommand interface {
    // UpdateContext updates a PluginContext with command-specific help text, like description and examples.
    // Can be a no-op if default help text is desired.
    UpdateContext(*PluginContext)
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

type PluginContext struct {
    // Description is a description of what this subcommand does. It is used to display help.
    Description string
    // Examples are one or more examples of the command-line usage
    // of this plugin's project subcommand support. It is used to display help.
    Examples string
}
```

#### Deprecated Plugins

To generically support deprecated project versions, we could also add a `Deprecated` interface that the CLI could use to decide when to print deprecation warnings:

```go
// Deprecated is an interface that, if implemented, informs the CLI
// that the plugin is deprecated.  The CLI uses this to print deprecation
// warnings when the plugin is in use.
type Deprecated interface {
    // DeprecationWarning returns a deprecation message that callers
    // can use to warn users of deprecations
    DeprecationWarning() string
}
```

## Configuration

### Config version `3-alpha`

Any changes that break `PROJECT` file backwards-compatibility require a version
bump. This new version will be `3-alpha`, which will eventually be bumped to
`3` once the below config changes have stabilized.

### Project file plugin `layout`

The `PROJECT` file will specify what base plugin generated the project under
a `layout` key. `layout` will have the format: `Plugin.Name() + "/" + Plugin.Version()`.
`version` and `layout` have versions with different meanings: `version` is the
project config version, while `layout`'s version is the plugin semantic version.
The value in `version` will determine that in `layout` by a plugin's supported
project versions (via `SupportedProjectVersions()`).

Example `PROJECT` file:

```yaml
version: "3-alpha"
layout: go/v1.0.0
domain: testproject.org
repo: github.com/test-inc/testproject
resources:
- group: crew
  kind: Captain
  version: v1
```

## CLI

To make the above plugin system extensible and usable by other projects, we could add a new CLI package that Kubebuilder (and other projects) could use as their entrypoint.

Example Kubebuilder main.go:

```go
func main() {
	c, err := cli.New(
		cli.WithPlugins(
			&golangv1.Plugin{},
			&golangv2.Plugin{},
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}
```

Example Operator SDK main.go:

```go
func main() {
	c, err := cli.New(
		cli.WithCommandName("operator-sdk"),
		cli.WithDefaultProjectVersion("2"),
		cli.WithExtraCommands(newCustomCobraCmd()),
		cli.WithPlugins(
			&golangv1.Plugin{},
			&golangv2.Plugin{},
			&helmv1.Plugin{},
			&ansiblev1.Plugin{},
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## Comments & Questions

### Cobra Commands

**RESOLUTION:** `cobra` will be used directly in Phase 1 since it is a widely used, feature-rich CLI package. This, however unlikely, may change in future phases.

As discussed earlier as part of [#1148](https://github.com/kubernetes-sigs/kubebuilder/pull/1148), one goal is to eliminate the use of `cobra.Command` in the exported API of Kubebuilder since that is considered an internal implementation detail.

However, at some point, projects that make use of this extensibility will likely want to integrate their own subcommands. In this proposal, `cli.WithExtraCommands()` _DOES_ expose `cobra.Command` to allow callers to pass their own subcommands to the CLI.

In [#1148](https://github.com/kubernetes-sigs/kubebuilder/pull/1148), callers would use Kubebuilder's cobra commands to build their CLI. Here, control of the CLI is retained by Kubebuilder, and callers pass their subcommands to Kubebuilder. This has several benefits:
1. Kubebuilder's CLI subcommands are never exposed except via the explicit plugin interface. This allows the Kubebuilder project to re-implement its subcommand internals without worrying about backwards compatibility of consumers of Kubebuilder's CLI.
2. If desired, Kubebuilder could ensure that extra subcommands do not overwrite/reuse the existing Kubebuilder subcommand names. For example, only Kubebuilder gets to define the `init` subcommand
3. The overall binary's help handling is self-contained in Kubebuilder's CLI. Callers don't have to figure out how to have a cohesive help output between the Kubebuilder CLI and their own custom subcommands.

With all of that said, even this exposure of `cobra.Command` could be problematic. If Kubebuilder decides in the future to transition to a different CLI framework (or to roll its own) it has to either continue maintaining support for these extra cobra commands passed into it, or it was to break the CLI API.

Are there other ideas for how to handle the following requirements?
* Eliminate use of cobra in CLI interface
* Allow other projects to have custom subcommands
* Support cohesive help output

### Other
1. ~Should the `InitPlugin` interface methods be required of all plugins?~ No
2. ~Any other approaches or ideas?~
3. ~Anything I didn't cover that could use more explanation?~
