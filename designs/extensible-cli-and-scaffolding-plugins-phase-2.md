| Authors       | Creation Date | Status      | Extra                                                           |
|---------------|---------------|-------------|-----------------------------------------------------------------|
| @rashmigottipati | Mar 9, 2021  | partial implemented | [Plugins doc](https://book.kubebuilder.io/plugins/plugins.html) |

# Extensible CLI and Scaffolding Plugins - Phase 2

## Overview

Plugin [Phase 1.5](https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1-5.md) was designed and implemented to allow chaining of plugins. The purpose of Phase 2 plugins is to discover and use external plugins, also referred to as out-of-tree plugins (which can be implemented in any language). Phase 2 achieves both chaining and discovery of external plugins/source code not compiled with the `kubebuilder` CLI binary. By achieving this goal, we could (for example) externalize the optional [declarative plugin](https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/golang/declarative/v1) which means that the CLI would still be able to use it, however, its source code would no longer be required to be inside of the Kubebuilder repository.

### Related issues and PRs

* [Feature Request: Plugins Phase 2](https://github.com/kubernetes-sigs/kubebuilder/issues/1378)
* [Extensible CLI and Scaffolding Plugins - Phase 1.5](https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1-5.md)
* [Phase 1.5 Implementation PR](https://github.com/kubernetes-sigs/kubebuilder/pull/2060)
* [Plugin Resolution Enhancement Proposal](https://github.com/kubernetes-sigs/kubebuilder/pull/1942)

### Prototype implementation

[POC](https://github.com/rashmigottipati/POC-Phase2-Plugins) - Invoke an external python program that simulates a plugin from a go main and pass messages from `kubebuilder` to the plugin and vice-versa using `stdin/stdout/stderr`.

### User Stories

* As a plugin developer, I would like to be able to provide external plugins path for the CLI to perform the scaffolds, so that I could take advantage of external initiatives which are implemented using Kubebuilder as a lib and following its standards but are not shipped with its CLI binaries.

* As a Kubebuilder maintainer, I would like to support external plugins not maintained by the core project.
  * For example, once the Phase 2 plugin implementation is completed, some internal plugins can be re-implemented as external plugins removing the necessity to build those plugins in the `kubebuilder` binary.

### Goals

* `kubebuilder` is able to discover plugin binaries and run those plugins using the CLI.

* Kubebuilder can use the external plugins as well as its own internal ones to do scaffolding.

* `kubebuilder` should be able to show plugin specific information via the `--help` flag.

* Support for standard streams i.e. `stdin/stdout/stderr` as the only IPC method between `kubebuilder` and plugins.

* Kubebuilder library consumers can support chaining and discovery of out-of-tree plugins.

### Non-Goals

* Addition of new arbitrary subcommands other than the subcommands that we already support i.e `init`, `create api`, and `create webhook`.

* Discovering plugin binaries that are not locally present on the machine (i.e. binary exists in a remote repository).

* Providing other options (other than standard streams such as `stdin/stdout/stderr`) for inter-process communication between `kubebuilder` and external plugins.
  * Other IPC methods may be allowed in the future, although EPs are required for those methods.

### Examples

* `kubebuilder create api --plugins=myexternalplugin/v1`
  * should scaffold files using the external plugin as defined in its implementation of the `create api` method.

* `kubebuilder create api --plugins=myexternalplugin/v1,myotherexternalplugin/v2`
  * should scaffold files using the external plugin as defined in their implementation of the `create api` method (by respecting the plugin chaining order, i.e. in the order of `create api` of v1 and then `create api` of v2 as specified in the layout field in the configuration).

* `kubebuilder create api --plugins=myexternalplugin/v1 --help`
  * should display help information of the plugin which is not shipped in the binary (myexternalplugin/v1 is present outside of the `kubebuilder` binary).

* `kubebuilder create api --plugins=go/v3,myexternalplugin/v2`
  * should create files using the `go/v3` plugin, then pass those files to `myexternalplugin/v2` as defined in its implementation of the `create api` method by respecting the plugin chaining order.

## Proposal

### Discovery of plugin binaries

The method [kustomize](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/) uses to discover plugins, by following a GVK path scheme, is the most natural for this use case since plugins must have a group-like name and version.

Every plugin gets its own directory constructed using the plugin name and plugin version for the executable to be placed in and `kubebuilder` will search for a plugin binary with the name of the plugin in the `${name}/${version}` directory of the plugin. This information (plugin name and plugin version) is obtained by `kubebuilder` via the value passed to the `--plugins` CLI flag. Once `kubebuilder` successfully locates the plugin, it will run the plugin using the CLI.

Every plugin gets its own directory as below.

On Linux:

```shell
    $XDG_CONFIG_HOME/kubebuilder/plugins/${name}/${version}
```

The default value of XDG_CONFIG_HOME is `$HOME/.config`.

On OSX:

```shell
    ~/Library/Application Support/kubebuilder/plugins/${name}/${version}
```

Based on the above directory scheme, let's say that if the value passed to the `--plugins` CLI flag is `myexternalplugin/v1`:

* On Linux:
  * `kubebuilder` will search for the `myexternalplugin` binary in `$XDG_CONFIG_HOME/kubebuilder/plugins/myexternalplugin/v1`, where the base of this path in is the binary name.
* On OSX:
  * Kubebuilder will search for the `myexternalplugin` binary in `$HOME/Library/Application Support/kubebuilder/plugins/myexternalplugin/v1`.

Note: If the name is ambiguous, then the qualified name `myexternalplugin.my.domain` would be used, so the path would be `$XDG_CONFIG_HOME/kubebuilder/plugins/my/domain/myexternalplugin/v1` on Linux and `$HOME/Library/Application Support/kubebuilder/plugins/my/domain/myexternalplugin/v1` on OSX.

* Pros
  * `kustomize` which is popular and robust tool, follows this approach in which `apiVersion` and `kind` fields are used to locate the plugin.

  * This approach enforces naming constraints as the permitted character set must be directory name-compatible following naming rules for both Linux and OSX systems.

  * The one-plugin-per-directory requirement eases creation of a plugin bundle for sharing.

### What Plugin system should we use

I propose we use our own plugin system that passes JSON blobs back and forth across `stdin/stdout/stderr` and make this the only option for now as it’s a language-agnostic medium and it is easy to work with in most languages.

We came to the conclusion that a kubebuilder-specific plugin library should be written after evaluating plugin libraries such as the [built-in go-plugin library](https://golang.org/pkg/plugin/) and [Hashicorp’s plugin library](https://github.com/hashicorp/go-plugin):

* The built-in plugin library seems to be more suitable for in-tree plugins rather than out-of-tree plugins and it doesn’t offer cross-language support, thereby making it a non-starter.
* Hashicorp’s go plugin system is more suitable than the built-in go-plugin library as it enables cross language/platform support. However, it is more suited for long running plugins as opposed to short lived plugins and the usage of protobuf could be overkill as we will not be handling 10s of 1000s of deserializations.

In the future, if a need arises (for example, users are hitting performance issues), we can then explore the possibility of using the Hashicorp’s go plugin library. From a design standpoint, to leave it architecturally open, I propose using a `type` field in the PROJECT file to potentially allow other plugin libraries in the future and make this a seperate field in the PROJECT file per plugin; and this field determines how the `universe` will be passed for a given plugin. However, for the sake of simplicity in initial design and not to introduce any breaking changes as Project version 3 would suffice for our needs, this option is out of scope in this proposal.

### Project configuration

Currently, the project configuration has two fields to store plugin specific information.

* `Layout` field (of type []string) is used for plugin chain resolution on initialized projects. This will be the default if no plugins are specified for a subcommand.
* `Plugins` field (of type map[string]interface{}) is used for option plugin configuration that stores configuration information of any plugin.

* So, where should external plugins be defined in the configuration?

  * I propose that the external plugin should get encoded in the project configuration as a part of the `layout` field.
    * For example, external plugin `myexternalplugin/v2` can be specified through the `--plugins` flag for every subcommand and also be defined in the project configuration in the `layout` field for plugin resolution.

Example `PROJECT` file:

```yaml
version: "3"
domain: testproject.org
layout:
- go.kubebuilder.io/v3
- myexternalplugin/v2
plugins:
  myexternalplugin/v2:
    resources:
    - domain: testproject.org
      group: crew
      kind: Captain
      version: v2
  declarative.go.kubebuilder.io/v1:
    resources:
    - domain: testproject.org
      group: crew
      kind: FirstMate
      version: v1
repo: github.com/test-inc/testproject
resources:
- group: crew
  kind: Captain
  version: v1
```

### Communication between `kubebuilder` and external plugins

* Why do we need communication between `kubebuilder` and external plugins?

  * The in-tree plugins do not need any inter-process communication as they are the same process, and hence, direct calls are made to the respective functions (also referred as hooks) based on the supported subcommands for an in-tree plugin. As Phase 2 plugins is tackling out-of-tree or external plugins, there's a need for inter-process communication between `kubebuilder` and the external plugin as they are two separate processes/binaries. `kubebuilder` needs to communicate the subcommand that the external plugin should run, and all the arguments received in the CLI request by the user. These arguments contain flags which will have to be directly passed to all plugins in the chain. Additionally, it's important to have context of all the files that were scaffolded until that point especially if there is more than one external plugin in the chain. `kubebuilder` attaches that information in the request, along with the command and arguments. For the external plugin, it would need to communicate the subcommand it ran and the updated file contents information that the external plugin scaffolded to `kubebuilder`. The external plugin would also need to provide its help text if requested by `kubebuilder`. As discussed earlier, standard streams seems to be a desirable IPC method of communication for the use-cases that Phase 2 is trying to solve that involves discovery and chaining of external plugins.

* How does `kubebuilder` communicate to external plugins?

  * Standard streams have three I/O connections: standard input (`stdin`), standard output (`stdout`) and standard error (`stderr`) and they work well with chaining applications, meaning that output stream of one program can be redirected to the input stream of another.
  * Let's say there are two external plugins in the plugin chain. Below is the sequence of how `kubebuilder` communicates to the plugins `myfirstexternalplugin/v1` and `mysecondexternalplugin/v1`.

![Kubebuilder to external plugins sequence diagram](https://github.com/rashmigottipati/POC-Phase2-Plugins/blob/main/docs/externalplugins-sequence-diagram.png)

* What to pass between `kubebuilder` and an external plugin?

Message passing between `kubebuilder` and the external plugin will occur through a request / response mechanism. The `PluginRequest` will contain information that `kubebuilder` sends *to* the external plugin. The `PluginResponse` will contain information that `kubebuilder` receives *from* the external plugin.

The following scenarios shows what `kubebuilder` will send/receive to the external plugin:

* `kubebuilder` to external plugin:
  * `kubebuilder` constructs a `PluginRequest` that contains the `Command` (such as `init`, `create api`, or `create webhook`), `Args` containing all the raw flags from the CLI request and license boilerplate without comment delimiters, and an empty `Universe` that contains the current virtual state of file contents that is not written to the disk yet. `kubebuilder` writes the `PluginRequest` through `stdin`.

* External plugin to `kubebuilder`:
  * The plugin reads the `PluginRequest` through its `stdin` and processes the request based on the `Command` that was sent. If the `Command` doesn't match what the plugin supports, it writes back an error immediately without any further processing. If the `Command` matches what the plugin supports, it constructs a `PluginResponse` containing the `Command` that was executed by the plugin, and modified `Universe` based on the new files that were scaffolded by the external plugin, `Error` and `ErrorMsg` that add any error information, and writes the `PluginResponse` back to `kubebuilder` through `stdout`.

* Note: If `--help` flag is being passed from `kubebuilder` to the external plugin through `PluginRequest`, the plugin attaches its help text information in the `Metadata` field of the `PluginResponse`. Both `PluginRequest` and `PluginResponse` also contain `APIVersion` field to have compatible versioned schemas.

* Handling plugin failures across the chain:

  * If any plugin in the chain fails, the plugin reports errors back through `PluginResponse` to `kubebuilder` and plugin chain execution will be halted, as one plugin may be dependent on the success of another. All the files that were scaffolded already until that point will not be written to disk to prevent a half committed state.

## Implementation Details/Notes/Constraints

`PluginRequest` holds all the information `kubebuilder` receives from the CLI and the plugins that were executed before it and the `PluginRequest` will be marshaled into a JSON and sent over `stdin` to the external plugin. `PluginResponse` is what the plugin constructs with the updated universe and sent back to `kubebuilder`. The following structs would be defined on the Kubebuilder side.

```go
// PluginRequest contains all information kubebuilder received from the CLI
// and plugins executed before it.
type PluginRequest struct {
  // Command contains the command to be executed by the plugin such as init, create api, etc.
  Command       string              `json:"command"`

  // APIVersion defines the versioned schema of the PluginRequest that is encoded and sent from Kubebuilder to plugin.
  // Initially, this will be marked as alpha (v1alpha1).
  APIVersion    string              `json:"apiVersion"`

  // Args holds the plugin specific arguments that are received from the CLI which are to be passed down to the plugin.
  Args          []string            `json:"args"`

  // Universe represents the modified file contents that gets updated over a series of plugin runs
  // across the plugin chain. Initially, it starts out as empty.
  Universe      map[string]string   `json:"universe"`
}

// PluginResponse is returned to kubebuilder by the plugin and contains all files
// written by the plugin following a certain command.
type PluginResponse struct {
  // Command holds the command that gets executed by the plugin such as init, create api, etc.
  Command       string                   `json:"command"`

  // Metadata contains the plugin specific help text that the plugin returns to Kubebuilder when it receives
  // `--help` flag from Kubebuilder.
  Metadata plugin.SubcommandMetadata `json:"metadata"`

  // APIVersion defines the versioned schema of the PluginResponse that will be written back to kubebuilder.
  // Initially, this will be marked as alpha (v1alpha1).
  APIVersion    string                   `json:"apiVersion"`

  // Universe in the PluginResponse represents the updated file contents that was written by the plugin.
  Universe      map[string]string        `json:"universe"`

  // Error is a boolean type that indicates whether there were any errors due to plugin failures.
  Error         bool                     `json:"error,omitempty"`

  // ErrorMsg holds the specific error message of plugin failures.
  ErrorMsg      string                   `json:"error_msg,omitempty"`
}
```

The following function handles construction of the `PluginRequest` based on the information `kubebuilder` receives from the CLI and the request is marshaled into JSON. The command to run the external plugin by providing the plugin path will be invoked and `kubebuilder` will send the marshaled `PluginRequest` JSON to the plugin over `stdin`.

```go
func (p *ExternalPlugin) runExternalProgram(req PluginRequest) (res PluginResponse, err error) {
  pluginReq, err := json.Marshal(req)
  if err != nil {
    return res, err
  }

  cmd := exec.Command(p.Path)
  cmd.Dir = p.DirContext
  cmd.Stdin = bytes.NewBuffer(pluginReq)
  cmd.Stderr = os.Stderr

  out, err := cmd.Output()
  if err != nil {
    fmt.Fprint(os.Stdout, string(out))
    return res, err
  }

  if json.Unmarshal(out, &res); err != nil {
    return res, err
  }
  return res, nil
}
```

On the plugin side, the request JSON will be decoded and depending on what the `Command` in the `PluginRequest` is, the corresponding function to handle `init` or `create api` will be invoked thereby modifying the universe by writing the updated files to it. After `init` or `create api` functions execute successfully, the plugin will write back `PluginResponse` with updated universe and errors (if any) in JSON format through `stdout` to `kubebuilder`. `PluginResponse` also contains error fields `Error` and `ErrorMsg` that the plugin can utilize to add error context if any errors occur.
`kubebuilder` receives the command output and decodes into `PluginResponse` struct. This is how message passing will occur between `kubebuilder` and the external plugin. Refer to [POC](https://github.com/rashmigottipati/POC-Phase2-Plugins) for specifics.

### Simple Example

```shell
kubebuilder init --plugins=myexternalplugin/v1 --domain example.com
```

What happens when the above is invoked?

![Kubebuilder to external plugins](https://github.com/rashmigottipati/POC-Phase2-Plugins/blob/main/docs/externalplugins-sequence-diagram-2.png)

* `kubebuilder` discovers `myexternalplugin/v1` plugin binary and runs the plugin from the discovered path.

* Send `PluginRequest` as a JSON over `stdin` to `myexternalplugin` plugin.

`PluginRequest JSON`:

```JSON
{
  "command":"init",
  "args":["--domain","example.com"],
  "universe":{}
}
```

* `myexternalplugin` plugin parses the `PluginRequest` and based on the `Command` specified in the request i.e `init`, performs the necessary scaffolding.

* `myexternalplugin` plugin constructs `PluginResponse` with modified `Universe` that contains the updated file contents and errors if any.

* Plugin writes `PluginResponse` to stdout in a JSON format back to `kubebuilder`.

* `kubebuilder` receives the command output containing the `PluginResponse` JSON which will be decoded into the `PluginResponse` struct.

* `kubebuilder` writes the files in the universe to disk.

`PluginResponse JSON`:

```JSON
{
  "command": "init",
  "universe": {
    "LICENSE": "Apache 2.0 License\n",
    "main.py": "..."
  }
}

```

## Alternatives

### Plugin discovery

#### User specified file paths

A user will provide a list of file paths for `kubebuilder` to discover the plugins in. We will define a variable `KUBEBUILDER_PLUGINS_DIRS` that will take a list of file paths to search for the plugin name. It will also have a default value to search in, in case no file paths are provided. It will search for the plugin name that was provided to the `--plugins` flag in the CLI. `kubebuilder` will recursively search for all file paths until the plugin name is found and returns the successful match, and if it doesn’t exist, it returns an error message that the plugin is not found in the provided file paths. Also use the host system mechanism for PATH separation.

* Alternatively, this could be handled in a way that [helm kustomize plugin](https://helm.sh/docs/topics/advanced/#post-rendering) discovers the plugin based on the non-existence of a separator in the path provided, in which case `kubebuilder` will search in `$PATH`, otherwise resolve any relative paths to a fully qualified path.

* Pros
  * This provides flexibility for the user to specify the file paths that the plugin would be placed in and `kubebuilder` could discover the binaries in those user specified file paths.

  * No constraints on plugin binary naming or directory placements from the Kubebuilder side.

  * Provides a default value for the plugin directory in case user wants to use that to drop their plugins.

#### Prefixed plugin executable names in $PATH

Another approach is adding plugin executables with a prefix `kubebuilder-` followed by the plugin name to the PATH variable. This will enable `kubebuilder` to traverse through the PATH looking for the plugin executables starting with the prefix `kubebuilder-` and matching by the plugin name that was provided in the CLI. Furthermore, a check should be added to verify that the match is an executable or not and return an error if it's not an executable. This approach provides a lot of flexibility in terms of plugin discovery as all the user needs to do is to add the plugin executable to the PATH and `kubebuilder` will discover it.

* Pros
  * `kubectl` and `git` follow the same approach for discovering plugins, so there’s prior art.

  * There’s a lot of flexibility in just dropping plugin binaries to PATH variable and enabling the discovery without having to enforce any other constraints on the placements of the plugins.

* Cons
  * Enumerating the list of all available plugins might be a bit tough compared to having a single folder with the list of available plugins and having to enumerate those.

  * These plugin binaries cannot be run in a standalone manner outside of Kubebuilder, so may not be very ideal to add them to the PATH var.

## Open questions

* Do we want to support the addition of new arbitrary subcommands other than the subcommands (init, create api, create webhook) that we already support?
  * Not for the EP or initial implementation, but can revisit later.

* Do we need to discover flags by calling the plugin binary or should we have users define them in the project configuration?
  * Flags will be passed directly to the external plugins as a string. Flag parse errors will be passed back via `PluginResponse`.

* What alternatives to stdin/stdout exist and why shouldn't we use them?
  * Other alternatives exist such as named pipe and sockets, but stdin/stdout seems to be more suitable for our needs.

* What happens when two plugins bind the same flag name? Will there be any conflicts?
  * As mentioned in the implementation details section, flags are passed directly as a string to plugins and the same string will be passed to each plugin in the chain, so all plugins get the same flag set. Errors should not be returned if an unrecognized flag is parsed.

* How should we handle environment variables?
  * We would pass the entire CLI environment to the plugin to permit simple external plugin configuration without jumping through hoops.

* Should the API version be a part of the plugin request spec?
  * It would be nice to encode APIVersion for `PluginRequest` and `PluginResponse` so the initial schemas can be marked as `v1alpha1`.
