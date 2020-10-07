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
the [book-v2][book-branch] (a version built off the main branch can be
found at https://master.book.kubebuilder.io).

Docs changes that aren't specific to a new feature should be
cherry-picked to the aforementioned branch to get them to be published.
The cherry-picks will automatically be published to the book once their PR
merges.

**When you publish a KubeBuilder release**, be sure to also submit a PR
that merges the main branch into [book-v2][book-branch], so that it
describes the latest changes in the new release.

[book-branch]: https://github.com/kubernetes-sigs/kubebuilder/tree/tools-releases

## Tools Releases

In order to update the [envtest tools][envtest-ref], you'll need to do an
update to the [tools-releases branch][tools-branch].  Simply submit a PR
against that branch that changes all references to the current version to
the desired next version.  Once the PR is merged, Google Cloud Build will
take care of building and publishing the artifacts.

[envtest-ref]: https://book.kubebuilder.io/reference/artifacts.html
[tools-branch]: https://github.com/kubernetes-sigs/kubebuilder/tree/tools-releases)

## Understanding the versions

|   Name	|   Example	|  Description |
|---	|---	|---	|
|  PROJECT version |  `v1`,`v2`,`v3` | As of the introduction of [Extensible CLI and Scaffolding Plugins](https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md), PROJECT version represents the layout of PROJECT file itself.  For `v1` and `v2` projects, there's extra meaning -- see below.  |
|  Release version | `v2.2.0`, `v2.3.0`, `v2.3.1` | Tagged versions of the KubeBuilder project, representing changes to the source code in this repository. See the [releases](https://github.com/kubernetes-sigs/kubebuilder/releases) page. |
|  Plugin Versions | `go.kubebuilder.io/v2.0` | Represents the version of an individual plugin, as well as the corresponding scaffolding that it generates. |

Note that PROJECT version should only be bumped if a breaking change is introduced in the PROJECT file format itself.  Changes to the Go scaffolding or the KubeBuilder CLI *do not* affect the PROJECT version.

Similarly, the introduction of a new major version (`(x+1).0`) of the Go plugin might only lead to a new minor (`x.(y+1)`) release of KubeBuilder, since no breaking change is being made to the CLI itself.  It'd only be a breaking change to KubeBuilder if we remove support for an older version of the plugin.

For more information on how the release and plugin versions work, see the [semver](https://semver.org/) documentation.

**NOTE:** In the case of the `v1` and `v2` PROJECT version, a corresponding Plugin version is implied and constant -- `v1` implies the `go.kubebuilder.io/v1` Plugin, and similarly for `v2`.  This is for legacy purposes -- no such implication is made with the `v3` PROJECT version.

## Introducing changes in the scaffold files

Changes in the scaffolded files require a new Plugin version. If we delete or update a file that is scaffolded by default, it's a breaking change and requires a `MAJOR` Plugin version.  If we add a new file, it may not be a breaking change.

**More simply:** any change that will break the expected behaviour of a project built with the previous `MINOR` Plugin versions is a breaking change to that plugin. 

**EXAMPLE:**

KubeBuilder Release version (`5.3.1`) scaffolds projects with the plugin version `3.2` by default.

The changes introduced in our PR will not work well with the projects which were built with the plugin versions `3.0...3.5` without users taking manual steps to update their projects. Thus, our changes introduce a breaking change to the Go plugin, and require a `MAJOR` Plugin version bump.

In the PR, we should add a migration guide to the [Migrations](https://book.kubebuilder.io/migrations.html) section of the KubeBuilder book. It should detail the required steps that users should take to upgrade their projects from `go.kubebuilder.io/3.X` to the new `MAJOR` Plugin version `go.kubebuilder.io/4.0`.

This also means we should introduce a new KubeBuilder minor version `5.4` when the project is released: since we've only added a new plugin version without removing the old one, this is considered a new feature to KubeBuilder, and not a breaking change.

**IMPORTANT** Breaking changes cannot be made to PROJECT versions v1 and v2, and consequently plugin versions `go.kubebuilder.io/1` and `go.kubebuilder.io/2`.

