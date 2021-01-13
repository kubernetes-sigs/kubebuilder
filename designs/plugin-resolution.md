# Plugin and project version resolution

## Old behavior

The same process is followed for all commands, without distinction between `init` and the rest. The only difference is
that, for `init`, the configuration file will not be present.

1. Obtain the project version from project configuration file's `version` field, `--project-verison` flag, or default
   project version configured during CLI creation (`cli.New(cli.WithDefaultProjectVersion(...))`).
    * In case both a project configuration file and the flag are found, fail if they are different.
2. Obtain the plugin keys from the project configuration file's `layout` field, `--plugins` flag, or default plugins
   configured during CLI creation for the above project version (`cli.New(cli.WithDefaultPlugins(...))`).
    * In case both a project configuration file and the flag are found, fail if they are different.
3. In case any of the plugin keys is not fully qualified (full name and version), check if there is only one plugin that
   fits that unqualified plugin key.
4. Resolve the plugins with the list of qualified keys above.
5. Verify the plugins support the project version.
   

### Flaws

* Default plugins need to be provided per project version
* Project version can't be resolved (e.g., when a single project version is supported by the provided plugins).
* Command calls are not allowed to override plugin keys.

## Proposal

The following plugin resolution algorithm solves the flaws described above:

1. [All commands but Init] Obtain the project version and plugin keys from the project configuration if available.
     * If they are available, plugin keys will be fully qualified.
2. [Init command only] Obtain the project version from `--project-version` flag.
     * The flag is optional, we will try to resolve project version later in case it is not provided.
3. Obtain the optional plugin keys from `--plugins` flag.
     * [Init command only] If not present, use default plugins.
     * [All commands but Init] If not present, use plugin keys from configuration file (step 1).
4. Qualify the unqualified keys from step 3.
     * If we know the project version, use this info to filter the available plugins so that we can be more accurate.
5. Resolve the plugins with the list of qualified keys above.
6. [Init command only] Resolve the project version in case it wasn't provided and there is only one project version
   supported by all the resolved plugins.
   
### Example

Based on the following main.go file:
```go
package main

import (
   "log"

   "sigs.k8s.io/kubebuilder/v2/pkg/cli"
   "sigs.k8s.io/kubebuilder/v2/pkg/model/config"
   pluginv2 "sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v2"
   pluginv3 "sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3"
)

func main() {
   c, err := cli.New(
      cli.WithCommandName("kubebuilder"),
      cli.WithVersion(versionString()),
      cli.WithPlugins(
         &pluginv2.Plugin{},
         &pluginv3.Plugin{},
      ),
      cli.WithDefaultPlugins(&pluginv3.Plugin{}),
      cli.WithCompletion,
   )
   if err != nil {
      log.Fatal(err)
   }
   if err := c.Run(); err != nil {
      log.Fatal(err)
   }
}
```

As the `go.kubebuilder.io/v3` only supports project version "3", the project version can be resolved and doesn't
need to be specified in the flags.
```
kubebuilder init --plugins=go.kubebuilder.io/v3
```

```
kuebebuilder create api --plugins=declarative.kubebuilder.io/v1 ...
```
For this command, the declarative plugin will be used instead of the base plugin,
but following command calls won't use it unless provided again.