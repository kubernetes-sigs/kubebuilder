# Creating External Plugins for Kubebuilder

## Overview

Kubebuilder's functionality can be extended through external plugins.
These plugins are executables (written in any language) that follow an
execution pattern recognized by Kubebuilder. Kubebuilder interacts with
these plugins via `stdin` and `stdout`, enabling seamless communication.

## Why Use External Plugins?

External plugins enable third-party solution maintainers to integrate their tools with Kubebuilder.
Much like Kubebuilder's own plugins, these can be opt-in, offering users
flexibility in tool selection. By developing plugins in their repositories,
maintainers ensure updates are aligned with their CI pipelines and can
manage any changes within their domain of responsibility.

If you are interested in this type of integration, collaborating with the
maintainers of the third-party solution is recommended. Kubebuilder's maintainers
are always willing to provide support in extending its capabilities.

## How to Write an External Plugin

Communication between Kubebuilder and an external plugin occurs via
standard I/O. Any language can be used to create the plugin, as long
as it follows the [PluginRequest][code-plugin-external] and [PluginResponse][code-plugin-external]
structures.

`PluginRequest` contains the data collected from the CLI and any previously executed plugins. Kubebuilder sends this data as a JSON object to the external plugin via `stdin`.

**Fields:**
- `apiVersion`: Version of the PluginRequest schema.
- `args`: Command-line arguments passed to the plugin.
- `command`: The subcommand being executed (e.g., `init`, `create api`, `create webhook`, `edit`).
- `universe`: Map of file paths to contents, updated across the plugin chain.
- `pluginChain` (optional): Array of plugin keys in the order they were executed. External plugins can inspect this to tailor behavior based on other plugins that ran (for example, `go.kubebuilder.io/v4` or `kustomize.common.kubebuilder.io/v2`).
- `config` (optional): Serialized PROJECT file configuration for the current project. Use it to inspect metadata, existing resources, or plugin-specific settings. Kubebuilder omits this field before the PROJECT file exists—typically during the first `init`—so plugins should check for its presence.


**Note:** Whenever Kubebuilder has a PROJECT file available (for example during `create api`, `create webhook`, `edit`, or a subsequent `init` run), `PluginRequest` includes the `config` field. During the very first `init` run the field is omitted because the PROJECT file does not exist yet.

**Example `PluginRequest` (triggered by `kubebuilder init --plugins go/v4,sampleexternalplugin/v1 --domain my.domain`):**

```json
{
  "apiVersion": "v1alpha1",
  "args": ["--domain", "my.domain"],
  "command": "init",
  "universe": {},
  "pluginChain": ["go.kubebuilder.io/v4", "kustomize.common.kubebuilder.io/v2", "sampleexternalplugin/v1"]
}
```

**Example `PluginRequest` for `create api` (includes `config`):**
```json
{
  "apiVersion": "v1alpha1",
  "args": ["--group", "crew", "--version", "v1", "--kind", "Captain"],
  "command": "create api",
  "universe": {},
  "pluginChain": ["go.kubebuilder.io/v4", "kustomize.common.kubebuilder.io/v2", "sampleexternalplugin/v1"],
  "config": {
    "domain": "my.domain",
    "repo": "github.com/example/my-project",
    "projectName": "my-project",
    "version": "3",
    "layout": ["go.kubebuilder.io/v4"],
    "multigroup": false,
    "resources": []
  }
}
```

**Example `PluginRequest` (triggered by `kubebuilder edit --plugins sampleexternalplugin/v1`):**

```json
{
  "apiVersion": "v1alpha1",
  "args": [],
  "command": "edit",
  "universe": {}
}
```

### PluginResponse

`PluginResponse` contains the modifications made by the plugin to the project. This data is serialized as JSON and returned to Kubebuilder through `stdout`.

**Example `PluginResponse`:**
```json
{
  "apiVersion": "v1alpha1",
  "command": "edit",
  "metadata": {
    "description": "The `edit` subcommand adds Prometheus instance configuration for monitoring your operator.",
    "examples": "kubebuilder edit --plugins sampleexternalplugin/v1"
  },
  "universe": {
    "config/prometheus/prometheus.yaml": "# Prometheus CR manifest...",
    "config/prometheus/kustomization.yaml": "resources:\n  - prometheus.yaml\n",
    "config/default/kustomization_prometheus_patch.yaml": "# Instructions for enabling Prometheus..."
  },
  "error": false,
  "errorMsgs": []
}
```

<aside>
<H1> </H1>

Avoid printing directly to `stdout` in your external plugin.
Since communication between Kubebuilder and the plugin occurs through
`stdin` and `stdout` using structured JSON, any unexpected output
(like debug logs) may cause errors. Write logs to a file if needed.

</aside>

## How to Use an External Plugin

### Prerequisites

- Kubebuilder CLI version > 3.11.0
- An executable for the external plugin
- Plugin path configuration using `${EXTERNAL_PLUGINS_PATH}` or default OS-based paths:
  - Linux: `$HOME/.config/kubebuilder/plugins/${name}/${version}/${name}`
  - macOS: `~/Library/Application Support/kubebuilder/plugins/${name}/${version}/${name}`

**Example:** For a plugin `foo.acme.io` version `v2` on Linux, the path would be `$HOME/.config/kubebuilder/plugins/foo.acme.io/v2/foo.acme.io`.

### Available Subcommands

External plugins can support the following Kubebuilder subcommands:
- `init`: Project initialization
- `create api`: Scaffold Kubernetes API definitions
- `create webhook`: Scaffold Kubernetes webhooks
- `edit`: Update project configuration

**Optional subcommands for enhanced user experience:**
- `metadata`: Provide plugin descriptions and examples with the `--help` flag.
- `flags`: Inform Kubebuilder of supported flags, enabling early error detection.

<aside class="note">
<h1>More about `flags` subcommand</h1>

The `flags` subcommand in an external plugin allows for early error detection by informing Kubebuilder about the flags the plugin supports. If an unsupported flag is identified, Kubebuilder can issue an error before the plugin is called to execute.
If a plugin does not implement the `flags` subcommand, Kubebuilder will pass all flags to the plugin, making it the external plugin's responsibility to handle any invalid flags.

</aside>

### Configuring Plugin Path

Set the environment variable `$EXTERNAL_PLUGINS_PATH`
to specify a custom plugin binary path:

```sh
export EXTERNAL_PLUGINS_PATH=<custom-path>
```

Otherwise, Kubebuilder would search for the plugins in a default path based on your OS.

### Example CLI Commands

You can now use it by calling the CLI commands:

```sh
# Add Prometheus monitoring to an existing project
kubebuilder edit --plugins sampleexternalplugin/v1

# Update an existing project with Prometheus monitoring
kubebuilder edit --plugins sampleexternalplugin/v1

# Display help information for the init subcommand
kubebuilder init --plugins sampleexternalplugin/v1 --help

# Display help information for the edit subcommand
kubebuilder edit --plugins sampleexternalplugin/v1 --help

# Plugin chaining example: Use go/v4 plugin first, then apply external plugin
kubebuilder edit --plugins go/v4,sampleexternalplugin/v1
```

## Further resources

- A [sample external plugin written in Go](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/simple-external-plugin-tutorial/testdata/sampleexternalplugin/v1)
- A [sample external plugin written in Python](https://github.com/rashmigottipati/POC-Phase2-Plugins)
- A [sample external plugin written in JavaScript](https://github.com/Eileen-Yu/kb-js-plugin)

[code-plugin-external]: https://github.com/kubernetes-sigs/kubebuilder/blob/book-v4/pkg/plugin/external/types.go
