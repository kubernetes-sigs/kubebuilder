Dynamic Bundle
===================

## Overview
Kubebuilder currently has an implementation of a `Bundle` which allows multiple plugins to be *bundled* together. The execution of the plugins within a `Bundle` follows a sequential order based on their definition. 

An example of the current method:

```go
gov3Bundle, _ := plugin.NewBundle(golang.DefaultNameQualifier, plugin.Version{Number: 3},
	kustomizecommonv1.Plugin{},
	golangv3.Plugin{},
)
```

In the above example a plugin `Bundle` is created that ensures that the `kustomize/v1` plugin is run before the `go/v3` plugin.
 
With the addition of Phase 2 Plugins on the way, users using the `--plugins` flag to utilize external plugins will not get the benefits of using some default functionality that previous Kubebuilder users using the default `go/v3` plugin gained via the `kustomize/v1` plugin.

## Proposal
The proposed solution is to create a new form of `Bundle` that is more dynamic and can be used to create a default bundle that contains sensible default plugins that can be used with all plugins. This could be overriden by a command line flag.

## User Stories
- As a Kubebuilder maintainer I would like to have Kubebuilder implement sane default plugins that always run before and/or after plugins, specifying this once and it applying to all uses

- As a Kubebuilder user I expect Kubebuilder to have default functionality to make my operator development easier

- As a Kubebuilder user I would like to be able to override the default Kubebuilder plugin chain, scaffolding only exactly what I feel I need

## Goals
- `kubebuilder` is able to define sane defaults for plugins that should always be run before and/or after user specified plugins

- A `Bundle` no longer has to be created for each plugin to implement sane defaults

## Examples
Context: Kubebuilder defines that the `kustomize/v1` plugin should always run before any other plugin. Kubebuilder also defines the default plugin to be run if a user does not specify plugins via the `--plugins` flag is `go/v3`

- `kubebuilder init`
	- Should scaffold using the `kustomize/v1` plugin and the `go/v3` plugin
- `kubebuilder init --plugins other.plugin/v3`
	- Should scaffold using the `kustomize/v1` plugin and the `other.plugin/v3` plugin
- `kubebuilder init --plugins other.plugin/v3 --override-default-plugin-chain`
	- Should scaffold using only the `other.plugin/v3` plugin

## Implementation Details
### Interface
A new interface called `DynamicBundle` would be created that implements the pre-existing `Bundle` interface, but adds two functions.
1. `BeforePlugins` to retrieve a list of plugins that should be run during before any other plugins
2. `AfterPlugins` to retrieve a list of plugins that should be run after any other plugins

Proposed implementation:

File pkg/plugin/plugin.go:
```go
type DynamicBundle interface {
	Bundle

	// BeforePlugins returns the list of plugins that should be run before all plugins
	BeforePlugins() []Plugin

	// AfterPlugins returns the list of plugins that should be run after all plugins
	AfterPlugins() []Plugin
}
```

### Struct
A new struct would be created that implements the `DynamicBundle` interface.

Proposed implementation:

File pkg/plugin/bundle.go:
```go
type dynamicBundle struct {
	bundle

	beforePlugins []Plugin
	afterPlugins  []Plugin
}

// Name implements Plugin
func (db dynamicBundle) Name() string {
	return db.bundle.name
}

// Version implements Plugin
func (db dynamicBundle) Version() Version {
	return db.bundle.version
}

// SupportedProjectVersions implements Plugin
func (db dynamicBundle) SupportedProjectVersions() []config.Version {
	return db.bundle.supportedProjectVersions
}

// Plugins implements Bundle
func (db dynamicBundle) Plugins() []Plugin {
	return append(db.beforePlugins, append(db.bundle.plugins, db.afterPlugins...)...)
}

// BeforePlugins returns a list of plugins that should be run before all other plugins
func (db dynamicBundle) BeforePlugins() []Plugin {
	return db.beforePlugins
}

// AfterPlugins returns a list of plugins that should be run after all other plugins
func (db dynamicBundle) AfterPlugins() []Plugin {
	return db.afterPlugins
}

// NewDynamicBundle returns a new DynamicBundle with the specified name, version, beforePlugins, injectedPlugins, and afterPlugins.
// An error is returned if the plugins do not all support the same project versions
func NewDynamicBundle(name string, version Version, beforePlugins []Plugin, injectedPlugins []Plugin, afterPlugins []Plugin) (DynamicBundle, error) {
	supportedProjectVersions := CommonSupportedProjectVersions(append(beforePlugins, (append(injectedPlugins, afterPlugins...))...)...)
	if len(supportedProjectVersions) == 0 {
		return nil, fmt.Errorf("in order to bundle plugins, they must all support at least one common project version")
	}

	return newDynamicBundle(name, version, supportedProjectVersions, beforePlugins, injectedPlugins, afterPlugins), nil
}

func newDynamicBundle(name string, version Version, spv []config.Version, bp []Plugin, ip []Plugin, ap []Plugin) dynamicBundle {
	var db dynamicBundle
	db.bundle.name = name
	db.bundle.version = version
	db.bundle.supportedProjectVersions = spv
	db.bundle.plugins = append(db.bundle.plugins, ip...)

	db.beforePlugins = append(db.beforePlugins, bp...)
	db.afterPlugins = append(db.afterPlugins, ap...)

	return db
}
```

### CLI
The logic that occurs in the CLI will have to be modified to accomodate the functionality that the `DynamicBundle` would provide.

The biggest modification would be rather than using the plugin keys that a user specifies via the `--plugins` flag to run plugins, it would be used to inject the plugins necessary into the `DynamicBundle` and overwritting the plugin keys to use the `DynamicBundle` as the plugin that runs.

This should be functionality that can be overridden by the user via a flag such as `--override-default-plugin-chain`

Proposed implementation:

File: pkg/cli/cli.go in function `getInfoFromFlags` before line 251:
```go
var override bool
fs.BoolVar(&override, "override-default-plugin-chain", false, "override the default plugin chain")
```

File: pkg/cli/cli.go in function `getInfoFromFlags` between lines 279 and 281:
```go
if len(c.pluginKeys) != 0 && !override {
	projectVersion := c.defaultProjectVersion
	if projectVersionStr != "" {
		projectVersion = c.projectVersion
	}

	pluginKey := c.defaultPlugins[projectVersion][0]
	defaultPlugin, ok := c.plugins[pluginKey]
	if !ok {
		fmt.Println("\nFAILED TO FIND DEFAULT PLUGIN FOR PROJECT VERSION", projectVersion)
	}

	dynamicBundle := defaultPlugin.(plugin.DynamicBundle)
	var pluginsToInject []plugin.Plugin

	var injectedPluginString string
	for _, pluginKey := range c.pluginKeys {
		pluginToInject, ok := c.plugins[pluginKey]
		if !ok {
			fmt.Println("\nFAILED TO FIND PLUGIN FOR --", pluginKey)
			continue
		}
		pluginsToInject = append(pluginsToInject, pluginToInject)
	}

	dynamicBundle, err := plugin.NewDynamicBundle(dynamicBundle.Name(), dynamicBundle.Version(), dynamicBundle.BeforePlugins(), pluginsToInject, dynamicBundle.AfterPlugins())

	if err != nil {
		return fmt.Errorf("\nDYNAMIC BUNDLE ERROR -- %w", err)
	}

	c.plugins[pluginKey] = dynamicBundle

	c.pluginKeys = []string{plugin.KeyFor(dynamicBundle)}
}
```

For each of the subcommands we would also need to add a new flag like:
```go
cmd.Flags().Bool("override-default-plugin-chain", false, "override the default plugin chain")
```

### Implementation in main
In the file cmd/main.go we would need to replace the current `Bundle` implementation with an implementation of the new `DynamicBundle`

Proposed implementation:

Defining the `DynamicBundle`:
```go
v3Dynamic, _ := plugin.NewDynamicBundle(
	"dynamic."+plugins.DefaultNameQualifier,
	plugin.Version{Number: 3},
	[]plugin.Plugin{
		kustomizecommonv1.Plugin{},
	},
	[]plugin.Plugin{
		golangv3.Plugin{},
	},
	nil)

v2Dynamic, _ := plugin.NewDynamicBundle(
	"dynamic."+plugins.DefaultNameQualifier,
	plugin.Version{Number: 2},
	nil,
	[]plugin.Plugin{
		golangv2.Plugin{},
	},
	nil)
```

Adding to the CLI plugins:
```go
c, err := cli.New(
	cli.WithCommandName("kubebuilder"),
	cli.WithVersion(versionString()),
	cli.WithPlugins(
		golangv2.Plugin{},
		golangv3.Plugin{},
		gov3Bundle,
		&kustomizecommonv1.Plugin{},
		&declarativev1.Plugin{},
		v2Dynamic,
		v3Dynamic,
		),
	cli.WithDefaultPlugins(cfgv2.Version, v2Dynamic),
	cli.WithDefaultPlugins(cfgv3.Version, v3Dynamic),
	cli.WithDefaultProjectVersion(cfgv3.Version),
	cli.WithCompletion(),
)
```

## Proof of Concept
A basic PoC has been created to show what this implementation could look like.

The PoC can be found here:
https://github.com/everettraven/kubebuilder/tree/poc/dynamic-bundle

It includes the creation of another plugin for testing purposes. The plugin key for the testing plugin is `base.default.test.plugin.one/v3`. All it does is scaffold a file called FAKE.md

The PoC only makes enough changes to get this logic working on the `kubebuilder init` subcommand and therefore the other subcommands will not function as expected.

The PoC also includes a lot of print statements so that you can see the changes occurring to the `DynamicBundle` when running `kubebuilder init`

### Examples

Using one plugin passed via `--plugins` flag:

![one-plugin](./assets/dynamic-bundle/one-plugin.gif)

Using two plugins passed via `--plugins` flag:

![two-plugin](./assets/dynamic-bundle/two-plugin.gif)

Using two plugins passed via `--plugins` flag and passing the `override-default-plugin-chain` flag:

![override-plugin](./assets/dynamic-bundle/override-plugin.gif)
