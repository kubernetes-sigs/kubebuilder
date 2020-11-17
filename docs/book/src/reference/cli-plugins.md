# CLI Plugins

Kubebuilder CLI plugins wrap scaffolding and CLI features in conveniently packaged Go types that are executed by the
`kubebuilder` binary, or any binary that imports them. More specifically, a plugin configures the execution of one
of the following CLI commands:
* `init`: project initialization.
* `create api`: scaffold Kubernetes API definitions.
* `create webhook`: scaffold Kubernetes webhooks.

Plugins are identified by a key of the form `<name>/<version>`. There are two ways to specify a plugin to run:
* Setting `kubebuilder init --plugins=<plugin key>`, which will initialize a project configured for plugin with key
 `<plugin key>`.
* A `layout: <plugin key>` in the scaffolded `PROJECT` configuration file. Commands (except for `init`, which scaffolds
  this file) will look at this value before running to choose which plugin to run.

By default, `<plugin key>` will be `go.kubebuilder.io/vX`, where `X` is some integer.

## Plugin interfaces

Each plugin is required to implement the [`Base`][plugin-base] interface.

```go
type Base interface {
  // Version returns the plugin's version, which contains a positive integer
  // and an optional "stage" string. The string representation of this version
  // has format:
  // (v)?[1-9][0-9]*(-(alpha|beta))?
  Version() Version
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

### Plugin types

On top of being a `Base`, a plugin should also implement the [`GenericSubcommand`][plugin-subc] interface so it can be
run with a CLI:

```go
type GenericSubcommand interface {
  // UpdateContext updates a PluginContext with command-specific help text,
  // like description and examples. Can be a no-op if default help text is desired.
  UpdateContext(*PluginContext)
  // BindFlags binds the plugin's flags to the CLI. This allows each plugin to
  // define its own command line flags for the kubebuilder subcommand.
  BindFlags(*pflag.FlagSet)
  // InjectConfig passes a config to a plugin. The plugin may modify the
  // config. Initializing, loading, and saving the config is managed by the
  // cli package.
  InjectConfig(*config.Config)
  // Run runs the subcommand.
  Run() error
}
```

The [plugin context][plugin-context] is optionally updated by `UpdateContext(ctx)` to set custom help text for the target
command; this method can be a no-op, which will preserve the default help text set by the [cobra][cobra] command constructors.

A plugin also implements one of the following interface pairs to declare its support for specific subcommands:

```go
// To implement the 'init' subcommand.
type InitPluginGetter interface {
  Base
  GetInitPlugin() Init
}

type Init interface {
  GenericSubcommand
}

// To implement the 'create api' subcommand.
type CreateAPIPluginGetter interface {
  Base
  GetCreateAPIPlugin() CreateAPI
}

type CreateAPI interface {
  GenericSubcommand
}

// To implement the 'create webhook' subcommand.
type CreateWebhookPluginGetter interface {
  Base
  GetCreateWebhookPlugin() CreateWebhook
}

type CreateWebhook interface {
  GenericSubcommand
}
```

The pair system allows a plugin implementation to have the same `Base` "parent" plugin with child typed plugins. For
example, the following plugin `go.example.com/v1` implements each of the above pairs:

```go
import (
  "github.com/spf13/pflag"
  "sigs.k8s.io/kubebuilder/pkg/model/config"
  "sigs.k8s.io/kubebuilder/pkg/plugin"
)

// Plugin embeds internal types that implement one plugin type each,
// while implementing `Base` itself.
type Plugin struct {
  initPlugin
  createAPIPlugin
  createWebhookPlugin
}

// Base implementation.
func (Plugin) Name() string                                   { return "go.example.com" }
func (Plugin) Version() plugin.Version                        { return plugin.Version{Number: 1} }
func (Plugin) SupportedProjectVersions() []string             { return []string{"3"} }
// Getters.
func (p Plugin) GetInitPlugin() plugin.Init                   { return &p.initPlugin }
func (p Plugin) GetCreateAPIPlugin() plugin.CreateAPI         { return &p.createAPIPlugin }
func (p Plugin) GetCreateWebhookPlugin() plugin.CreateWebhook { return &p.createWebhookPlugin }

// Example of one of the internal implementations of GenericSubcommand.
type createAPIPlugin struct { ... }
func (p createAPIPlugin) UpdateContext(ctx *plugin.Context) { ... }
func (p *createAPIPlugin) BindFlags(fs *pflag.FlagSet)      { ... }
func (p *createAPIPlugin) InjectConfig(c *config.Config)    { ... }
func (p *createAPIPlugin) Run() error                       { ... }
```

For a full implementation example, check out Kubebuilder's native [`go.kubebuilder.io`][kb-go-plugin] plugin.

#### Deprecated Plugins

Once a plugin is deprecated, have it implement a `Deprecated` interface so a deprecation warning will be printed when it is used:

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

## CLI system

Plugins are run using a [`CLI`][cli] object, which maps a plugin type to a subcommand and calls that plugin's methods.
For example, writing a program that injects an `Init` plugin into a `CLI` then calling `CLI.Run()` will call the
plugin's `UpdateContext`, `BindFlags`, `InjectConfig`, and `Run` methods with information a user has passed to the
program in `kubebuilder init`. Hopefully the following code example will clarify this rather confusing description:

```go
// Several plugins for different languages with different versions.
import (
  ansiblev1 "github.com/example/my-plugins/pkg/plugins/ansible/v1"
  golangv1 "github.com/example/my-plugins/pkg/plugins/golang/v1" // From the above example.
  golangv2 "github.com/example/my-plugins/pkg/plugins/golang/v2"
  helmv1 "github.com/example/my-plugins/pkg/plugins/helm/v1"
)

// Create a CLI with name 'controller-builder' that supports
// project version "3" and Go, Helm, and Ansible plugins
// of various versions. This CLI defaults to running the latest
// Go plugin version unless '--plugins' or 'layout' specify
// one of the other plugins.
func main() {
  rootCmdCfg := cli.RootCommandConfig{
        CommandName: "controller-builder",
	}
  c, err := cli.New(
    cli.WithRootCommandConfig(rootCmdCfg),
    cli.WithDefaultProjectVersion("3"),
    cli.WithExtraCommands(newCustomCobraCmd()),
    cli.WithPlugins(
      &golangv1.Plugin{},  // "go.example.com/v1"
      &golangv2.Plugin{},  // "go.example.com/v2-alpha"
      &helmv1.Plugin{},    // "helm.example.com/v1"
      &ansiblev1.Plugin{}, // "ansible.example.com/v1"
    ),
    cli.WithDefaultPlugins(
      &golangv1.Plugin{},  // "go.example.com/v1", default
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

This program can then be built and run in the following ways:

Default behavior:

```sh
# Initialize a project with the default Init plugin, "go.example.com/v1".
# This key is automatically written to a PROJECT config file.
$ controller-builder init
# Create an API and webhook with "go.example.com/v1" CreateAPI and
# CreateWebhook plugin methods. This key was read from the config file.
$ controller-builder create api [flags]
$ controller-builder create webhook [flags]
```

Selecting a plugin using `--plugins`:

```sh
# Initialize a project with the "ansible.example.com/v1" Init plugin.
# Like above, this key is written to a config file.
$ controller-builder init --plugins ansible
# Create an API and webhook with "ansible.example.com/v1" CreateAPI
# and CreateWebhook plugin methods. This key was read from the config file.
$ controller-builder create api [flags]
$ controller-builder create webhook [flags]
```

## Plugin naming

Plugin names must be DNS1123 labels and should be fully qualified, i.e. they have a suffix like
`.example.com`. For example, the base Go scaffold used with `kubebuilder` commands has name `go.kubebuilder.io`.
Qualified names prevent conflicts between plugin names; both `go.kubebuilder.io` and `go.example.com` can both scaffold
Go code and can be specified by a user.

## Plugin versioning

A plugin's `Version()` method returns a [`plugin.Version`][plugin-version-type] object containing an integer value
and optionally a stage string of either "alpha" or "beta". The integer denotes the current version of a plugin.
Two different integer values between versions of plugins indicate that the two plugins are incompatible. The stage
string denotes plugin stability:
* `alpha` should be used for plugins that are frequently changed and may break between uses.
* `beta` should be used for plugins that are only changed in minor ways, ex. bug fixes.

### Breaking changes

Any change that will break a project scaffolded by the previous plugin version is a breaking change.


[plugin-base]:https://pkg.go.dev/sigs.k8s.io/kubebuilder/pkg/plugin#Base
[plugin-subc]:https://pkg.go.dev/sigs.k8s.io/kubebuilder/pkg/plugin#GenericSubcommand
[plugin-context]:https://pkg.go.dev/sigs.k8s.io/kubebuilder/pkg/plugin#Context
[cobra]:https://pkg.go.dev/github.com/spf13/cobra
[kb-go-plugin]:https://pkg.go.dev/sigs.k8s.io/kubebuilder/pkg/plugin/v2#Plugin
[cli]:https://pkg.go.dev/sigs.k8s.io/kubebuilder/pkg/cli#CLI
[plugin-version-type]:https://pkg.go.dev/sigs.k8s.io/kubebuilder/pkg/plugin#Version
