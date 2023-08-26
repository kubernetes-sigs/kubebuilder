# Extending the CLI and Scaffolds

## Overview

You can extend Kubebuilder to allow your project to have the same CLI features and provide the plugins scaffolds.

## CLI system

Plugins are run using a [`CLI`][cli] object, which maps a plugin type to a subcommand and calls that plugin's methods.
For example, writing a program that injects an `Init` plugin into a `CLI` then calling `CLI.Run()` will call the
plugin's [SubcommandMetadata][plugin-sub-command], [UpdatesMetadata][plugin-update-meta] and `Run` methods with information a user has passed to the
program in `kubebuilder init`. Following an example:

```go
package cli

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/cli"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	kustomizecommonv1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"
	declarativev1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/declarative/v1"
	golangv3 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3"

)

var (
	// The following is an example of the commands
	// that you might have in your own binary
	commands = []*cobra.Command{
		myExampleCommand.NewCmd(),
	}
	alphaCommands = []*cobra.Command{
		myExampleAlphaCommand.NewCmd(),
	}
)

// GetPluginsCLI returns the plugins based CLI configured to be used in your CLI binary
func GetPluginsCLI() (*cli.CLI) {
	// Bundle plugin which built the golang projects scaffold by Kubebuilder go/v3
	gov3Bundle, _ := plugin.NewBundleWithOptions(plugin.WithName(golang.DefaultNameQualifier), 
		plugin.WithVersion(plugin.Version{Number: 3}),
		plugin.WithPlugins(kustomizecommonv1.Plugin{}, golangv3.Plugin{}),
	)


	c, err := cli.New(
		// Add the name of your CLI binary
		cli.WithCommandName("example-cli"),
		
		// Add the version of your CLI binary
		cli.WithVersion(versionString()),
		
		// Register the plugins options which can be used to do the scaffolds via your CLI tool. See that we are using as example here the plugins which are implemented and provided by Kubebuilder
		cli.WithPlugins(
			gov3Bundle,
			&declarativev1.Plugin{},
		),
		
		// Defines what will be the default plugin used by your binary. It means that will be the plugin used if no info be provided such as when the user runs `kubebuilder init`
		cli.WithDefaultPlugins(cfgv3.Version, gov3Bundle),
		
		// Define the default project configuration version which will be used by the CLI when none is informed by --project-version flag.
		cli.WithDefaultProjectVersion(cfgv3.Version),
		
		// Adds your own commands to the CLI
		cli.WithExtraCommands(commands...),
		
		// Add your own alpha commands to the CLI
		cli.WithExtraAlphaCommands(alphaCommands...),
		
		// Adds the completion option for your CLI
		cli.WithCompletion(),
	)
	if err != nil {
		log.Fatal(err)
	}

	return c
}

// versionString returns the CLI version
func versionString() string {
	// return your binary project version
}
```

This program can then be built and run in the following ways:

Default behavior:

```sh
# Initialize a project with the default Init plugin, "go.example.com/v1".
# This key is automatically written to a PROJECT config file.
$ my-bin-builder init
# Create an API and webhook with "go.example.com/v1" CreateAPI and
# CreateWebhook plugin methods. This key was read from the config file.
$ my-bin-builder create api [flags]
$ my-bin-builder create webhook [flags]
```

Selecting a plugin using `--plugins`:

```sh
# Initialize a project with the "ansible.example.com/v1" Init plugin.
# Like above, this key is written to a config file.
$ my-bin-builder init --plugins ansible
# Create an API and webhook with "ansible.example.com/v1" CreateAPI
# and CreateWebhook plugin methods. This key was read from the config file.
$ my-bin-builder create api [flags]
$ my-bin-builder create webhook [flags]
```

### CLI manages the PROJECT file

The CLI is responsible for managing the [PROJECT file config][project-file-config], representing the configuration of the projects that are scaffold by the CLI tool.

## Plugins

Kubebuilder provides scaffolding options via plugins. Plugins are responsible for implementing the code that will be executed when the sub-commands are called. You can create a new plugin by implementing the [Plugin interface][plugin-interface]. 

On top of being a `Base`, a plugin should also implement the [`SubcommandMetadata`][plugin-subc] interface so it can be run with a CLI. It optionally to set custom help text for the target  command; this method can be a no-op, which will preserve the default help text set by the [cobra][cobra] command constructors.

Kubebuilder CLI plugins wrap scaffolding and CLI features in conveniently packaged Go types that are executed by the
`kubebuilder` binary, or any binary which imports them. More specifically, a plugin configures the execution of one
of the following CLI commands:

- `init`: project initialization.
- `create api`: scaffold Kubernetes API definitions.
- `create webhook`: scaffold Kubernetes webhooks.

Plugins are identified by a key of the form `<name>/<version>`. There are two ways to specify a plugin to run:

- Setting `kubebuilder init --plugins=<plugin key>`, which will initialize a project configured for plugin with key
 `<plugin key>`.
 
- A `layout: <plugin key>` in the scaffolded [PROJECT configuration file][project-file]. Commands (except for `init`, which scaffolds this file) will look at this value before running to choose which plugin to run. 

By default, `<plugin key>` will be `go.kubebuilder.io/vX`, where `X` is some integer.

For a full implementation example, check out Kubebuilder's native [`go.kubebuilder.io`][kb-go-plugin] plugin.

### Plugin naming

Plugin names must be DNS1123 labels and should be fully qualified, i.e. they have a suffix like
`.example.com`. For example, the base Go scaffold used with `kubebuilder` commands has name `go.kubebuilder.io`.
Qualified names prevent conflicts between plugin names; both `go.kubebuilder.io` and `go.example.com` can both scaffold
Go code and can be specified by a user.

### Plugin versioning

A plugin's `Version()` method returns a [`plugin.Version`][plugin-version-type] object containing an integer value
and optionally a stage string of either "alpha" or "beta". The integer denotes the current version of a plugin.
Two different integer values between versions of plugins indicate that the two plugins are incompatible. The stage
string denotes plugin stability:

- `alpha`: should be used for plugins that are frequently changed and may break between uses.
- `beta`: should be used for plugins that are only changed in minor ways, ex. bug fixes.

### Breaking changes

Any change that will break a project scaffolded by the previous plugin version is a breaking change.

### Plugins Deprecation 

Once a plugin is deprecated, have it implement a [Deprecated][deprecate-plugin-doc] interface so a deprecation warning will be printed when it is used.

## Bundle Plugins

[Bundle Plugins][bundle-plugin-doc] allow you to create a plugin that is a composition of many plugins:

```go
   // see that will be like myplugin.example/v1`  
  myPluginBundle, _ := plugin.NewBundle(plugin.WithName(`<plugin-name>`),
  		plugin.WithVersion(`<plugin-version>`),
		plugin.WithPlugins(pluginA.Plugin{}, pluginB.Plugin{}, pluginC.Plugin{}),
	)

```

Note that it means that when a user of your CLI calls this plugin, the execution of the sub-commands will be sorted by the order to which they were added in a chain:


> `sub-command` of plugin A ➔ `sub-command` of plugin B ➔ `sub-command` of plugin C

Then, to initialize using this "Plugin Bundle" which will run the chain of plugins:

```
kubebuider init --plugins=myplugin.example/v1 
```   

- Runs init `sub-command` of the plugin A
- And then, runs init `sub-command` of the plugin B
- And then, runs init `sub-command` of the plugin C 

[project-file-config]: ../reference/project-config.md
[plugin-interface]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/plugin#Plugin
[go-dev-doc]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3
[plugin-sub-command]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/plugin#Subcommand
[project-file]: ../reference/project-config.md
[plugin-subc]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/plugin#Subcommand
[cobra]:https://pkg.go.dev/github.com/spf13/cobra
[kb-go-plugin]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3
[bundle-plugin-doc]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/plugin#Bundle
[deprecate-plugin-doc]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/plugin#Deprecated
[plugin-update-meta]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/plugin#UpdatesMetadata
[cli]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/cli
[plugin-version-type]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/plugin#Version
