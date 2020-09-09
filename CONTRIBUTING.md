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
| make lint |  Run [golangci][golangci] lint checks | yes   |
| make lint-fix |   Run [golangci][golangci] to automatically perform fixes | no   |
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

## Understanding the versions

|   Name	|   Example	|  Description |
|---	|---	|---	|
|  PROJECT version |  `v1`,`v2`,`v3` | As of the introduction of [Extensible CLI and Scaffolding Plugins](https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md), PROJECT version represents the layout of PROJECT file itself.  For `v1` and `v2` projects, there's extra meaning -- see below.  |
|  Release version | `v2.2.0`, `v2.3.0`, `v2.3.1` | Tagged versions of the KubeBuilder project, representing changes to the source code in this repository. See the [releases](https://github.com/kubernetes-sigs/kubebuilder/releases) page. |
|  Plugin Versions | `go.kubebuilder.io/v2.0.0` | Represents the version of an individual plugin, as well as the corresponding scaffolding that it generates. |

Note that PROJECT version should only be bumped if a breaking change is introduced in the PROJECT file format itself.  Changes to the Go scaffolding or the KubeBuilder CLI *do not* affect the PROJECT version.

Similarly, the introduction of a new major version (`(x+1).0.0`) of the Go plugin might only lead to a new minor (`x.(y+1).0`) release of KubeBuilder, since no breaking change is being made to the CLI itself.  It'd only be a breaking change to KubeBuilder if we remove support for an older version of the plugin.

For more information on how the release and plugin versions work, see the the [semver](https://semver.org/) documentation.

**NOTE:** In the case of the `v1` and `v2` PROJECT version, a corresponding Plugin version is implied and constant -- `v1` implies the `go.kubebuilder.io/v1` Plugin, and similarly for `v2`.  This is for legacy purposes -- no such implication is made with the `v3` PROJECT version.

## Introducing changes in the scaffold files

Changes in the scaffolded files require a new Plugin version. If we delete or update a file that is scaffolded by default, it's a breaking change and requires a `MAJOR` Plugin version.  If we add a new file, it may not be a breaking change.

**More simply:** any change that will break the expected behaviour of a project built with the previous `MINOR` Plugin versions is a breaking change to that plugin. 

**EXAMPLE:**

KubeBuilder Release version (`5.3.1`) scaffolds projects with the plugin version `3.2.1` by default.

The changes introduced in our PR will not work well with the projects which were built with the plugin versions `3.0.0...3.5.1` without users taking manual steps to update their projects. Thus, our changes introduce a breaking change to the Go plugin, and require a `MAJOR` Plugin version bump.

In the PR, we should add a migration guide to the [Migrations](https://book.kubebuilder.io/migrations.html) section of the KubeBuilder book. It should detail the required steps that users should take to upgrade their projects from `go.kubebuilder.io/3.X.X` to the new `MAJOR` Plugin version `go.kubebuilder.io/4.0.0`.

This also means we should introduce a new KubeBuilder minor version `5.4.0` when the project is released: since we've only added a new plugin version without removing the old one, this is considered a new feature to KubeBuilder, and not a breaking change.

**IMPORTANT** Breaking changes cannot be made to PROJECT versions v1 and v2, and consequently plugin versions `go.kubebuilder.io/1` and `go.kubebuilder.io/2`.

## Community, discussion and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:
- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubebuilder)

## Becoming a reviewer or approver

Contributors may eventually become official reviewers or approvers in
KubeBuilder and the related repositories.  See
[CONTRIBUTING-ROLES.md](docs/CONTRIBUTING-ROLES.md) for more information.

## Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[golangci]:https://github.com/golangci/golangci-lint