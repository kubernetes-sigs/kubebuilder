# Contributing guidelines

This document describes how to contribute to the project.

## Sign the CLA

Kubernetes projects require that you sign a Contributor License Agreement (CLA) before we can accept your pull requests.

Please see https://git.k8s.io/community/CLA.md for more info.

## Prerequisites

- [go](https://golang.org/dl/) version v1.23+.
- [docker](https://docs.docker.com/install/) version 17.03+.
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) version v1.11.3+.
- [kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/site/content/en/docs/Getting%20started/installation.md) v3.1.0+
- Access to a Kubernetes v1.11.3+ cluster.

## Contributing steps

1. Submit an issue describing your proposed change to the repo in question.
1. The [repo owners](OWNERS) will respond to your issue promptly.
1. If your proposed change is accepted, and you haven't already done so, sign a Contributor License Agreement (see details above).
1. Fork the desired repo, develop and test your code changes.
1. Submit a pull request.

In addition to the above steps, we adhere to the following best practices to maintain consistency and efficiency in our project:

- **Single Commit per PR:** Each Pull Request (PR) should contain only one commit. This approach simplifies tracking changes and makes the history more readable.
- **One Issue per PR:** Each PR should address a single specific issue or need. This helps in streamlining our workflow and makes it easier to identify and resolve problems such as revert the changes if required.

For more detailed guidelines, refer to the [Kubernetes Contributor Guide][k8s-contrubutiong-guide].

## How to build kubebuilder locally

Note that, by building the kubebuilder from the source code we are allowed to test the changes made locally.

1. Run the following command to clone your fork of the project locally in the dir /src/sigs.k8s.io/kubebuilder

```
$ git clone git@github.com:<user>/kubebuilder.git $GOPATH/src/sigs.k8s.io/kubebuilder
```

1. Ensure you activate module support before continue (`$ export GO111MODULE=on`)
1. Run the command `make install` to create a bin with the source code

**NOTE** In order to check the local environment run `make test-unit`.

## What to do before submitting a pull request

1. Run the script `make generate` to update/generate the mock data used in the e2e test in `$GOPATH/src/sigs.k8s.io/kubebuilder/testdata/`
1. Run `make test-unit test-e2e-local`

- e2e tests use [`kind`][kind] and [`setup-envtest`][setup-envtest]. If you want to bring your own binaries, place them in `$(go env GOPATH)/bin`.

**IMPORTANT:** The `make generate` is very helpful. By using it, you can check if good part of the commands still working successfully after the changes. Also, note that its usage is a prerequisite to submit a PR.

Following the targets that can be used to test your changes locally.

| Command             | Description                                                   | Is called in the CI? |
| ------------------- | ------------------------------------------------------------- | -------------------- |
| make test-unit      | Runs go tests                                                 | no                   |
| make test           | Runs tests in shell (`./test.sh`)                             | yes                  |
| make lint           | Run [golangci][golangci] lint checks                          | yes                  |
| make lint-fix       | Run [golangci][golangci] to automatically perform fixes       | no                   |
| make test-coverage  | Run coveralls to check the % of code covered by tests         | yes                  |
| make check-testdata | Checks if the testdata dir is updated with the latest changes | yes                  |
| make test-e2e-local | Runs the CI e2e tests locally                                 | no                   |

**NOTE** `make lint` requires a local installation of `golangci-lint`. More info: https://github.com/golangci/golangci-lint#install

### Running e2e tests locally

See that you can run `test-e2e-local` to setup Kind and run e2e tests locally.
Another option is by manually starting up Kind and configuring it and then,
you can for example via your IDEA debug the e2e tests.

To manually setup run:

```shell
# To generate an Kubebuilder local binary with your changes
make install
# To create the cluster and configure a CNI which supports NetworkPolicy
kind create cluster --config ./test/e2e/kind-config.yaml
kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
```

Now, you can for example, run in debug mode the `test/e2e/v4/e2e_suite_test.go`:

![example](https://github.com/kubernetes-sigs/kubebuilder/assets/7708031/277d26d5-c94d-41f0-8f02-1381458ef750)

### Test Plugin

If your intended PR creates a new plugin, make sure the PR also provides test cases. Testing should include:

1. `e2e tests` to validate the behavior of the proposed plugin.
2. `sample projects` to verify the scaffolded output from the plugin.

#### 1. Plugin E2E Tests

All the plugins provided by Kubebuilder should be validated through `e2e-tests` across multiple platforms.

Current Kubebuilder provides the testing framework that includes testing code based on [ginkgo](https://github.com/onsi/ginkgo), [Github Actions](https://github.com/Kavinjsir/kubebuilder/blob/docs%2Ftest-plugin/.github/workflows/testdata.yml) for unit tests, and multiple env tests driven by [test-infra](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/kubebuilder/kubebuilder-presubmits.yaml).

To fully test the proposed plugin:

1. Create a new package(folder) under `test/e2e/<your-plugin>`.
2. Create [e2e_suite_test.go](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/v4/e2e_suite_test.go), which imports the necessary testing framework.
3. Create `generate_test.go` ([ref](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/v4/generate_test.go)). That should:
   - Introduce/Receive a `TextContext` instance
   - Trigger the plugin's bound subcommands. See [Init](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/utils/test_context.go#L213), [CreateAPI](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.6.0/test/e2e/utils/test_context.go#L222)
   - Use [PluginUtil](https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin/util) to verify the scaffolded outputs. See [InsertCode](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/pkg/plugin/util/util.go#L67), [ReplaceInFile](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.6.0/pkg/plugin/util/util.go#L196), [UncommendCode](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.6.0/pkg/plugin/util/util.go#L86)
4. Create `plugin_cluster_test.go` ([ref](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/v4/plugin_cluster_test.go)). That should:

   - 4.1. Setup testing environment, e.g:

     - Cleanup environment, create temp dir. See [Prepare](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/utils/test_context.go#L97)
     - If your test will cover the provided features then, ensure that you install prerequisites CRDs: See [InstallCertManager](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/utils/test_context.go#L138), [InstallPrometheusManager](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.6.0/test/e2e/utils/test_context.go#L171)

   - 4.2. Run the function from `generate_test.go`.

   - 4.3. Further make sure the scaffolded output works, e.g:

     - Execute commands in your `Makefile`. See [Make](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/utils/test_context.go#L240)
     - Temporary load image of the testing controller. See [LoadImageToKindCluster](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/utils/test_context.go#L283)
     - Call Kubectl to validate running resources. See [utils.Kubectl](https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/test/e2e/utils#Kubectl)

   - 4.4. Delete temporary resources after testing exited, e.g:
     - Uninstall prerequisites CRDs: See [UninstallPrometheusOperManager](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/utils/test_context.go#L183)
     - Delete temp dir. See [Destroy](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/utils/test_context.go#L255)

5. Add the command in [test/e2e/plugin](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/e2e/setup.sh#L65) to run your testing code:

```shell
go test $(dirname "$0")/<your-plugin-test-folder> $flags -timeout 30m
```

#### 2. Sample Projects from the Plugin

It is also necessary to test consistency of the proposed plugin across different env and the integration with other plugins.

This is performed by generating sample projects based on the plugins. The CI workflow defined in Github Action would validate the availability and the consistency.

See:

- [test/testdata/generated.sh](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/test/testdata/generate.sh#L144)
- [make generate](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/Makefile#L70)

## PR Process

See [VERSIONING.md](VERSIONING.md) for a full description. TL;DR:

Every PR should be annotated with an icon indicating whether it's
a:

- Breaking change: :warning: (`:warning:`)
- Non-breaking feature: :sparkles: (`:sparkles:`)
- Patch fix: :bug: (`:bug:`)
- Docs: :book: (`:book:`)
- Infra/Tests/Other: :seedling: (`:seedling:`)
- No release note: :ghost: (`:ghost:`)

Use :ghost: (no release note) only for the PRs that change or revert unreleased
changes, which don't deserve a release note. Please don't abuse it.

You can also use the equivalent emoji directly, since GitHub doesn't
render the `:xyz:` aliases in PR titles.

If the PR is "plugin" scoped, you may also append the responding plugin names in the prefix.
[For instance](https://github.com/kubernetes-sigs/kubebuilder/commit/0b36d0c4021bbf52f29d5a990157466761ec180c):

```
🐛 (kustomize/v2-alpha): Fix typo issue in the labels added to the manifests
```

Individual commits should not be tagged separately, but will generally be
assumed to match the PR. For instance, if you have a bugfix in with
a breaking change, it's generally encouraged to submit the bugfix
separately, but if you must put them in one PR, mark the commit
separately.

## Where the CI Tests are configured

1. See the [action files](.github/workflows) to check its tests, and the scripts used on it.
2. Note that the prow tests used in the CI are configured in [kubernetes-sigs/kubebuilder/kubebuilder-presubmits.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/kubebuilder/kubebuilder-presubmits.yaml).
3. Check that all scripts used by the CI are defined in the project.
4. Notice that our policy to test the project is to run against k8s version N-2. So that the old version should be removed when there is a new k8s version available.

## How to contribute to docs

The docs are published off of three branches:

- `book-v4`: [book.kubebuilder.io](https://book.kubebuilder.io) -- current docs
- `book-v3`: [book-v3.book.kubebuilder.io](https://book-v3.book.kubebuilder.io) -- legacy docs
- `book-v2`: [book-v2.book.kubebuilder.io](https://book-v2.book.kubebuilder.io) -- legacy docs
- `book-v1`: [book-v1.book.kubebuilder.io](https://book-v1.book.kubebuilder.io) -- legacy docs
- `master`: [master.book.kubebuilder.io](https://master.book.kubebuilder.io) -- "nightly" docs

See [VERSIONING.md](VERSIONING.md#book-releases) for more information.

There are certain writing style guidelines for Kubernetes documentation, checkout [style guide](https://kubernetes.io/docs/contribute/style/style-guide/) for more information.

### How to preview the changes performed in the docs

Check the CI job after to do the Pull Request and then, click on in the `Details` of `netlify/kubebuilder/deploy-preview`

## Community, discussion and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubebuilder)

## Becoming a reviewer or approver

Contributors may eventually become official reviewers or approvers in
Kubebuilder and the related repositories. See
[CONTRIBUTING-ROLES.md](docs/CONTRIBUTING-ROLES.md) for more information.

## Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[golangci]: https://github.com/golangci/golangci-lint
[kind]: https://kind.sigs.k8s.io/#installation-and-usage
[setup-envtest]: https://book.kubebuilder.io/reference/envtest
[k8s-contrubutiong-guide]: https://www.kubernetes.dev/docs/guide/contributing/
