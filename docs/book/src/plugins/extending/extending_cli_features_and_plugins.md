# Extending CLI Features and Plugins

Kubebuilder provides an extensible architecture to scaffold
projects using plugins. These plugins allow you to customize the CLI
behavior or integrate new features.

In this guide, we’ll explore how to extend CLI features,
create custom plugins, and bundle multiple plugins.

## Creating Custom Plugins

To create a custom plugin, you need to implement
the [Kubebuilder Plugin interface][plugin-interface].

This interface allows your plugin to hook into Kubebuilder’s
commands (`init`, `create api`, `create webhook`, etc.)
and add custom logic.

### Example of a Custom Plugin

You can create a plugin that generates both
language-specific scaffolds and the necessary configuration files,
using the [Bundle Plugin](#bundle-plugin). This example shows how to
combine the Golang plugin with a Kustomize plugin:

```go
import (
    kustomizecommonv2 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2"
    golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
)

mylanguagev1Bundle, _ := plugin.NewBundle(
    plugin.WithName("mylanguage.kubebuilder.io"),
    plugin.WithVersion(plugin.Version{Number: 1}),
    plugin.WithPlugins(kustomizecommonv2.Plugin{}, mylanguagev1.Plugin{}),
)
```

This composition allows you to scaffold a common
configuration base (via Kustomize) and the
language-specific files (via `mylanguagev1`).

You can also use your plugin to scaffold specific
resources like CRDs and controllers, using
the `create api` and `create webhook` subcommands.

### Plugin Subcommands

Plugins are responsible for implementing the code that will be executed when the sub-commands are called.
You can create a new plugin by implementing the [Plugin interface][plugin-interface].

On top of being a `Base`, a plugin should also implement the [`SubcommandMetadata`][plugin-subc-metadata]
interface so it can be run with a CLI. Optionally, a custom help
text for the target  command can be set; this method can be a no-op, which will
preserve the default help text set by the [cobra][cobra] command
constructors.

Kubebuilder CLI plugins wrap scaffolding and CLI features in conveniently packaged Go types that are executed by the
`kubebuilder` binary, or any binary which imports them. More specifically, a plugin configures the execution of one
of the following CLI commands:

- `init`: Initializes the project structure.
- `create api`: Scaffolds a new API and controller.
- `create webhook`: Scaffolds a new webhook.
- `edit`: edit the project structure.

Here’s an example of using the `init` subcommand with a custom plugin:

```sh
kubebuilder init --plugins=mylanguage.kubebuilder.io/v1
```

This would initialize a project using the `mylanguage` plugin.

### Plugin Keys

Plugins are identified by a key of the form `<name>/<version>`.
There are two ways to specify a plugin to run:

- Setting `kubebuilder init --plugins=<plugin key>`, which will initialize a project configured for plugin with key
  `<plugin key>`.

- A `layout: <plugin key>` in the scaffolded [PROJECT configuration file][project-file-config]. Commands (except for `init`, which scaffolds this file) will look at this value before running to choose which plugin to run.

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

### Boilerplates

The Kubebuilder internal plugins use boilerplates to generate the
files of code. Kubebuilder uses templating to scaffold files for plugins.
For instance, when creating a new project, the `go/v4` plugin
scaffolds the `go.mod` file using a template defined in
its implementation.

You can extend this functionality in your custom
plugin by defining your own templates and using
[Kubebuilder’s machinery library][machinery] to generate files.
This library allows you to:

- Define file I/O behaviors.
- Add [markers][markers-scaffold] to the scaffolded files.
- Specify templates for your scaffolds.

#### Example: Boilerplate

For instance, the go/v4 scaffolds the `go.mod` file by defining an object that [implements the machinery interface][machinery].
The raw template is set to the `TemplateBody` field on the `Template.SetTemplateDefaults` method:

```go
{{#include ./../../../../../pkg/plugins/golang/v4/scaffolds/internal/templates/gomod.go}}
```

Such object that implements the machinery interface will later pass to the
execution of scaffold:

```go
// Scaffold implements cmdutil.Scaffolder
func (s *initScaffolder) Scaffold() error {
	log.Println("Writing scaffold for you to edit...")

	// Initialize the machinery.Scaffold that will write the boilerplate file to disk
	// The boilerplate file needs to be scaffolded as a separate step as it is going to
	// be used by the rest of the files, even those scaffolded in this command call.
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	...

	return scaffold.Execute(
		...
		&templates.GoMod{
			ControllerRuntimeVersion: ControllerRuntimeVersion,
		},
		...
	)
}
```

#### Example: Overwriting a File in a Plugin

Let's imagine that when a subcommand is called, you want
to overwrite an existing file.

For example, to modify the `Makefile` and add custom build steps,
in the definition of your Template you can use the following option:

```go
f.IfExistsAction = machinery.OverwriteFile
```

By using those options, your plugin can take control
of certain files generated by Kubebuilder’s default scaffolds.

## Customizing Existing Scaffolds

Kubebuilder provides utility functions to help you modify the default scaffolds. By using the [plugin utilities][plugin-utils], you can insert, replace, or append content to files generated by Kubebuilder, giving you full control over the scaffolding process.

These utilities allow you to:

- **Insert content**: Add content at a specific location within a file.
- **Replace content**: Search for and replace specific sections of a file.
- **Append content**: Add content to the end of a file without removing or altering the existing content.

### Example

If you need to insert custom content into a scaffolded file,
you can use the `InsertCode` function provided by the plugin utilities:

```go
pluginutil.InsertCode(filename, target, code)
```

This approach enables you to extend and modify the generated
scaffolds while building custom plugins.

For more details, refer to the [Kubebuilder plugin utilities][kb-utils].

## Bundle Plugin

Plugins can be bundled to compose more complex scaffolds.
A plugin bundle is a composition of multiple plugins that
are executed in a predefined order. For example:

```go
myPluginBundle, _ := plugin.NewBundle(
    plugin.WithName("myplugin.example.com"),
    plugin.WithVersion(plugin.Version{Number: 1}),
    plugin.WithPlugins(pluginA.Plugin{}, pluginB.Plugin{}, pluginC.Plugin{}),
)
```

This bundle will execute the `init` subcommand for each
plugin in the specified order:

1. `pluginA`
2. `pluginB`
3. `pluginC`

The following command will run the bundled plugins:

```sh
kubebuilder init --plugins=myplugin.example.com/v1
```

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

	"sigs.k8s.io/kubebuilder/v4/pkg/cli"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	kustomizecommonv2 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
	deployimage "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1"
    golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"

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
	// Bundle plugin which built the golang projects scaffold by Kubebuilder go/v4
	gov3Bundle, _ := plugin.NewBundleWithOptions(plugin.WithName(golang.DefaultNameQualifier),
		plugin.WithVersion(plugin.Version{Number: 3}),
		plugin.WithPlugins(kustomizecommonv2.Plugin{}, golangv4.Plugin{}),
	)


	c, err := cli.New(
		// Add the name of your CLI binary
		cli.WithCommandName("example-cli"),

		// Add the version of your CLI binary
		cli.WithVersion(versionString()),

		// Register the plugins options which can be used to do the scaffolds via your CLI tool. See that we are using as example here the plugins which are implemented and provided by Kubebuilder
		cli.WithPlugins(
			gov3Bundle,
			&deployimage.Plugin{},
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

### Inputs should be tracked in the PROJECT file

The CLI is responsible for managing the [PROJECT file configuration][project-file-config],
which represents the configuration of the projects scaffolded by the
CLI tool.

When extending Kubebuilder, it is recommended to ensure that your tool
or [External Plugin][external-plugin] properly uses the
[PROJECT file][project-file-config] to track relevant information.
This ensures that other external tools and plugins can properly
integrate with the project. It also allows tools features to help users
re-scaffold their projects such as the [Project Upgrade Assistant][upgrade-assistant]
provided by Kubebuilder, ensuring the tracked information in the
PROJECT file can be leveraged for various purposes.

For example, plugins can check whether they support the project setup
and re-execute commands based on the tracked inputs.

#### Example

By running the following command to use the
[Deploy Image][deploy-image] plugin to scaffold
an API and its controller:

```sh
kubebyilder create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:memcached:1.6.26-alpine3.19 --image-container-command="memcached,--memory-limit=64,-o,modern,-v" --image-container-port="11211" --run-as-user="1001" --plugins="deploy-image/v1-alpha" --make=false
```

The following entry would be added to the PROJECT file:

```yaml
...
plugins:
  deploy-image.go.kubebuilder.io/v1-alpha:
    resources:
    - domain: testproject.org
      group: example.com
      kind: Memcached
      options:
        containerCommand: memcached,--memory-limit=64,-o,modern,-v
        containerPort: "11211"
        image: memcached:memcached:1.6.26-alpine3.19
        runAsUser: "1001"
      version: v1alpha1
    - domain: testproject.org
      group: example.com
      kind: Busybox
      options:
        image: busybox:1.36.1
      version: v1alpha1
...
```

By inspecting the PROJECT file, it becomes possible to understand how
the plugin was used and what inputs were provided. This not only allows
re-execution of the command based on the tracked data but also enables
creating features or plugins that can rely on this information.

[sdk]: https://github.com/operator-framework/operator-sdk
[plugin-interface]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin
[machinery]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/machinery
[plugin-subc-metadata]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin#SubcommandMetadata
[plugin-version-type]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin#Version
[bundle-plugin-doc]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin#Bundle
[deprecate-plugin-doc]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin#Deprecated
[plugin-sub-command]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin#Subcommand
[plugin-update-meta]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin#UpdatesMetadata
[plugin-utils]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin/util
[markers-scaffold]: ./../../reference/markers/scaffold.md
[kb-utils]: https://github.com/kubernetes-sigs/kubebuilder/blob/book-v4/pkg/plugin/util/util.go
[project-file-config]: ./../../reference/project-config.md
[cli]: https://github.com/kubernetes-sigs/kubebuilder/tree/book-v4/pkg/cli
[kb-go-plugin]: https://github.com/kubernetes-sigs/kubebuilder/tree/book-v4/pkg/plugins/golang/v4
[cobra]: https://github.com/spf13/cobra
[external-plugin]: external-plugins.md
[deploy-image]: ./../available/deploy-image-plugin-v1-alpha.md
[upgrade-assistant]: ./../../reference/rescaffold.md
