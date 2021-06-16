# Versioning and Releasing for KubeBuilder

We (mostly) follow the [common KubeBuilder versioning
guidelines][guidelines], and use the corresponding tooling and PR process
described there.

For the purposes of the aforementioned guidelines, KubeBuilder counts as
a "CLI project".

[guidelines]: https://sigs.k8s.io/kubebuilder-release-tools/VERSIONING.md

## Compability

Note that we generally do not support older release branches, except in
extreme circumstances.

Bear in mind that changes to scaffolding generally constitute breaking
changes -- see [below](#understanding-the-versions) for more details.

## Releasing

When releasing, you'll need to:

- to update references in [the build directory](build/) to the latest
  version of the [envtest tools](#tools-releases) **before tagging the
  release.**

- reset the book branch: see [below](#book-releases)

You may also want to check that the book is generating the marker docs off
the latest controller-tools release.  That info is stored in
[docs/book/install-and-build.sh](/docs/book/install-and-build.sh).

## Book Releases

The book's main version (https://book.kubebuilder.io) is published off of
the [book-v3][book-branch] (a version built off the main branch can be
found at https://master.book.kubebuilder.io).

Docs changes that aren't specific to a new feature should be
cherry-picked to the aforementioned branch to get them to be published.
The cherry-picks will automatically be published to the book once their PR
merges.

**When you publish a KubeBuilder release**, be sure to also submit a PR
that merges the main branch into [book-v3][book-branch], so that it
describes the latest changes in the new release.

[book-branch]: https://github.com/kubernetes-sigs/kubebuilder/tree/tools-releases

## Tools Releases

In order to update the [envtest tools][envtest-ref], you'll need to do an
update to the [tools-releases branch][tools-branch].  Simply submit a PR
against that branch that changes all references to the current version to
the desired next version.  Once the PR is merged, Google Cloud Build will
take care of building and publishing the artifacts.

[envtest-ref]: https://book.kubebuilder.io/reference/artifacts.html
[tools-branch]: https://github.com/kubernetes-sigs/kubebuilder/tree/tools-releases

## Versioning

|   Name	|   Example	|  Description |
|---	|---	|---	|
|  KubeBuilder version | `v2.2.0`, `v2.3.0`, `v2.3.1` | Tagged versions of the KubeBuilder project, representing changes to the source code in this repository. See the [releases][kb-releases] page for binary releases. |
|  Project version |  `"1"`, `"2"`, `"3"` | Project version defines the scheme of a `PROJECT` configuration file. This version is defined in a `PROJECT` file's `version`. |
|  Plugin version | `v2`, `v3` | Represents the version of an individual plugin, as well as the corresponding scaffolding that it generates. This version is defined in a plugin key, ex. `go.kubebuilder.io/v2`. See the [design doc][cli-plugins-versioning] for more details. |

### Incrementing versions

For more information on how KubeBuilder release versions work, see the [semver](https://semver.org/) documentation.

Project versions should only be increased if a breaking change is introduced in the PROJECT file scheme itself. Changes to the Go scaffolding or the KubeBuilder CLI *do not* affect project version.

Similarly, the introduction of a new plugin version might only lead to a new minor version release of KubeBuilder, since no breaking change is being made to the CLI itself. It'd only be a breaking change to KubeBuilder if we remove support for an older plugin version. See the plugins design doc [versioning section][cli-plugins-versioning]
for more details on plugin versioning.

**NOTE:** the scheme for project version `"2"` was defined before the concept of plugins was introduced, so plugin `go.kubebuilder.io/v2` is implicitly used for those project types. Schema for project versions `"3"` and beyond define a `layout` key that informs the plugin system of which plugin to use.

## Introducing changes to plugins

Changes made to plugins only require a plugin version increase if and only if a change is made to a plugin
that breaks projects scaffolded with the previous plugin version. Once a plugin version `vX` is stabilized (it doesn't
have an "alpha" or "beta" suffix), a new plugin package should be created containing a new plugin with version
`v(X+1)-alpha`. Typically this is done by (semantically) `cp -r pkg/plugins/golang/vX pkg/plugins/golang/v(X+1)` then updating
version numbers and paths. All further breaking changes to the plugin should be made in this package; the `vX`
plugin would then be frozen to breaking changes.

You must also add a migration guide to the [migrations](https://book.kubebuilder.io/migrations.html)
section of the KubeBuilder book in your PR. It should detail the steps required
for users to upgrade their projects from `vX` to `v(X+1)-alpha`.

### Example

KubeBuilder scaffolds projects with plugin `go.kubebuilder.io/v3` by default.

You create a feature that adds a new marker to the file `main.go` scaffolded by `init`
that `create api` will use to update that file. The changes introduced in your feature
would cause errors if used with projects built with plugins `go.kubebuilder.io/v2`
without users manually updating their projects. Thus, your changes introduce a breaking change
to plugin `go.kubebuilder.io`, and can only be merged into plugin version `v3-alpha`.
This plugin's package should exist already.

[kb-releases]:https://github.com/kubernetes-sigs/kubebuilder/releases
[cli-plugins-versioning]:docs/book/src/plugins/cli-plugins.md
