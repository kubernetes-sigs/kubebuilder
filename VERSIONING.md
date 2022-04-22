# Versioning and Releasing for Kubebuilder

We (mostly) follow the [common Kubebuilder versioning
guidelines][guidelines], and use the corresponding tooling and PR process
described there.

For the purposes of the aforementioned guidelines, Kubebuilder counts as
a "CLI project".

[guidelines]: https://sigs.k8s.io/kubebuilder-release-tools/VERSIONING.md

## Compatibility

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

**When you publish a Kubebuilder release**, be sure to also submit a PR
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
[kb-releases]:https://github.com/kubernetes-sigs/kubebuilder/releases
[cli-plugins-versioning]:docs/book/src/plugins/extending-cli.md#plugin-versioning
