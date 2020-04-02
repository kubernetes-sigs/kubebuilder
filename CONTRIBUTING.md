# Contributing guidelines

This document describes how to contribute to the project.

## Sign the CLA

Kubernetes projects require that you sign a Contributor License Agreement (CLA) before we can accept your pull requests.

Please see https://git.k8s.io/community/CLA.md for more info.

## Prerequisites

- [go](https://golang.org/dl/) version v1.13+.
- [dep](https://github.com/golang/dep) dep v0.5+
- [docker](https://docs.docker.com/install/) version 17.03+.
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) version v1.11.3+.
- [kustomize](https://sigs.k8s.io/kustomize/docs/INSTALL.md) v3.1.0+
- Access to a Kubernetes v1.14.1+ cluster.

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

