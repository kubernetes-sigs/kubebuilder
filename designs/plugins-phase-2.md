# Plugins Phase 2

## Overview

Phase 2 of the scaffold plugin system adds support for chaining plugins together. The idea for plugins phase 2 is to have support for a plugin architecture that enables plugins to be separate binaries discoverable by the Kubebuilder CLI binary via user specified plugin file paths and that they are language indepedent. Some of the phase 2 high-level requirements summarized from the feature request and discussions are: 

* mdbook-like structure
    * A Plugin receives a JSON blob plugin "universe" from upstream plugins containing file "nodes", each of which contain file metadata.
    * The plugin can add or delete nodes (nodes are not mutable), then returns the modified universe so downstream plugins can do the same.
* Pre-generate a project file for handling complex plugin use-cases.
* Enhanced "layout" PROJECT file key.
    * Either an ordered list of plugin {name}/{version}, or ordered map of {name}/{version} to plugin-specific config.
    * Order matters for certain plugins, ex. a bazel plugin would need to run after all Go files are generated.

## Evaluation of Plugin libraries 

##### Go built-in plugin package vs Hashicorp plugin library vs custom plugin library

Based on the evaluation of plugin libraries such as the built-in go-plugin library and Hashicorp’s plugin library, we have come to the conclusion that it is more suitable to write our own custom typed plugin library.

The built-in plugin library seems to be a non-starter as it is more suitable for in-tree plugins rather than out-of-tree and it doesn’t offer cross language support. Hashicorp’s go plugin system seems more suitable than built-in go-plugin library as it enables cross language/platform support. However, it is more suited for long running plugins as opposed to short lived plugins and it could be overkill with the usage of protobuf.
 
For the stated reasons, the proposal is to write our own plugin system that passes JSON blobs back and forth across stdin/stdout.
The plugin specification will include type metadata to allow the potential for using other plugin libraries in the future if the need arises. From an implementation standpoint, we could create a new implementation of our plugin interface based on the type specified in the project file. We can do a type switch on the type of plugin the data is being passed to. For ex: if it is native go, then we pass the universe directly & if it is a binary wrapper, then we pass the serialized stream of JSON bytes to it. 

For phase 2 plugin implementation, plugins need a data structure containing a set of file representations to read from and write to that can be passed between multiple plugins. Let's look at that below.

#### Intermediate Representation

The proposed plugin universe is: 

```go
type File struct {
    // Path is the file to write
    Path string `json:"path,omitempty"`
    
    // Contents is the generated output
    Contents string `json:"contents,omitempty"`
    
    // Info is the os.FileInfo 
    Info os.FileInfo  `json:"info,omitempty"`	
}
 
type Universe struct {
    // Config stores the project file  
    Config *config.Config `json:"config,omitempty"`

    // Files contains the model of the files that are being scaffolded
    Files map[string]File `json:"files,omitempty"`	
}

```

#### Implementation Details

The project file has a plugins field and one of the sub-fields of this field should be “type” that specifies either stdin/stdout or JSON. When kubebuilder reads plugins from the project file, we will have a new implementation of the plugin interface and it will be based on the type. For ex: if the plugin type is stdinStdout then we pass that struct which would implement the plugin interface and we would call a function and pass in the universe and that function returns the updated universe. The underlying implementation calls exec function on the binary and encodes into stdin stream and decodes from stdout stream into the structs we expect. So, Kubebuilder, as the calling side, is calling a function that sends a universe and receives an updated universe and the serialization happens on the kubebuilder side under the hood. And the plugin type is JSON that could be exec with stdin stdout, then we expect to get the universe as JSON stdin, read JSON, and serialize and write back on the stdout the JSON that’s expected by the kubebuilder. In the future, if we could like to extend the use cases and for some reason we would like to create a new plugin type (for example, the hashicorp’s go plugin) and in that case, instead of marshalling to JSON and setting up stdin stdout, we will receive the universe and marshal to the grpc proto format and send it over GRPC, and the plugin we ran should start up a grpc server on it’s own and implement the server interface and send the response back and we unpack that into the struct we expect.

Boilerplate is not currently in the plugin Universe because it’s language specific and depending on the language, it could change. Some alternatives to this are using a boilerplate path and the plugin could check for its existence, or it could exist in the universe not as a special file but as a list of strings without comments or markers which could be read line by line, thereby adding it to any file. Along with the above `plugin.Universe`, all the raw flags that would be passed to the subcommand need to be passed along across stdin/stdout. And the idea here is to make the plugin generic such that the args are passed as they are without specifying the resource explicitly, which eliminates the need to write a plugin struct to support every possible subcommand like init, create api, and create webhook. 


## Configuration

### Project file plugin type & enhanced `layout`

Currently, the `PROJECT` file specifies what base plugin generated the project under a `layout` key (which is the plugin semantic version) in the format: `Plugin.Name() + "/" + Plugin.Version()`.

To support phase 2 plugins, we propose that we modify the "layout" PROJECT file key to either of these options as order matters for certain plugins: 
    * Ordered list of plugin {name}/{version} or
    * Ordered map of {name}/{version} to plugin-specific config

Example `PROJECT` file:

```yaml
version: "3-alpha"
layout: ["go/v1.0.0", "go.kubebuilder.io/v3-alpha"]
domain: testproject.org
repo: github.com/test-inc/testproject
resources:
- group: crew
  kind: Captain
  version: v1
plugins:
  - go.sdk.operatorframework.io/v2-alpha: {
      type: JSON
  }
``` 

## CLI

Idea is to add support for a global --plugins flag that takes an ordered list of plugin names. The subcommand invoked with --plugins executes "downstream" plugins from that list that match those the kubebuilder binary knows about.

Given that we want to pass some initial universe between plugins, that universe should be initialized by the plugin invoked via CLI (the "base" plugin). For example, kubebuilder init --plugins addon will invoke an Init plugin, which passes its generated state to the addon plugin, modifying scaffolded files then writing them to disk.

## Prototype implementation

Kubebuilder feature branch: https://github.com/kubernetes-sigs/kubebuilder/tree/feature/plugins-part-2-electric-boogaloo

## Related issues and PRs

* [Feature request: Plugins Phase 2](https://github.com/kubernetes-sigs/kubebuilder/issues/1378)

## Comments & Questions

* How to handle help text? Are plugins allowed to define their own flags?
* Do we allow plugins to depend on environment variables? 