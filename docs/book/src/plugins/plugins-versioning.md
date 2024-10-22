# Plugins Versioning

| Name | Example                                 | Description |
|----------|-----------------------------------------|--------|
| Kubebuilder version | `v2.2.0`, `v2.3.0`, `v2.3.1`,  `v4.2.0` | Tagged versions of the Kubebuilder project, representing changes to the source code in this repository. See the [releases][kb-releases] page for binary releases. |
| Project version | `"1"`, `"2"`, `"3"`                     | Project version defines the scheme of a `PROJECT` configuration file. This version is defined in a `PROJECT` file's `version`. |
| Plugin version | `v2`, `v3`, `v4`                        | Represents the version of an individual plugin, as well as the corresponding scaffolding that it generates. This version is defined in a plugin key, ex. `go.kubebuilder.io/v2`. See the [design doc][cli-plugins-versioning] for more details. |

### Incrementing versions

For more information on how Kubebuilder release versions work, see the [semver][semver] documentation.

Project versions should only be increased if a breaking change is introduced in the PROJECT file scheme itself. Changes to the Go scaffolding or the Kubebuilder CLI *do not* affect project version.

Similarly, the introduction of a new plugin version might only lead to a new minor version release of Kubebuilder, since no breaking change is being made to the CLI itself. It'd only be a breaking change to Kubebuilder if we remove support for an older plugin version. See the plugins design doc [versioning section][cli-plugins-versioning]
for more details on plugin versioning.

## Introducing changes to plugins

Changes made to plugins only require a plugin version increase if and only if a change is made to a plugin
that breaks projects scaffolded with the previous plugin version. Once a plugin version `vX` is stabilized (it doesn't
have an "alpha" or "beta" suffix), a new plugin package should be created containing a new plugin with version
`v(X+1)-alpha`. Typically this is done by (semantically) `cp -r pkg/plugins/golang/vX pkg/plugins/golang/v(X+1)` then updating
version numbers and paths. All further breaking changes to the plugin should be made in this package; the `vX`
plugin would then be frozen to breaking changes.

You must also add a migration guide to the [migrations][migrations]
section of the Kubebuilder book in your PR. It should detail the steps required
for users to upgrade their projects from `vX` to `v(X+1)-alpha`.

<aside class="note">

<h1>Example</h1>

Kubebuilder scaffolds projects with plugin `go.kubebuilder.io/v4` by default.

You create a feature that adds a new marker to the file `main.go` scaffolded by `init` that `create api` will use to update that file.
The changes introduced in your feature would cause errors if used with projects built with
plugins `go.kubebuilder.io/v4` without users manually updating their projects.

Thus, your changes introduce a breaking change to plugin `go.kubebuilder.io`,
and can only be merged into plugin version `v5-alpha`.
This plugin's package should exist already.

</aside>


[semver]: https://semver.org/
[migrations]: ../migrations.md
[kb-releases]:https://github.com/kubernetes-sigs/kubebuilder/releases
[design-doc]: ./extending
[cli-plugins-versioning]:./extending#plugin-versioning