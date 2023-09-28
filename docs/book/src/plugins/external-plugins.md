# Creating External Plugins

## Overview

Kubebuilder's functionality can be extended through the use of external plugins.

An external plugin is an executable (can be written in any language) that implements an execution pattern that Kubebuilder knows how to interact with.

The Kubebuilder CLI loads the external plugin in the specified path and interacts with it through `stdin` & `stdout`.

## When is it useful?

- If you want to create helpers or addons on top of the scaffolds done by Kubebuilder's default scaffolding.

- If you design customized layouts and want to take advantage of functions from Kubebuilder library.

- If you are looking for implementing plugins in a language other than `Go`.

## How to write it?

The inter-process communication between your external plugin and Kubebuilder is through the standard I/O.

Your external plugin can be written in any language, given it adheres to the [PluginRequest][PluginRequest] and [PluginResponse][PluginResponse] type structures.

`PluginRequest` encompasses all the data Kubebuilder collects from the CLI and previously executed plugins in the plugin chain.
Kubebuilder conveys the marshaled PluginRequest (a `JSON` object) to the external plugin over `stdin`.

Below is a sample JSON object of the `PluginRequest` type, triggered by `kubebuilder init --plugins sampleexternalplugin/v1 --domain my.domain`:
```json
{
    "apiVersion": "v1alpha1",
    "args": ["--domain", "my.domain"],
    "command": "init",
    "universe": {}
}
```

`PluginResponse` represents the updated state of the project, as modified by the plugin. This data structure is serialized into `JSON` and sent back to Kubebuilder via `stdout`.

Here is a sample JSON representation of the `PluginResponse` type:
```json
{
    "apiVersion": "v1alpha1",
    "command": "init",
    "metadata": {
        "description": "The `init` subcommand is meant to initialize a project via Kubebuilder. It scaffolds a single file: `initFile`",
        "examples": "kubebuilder init --plugins sampleexternalplugin/v1 --domain my.domain"
    },
    "universe": {
        "initFile": "A simple file created with the `init` subcommand"
    },
    "error": false,
    "errorMsgs": []
}
```

In this example, the `init` command of the plugin has created a new file called `initFile`. 

The content of this file is: `A simple file created with the init subcommand`, which is recorded in the `universe` field of the response.

This output is then sent back to Kubebuilder, allowing it to incorporate the changes made by the plugin into the project.

<aside class="note">
<h1>Caution</h1>

When writing your own external plugin, you **should not** directly echo or print anything to the stdout. 

This is because Kubebuilder and your plugin are communicating with each other via `stdin` and `stdout` using structured `JSON` data.
Any additional information sent to stdout (such as debug messages or logs) that's not part of the expected PluginResponse JSON structure may cause parsing errors when Kubebuilder tries to read and decode the response from your plugin.

If you need to include logs or debug messages while developing your plugin, consider writing these messages to a log file instead.

</aside>

## How to use it?

### Prerequisites
- kubebuilder CLI > 3.11.0
- An executable for the external plugin.

  This could be a plugin that you've created yourself, or one from an external source.
- Configuration of the external plugin's path.

  This can be done by setting the `${EXTERNAL_PLUGINS_PATH}` environment variable, or by placing the plugin in a path that follows a `group-like name and version` scheme:
```sh
# for Linux
$HOME/.config/kubebuilder/plugins/${name}/${version}/${name}

# for OSX
~/Library/Application Support/kubebuilder/plugins/${name}/${version}/${name}
```
As an example, if you're on Linux and you want to use `v2` of an external plugin called `foo.acme.io`, you'd place the executable in the folder `$HOME/.config/kubebuilder/plugins/foo.acme.io/v2/` with a file name that also matches the plugin name up to an (optional) file extension.
In other words, passing the flag `--plugins=foo.acme.io/v2` to `kubebuilder` would find the plugin at either of these locations
* `$HOME/.config/kubebuilder/plugins/foo.acme.io/v2/foo.acme.io`
* `$HOME/.config/kubebuilder/plugins/foo.acme.io/v2/foo.acme.io.sh`
* `$HOME/.config/kubebuilder/plugins/foo.acme.io/v2/foo.acme.io.py`
* etc...

### Subcommands:

The external plugin supports the same subcommands as kubebuilder already provides:
- `init`: project initialization.
- `create api`: scaffold Kubernetes API definitions.
- `create webhook`: scaffold Kubernetes webhooks.
- `edit`: update the project configuration.

Also, there are **Optional** subcommands for a better user experience:
- `metadata`: add customized plugin description and examples when a `--help` flag is specified. 
- `flags`: specify valid flags for Kubebuilder to pass to the external plugin.

<aside class="note">
<h1>More about `flags` subcommand</h1>

The `flags` subcommand in an external plugin allows for early error detection by informing Kubebuilder about the flags the plugin supports. If an unsupported flag is identified, Kubebuilder can issue an error before the plugin is called to execute. 
If a plugin does not implement the `flags` subcommand, Kubebuilder will pass all flags to the plugin, making it the external plugin's responsibility to handle any invalid flags. 

</aside>

### Configuration

You can configure your plugin path with a ENV VAR `$EXTERNAL_PLUGINS_PATH` to tell Kubebuilder where to search for the plugin binary, such as:
```sh
export EXTERNAL_PLUGINS_PATH = <custom-path>
```

Otherwise, Kubebuilder would search for the plugins in a default path based on your OS.

Now, you can using it by calling the CLI commands:
```sh
# Initialize a new project with the external plugin named `sampleplugin`
kubebuilder init --plugins sampleplugin/v1

# Display help information of the `init` subcommand of the external plugin
kubebuilder init --plugins sampleplugin/v1 --help

# Create a new API with the above external plugin with a customized flag `number`
kubebuilder create api --plugins sampleplugin/v1 --number 2

# Create a webhook with the above external plugin with a customized flag `hooked`
kubebuilder create webhook --plugins sampleplugin/v1 --hooked

# Update the project configuration with the above external plugin
kubebuilder edit --plugins sampleplugin/v1

# Create new APIs with external plugins v1 and v2 by respecting the plugin chaining order
kubebuilder create api --plugins sampleplugin/v1,sampleplugin/v2

# Create new APIs with the go/v4 plugin and then pass those files to the external plugin by respecting the plugin chaining order
kubebuilder create api --plugins go/v4,sampleplugin/v1
```


## Further resources

- Check the [design proposal of the external plugin](https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-2.md) 
- Check the [plugin implementation](https://github.com/kubernetes-sigs/kubebuilder/pull/2338)
- A [sample external plugin written in Go](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/simple-external-plugin-tutorial/testdata/sampleexternalplugin/v1)
- A [sample external plugin written in Python](https://github.com/rashmigottipati/POC-Phase2-Plugins)
- A [sample external plugin written in JavaScript](https://github.com/Eileen-Yu/kb-js-plugin)

[PluginRequest]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugin/external/types.go#L23
[PluginResponse]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugin/external/types.go#L42
