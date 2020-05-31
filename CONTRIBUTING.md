# Contributing guidelines

This document describes how to contribute to the project.

## Sign the CLA

Kubernetes projects require that you sign a Contributor License Agreement (CLA) before we can accept your pull requests.

Please see https://git.k8s.io/community/CLA.md for more info.

## Prerequisites

- [go](https://golang.org/dl/) version v1.13+.
- [docker](https://docs.docker.com/install/) version 17.03+.
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) version v1.11.3+.
- [kustomize](https://sigs.k8s.io/kustomize/docs/INSTALL.md) v3.1.0+
- Access to a Kubernetes v1.11.3+ cluster.

## Contributing steps

1. Submit an issue describing your proposed change to the repo in question.
1. The [repo owners](OWNERS) will respond to your issue promptly.
1. If your proposed change is accepted, and you haven't already done so, sign a Contributor License Agreement (see details above).
1. Fork the desired repo, develop and test your code changes.
1. Submit a pull request.

## How to build kubebuilder locally

Note that, by building the kubebuilder from the source code we are allowed to test the changes made locally.

1. Run the following command to clone your fork of the project locally in the dir /src/sigs.k8s.io/kubebuilder

```
$ git clone git@github.com:<user>/kubebuilder.git $GOPATH/src/sigs.k8s.io/kubebuilder
```

1. Ensure you activate module support before continue (`$ export GO111MODULE=on`)
1. Run the command `make install` to create a bin with the source code

**NOTE** In order to check the local environment run `make go-test`.

## What to do before submitting a pull request

1. Run the script `make generate` to update/generate the mock data used in the e2e test in `$GOPATH/src/sigs.k8s.io/kubebuilder/testdata/`

**IMPORTANT:** The `make generate` is very helpful. By using it, you can check if good part of the commands still working successfully after the changes. Also, note that its usage is a pre-requirement to submit a PR.

Following the targets that can be used to test your changes locally.

|   Command	|   Description	|  Is called in the CI?  	|
|---	|---	|---	|
| make go-test |  Runs go tests | no   	|
| make test| Runs tests in shell (`./test.sh`)	|  yes 	|
| make lint |  Check the code implementation | yes   |
| make test-coverage |  Run coveralls to check the % of code covered by tests | yes   |
| make check-testdata |  Checks if the testdata dir is updated with the latest changes | yes   |
| make test-e2e-local |  Runs the CI e2e tests locally | no   |

**NOTE** To use the `make lint` is required to install `golangci-lint` locally. More info: https://github.com/golangci/golangci-lint#install

## Where the CI Tests are configured

1. See the [Travis](.travis.yml) file to check its tests and the scripts used on it.
1. Note that the prow tests used in the CI are configured in [kubernetes-sigs/kubebuilder/kubebuilder-presubmits.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/kubebuilder/kubebuilder-presubmits.yaml).
1. Check that all scripts used by the CI are defined in the project.  

## How to contribute to docs

We currently have 2 production branches, `book-v2` and `book-v1`. `book-v2` maps
to `book.kubebuilder.io` which contains our latest released features, while
`book-v1` maps to `book-v1.book.kubebuilder.io`, which contains our legacy docs
for kubebuilder V1.

Docs for unreleased features live in the `master` branch. We merge the `master`
branch into the `book-v2` branch when doing the releases.

If adding doc for an unreleased feature, the PR should target `master` branch.
If updating existing docs, the PR should target `master` branch and then
cherry-picked into `book-v2` branch.

### How to preview the changes performed in the docs

Check the CI job after to do the Pull Request and then, click on in the `Details` of `netlify/kubebuilder/deploy-preview`

## Versioning

|   Name	|   Example	|  Description |
|---	|---	|---	|
|  KubeBuilder version | `v2.2.0`, `v2.3.0`, `v2.3.1` | Tagged versions of the KubeBuilder project, representing changes to the source code in this repository. See the [releases][kb-releases] page for binary releases. |
|  Project version |  `"1"`, `"2"`, `"3-alpha"` | Project version defines the scheme of a `PROJECT` configuration file. This version is defined in a `PROJECT` file's `version`. |
|  Plugin version | `v2`, `v3-alpha` | Represents the version of an individual plugin, as well as the corresponding scaffolding that it generates. This version is defined in a plugin key, ex. `go.kubebuilder.io/v2`. See the [design doc][cli-plugins-versioning] for more details. |

### Incrementing versions

For more information on how KubeBuilder release versions work, see the [semver](https://semver.org/) documentation.

Project versions should only be increased if a breaking change is introduced in the PROJECT file scheme itself. Changes to the Go scaffolding or the KubeBuilder CLI *do not* affect project version.

Similarly, the introduction of a new plugin version might only lead to a new minor version release of KubeBuilder, since no breaking change is being made to the CLI itself. It'd only be a breaking change to KubeBuilder if we remove support for an older plugin version. See the plugins design doc [versioning section][cli-plugins-versioning]
for more details on plugin versioning.

**NOTE:** the scheme for project version `"2"` was defined before the concept of plugins was introduced, so plugin `go.kubebuilder.io/v2` is implicitly used for those project types. Schema for project versions `"3-alpha"` and beyond define a `layout` key that informs the plugin system of which plugin to use.

## Introducing changes to plugins

Changes made to plugins only require a plugin version increase if and only if a change is made to a plugin
that breaks projects scaffolded with the previous plugin version. Once a plugin version `vX` is stabilized (it doesn't
have an "alpha" or "beta" suffix), a new plugin package should be created containing a new plugin with version
`v(X+1)-alpha`. Typically this is done by (semantically) `cp -r pkg/plugin/vX pkg/plugin/v(X+1)` then updating
version numbers and paths. All further breaking changes to the plugin should be made in this package; the `vX`
plugin would then be frozen to breaking changes.

### Example

KubeBuilder scaffolds projects with plugin `go.kubebuilder.io/v2` by default. A `v3-alpha` version
was created after `v2` stabilized.

You create a feature that adds a new marker to the file `main.go` scaffolded by `init`
that `create api` will use to update that file. The changes introduced in your feature
would cause errors if used with projects built with plugins `go.kubebuilder.io/v2`
without users manually updating their projects. Thus, your changes introduce a breaking change
to plugin `go.kubebuilder.io`, and can only be merged into plugin version `v3-alpha`.
This plugin's package should exist already, so a PR must be made against the

You must also add a migration guide to the [migrations](https://book.kubebuilder.io/migrations.html)
section of the KubeBuilder book in your PR. It should detail the steps required
for users to upgrade their projects from `v2` to `v3-alpha`.

## Community, discussion and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:
- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubebuilder)

## Becoming a reviewer or approver

Contributors may eventually become official reviewers or approvers in
KubeBuilder and the related repositories. See
[CONTRIBUTING-ROLES.md](docs/CONTRIBUTING-ROLES.md) for more information.

## Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[kb-releases]:https://github.com/kubernetes-sigs/kubebuilder/releases
[cli-plugins-versioning]:docs/book/src/reference/cli-plugins.md
