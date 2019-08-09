# What's in a basic project?

When scaffolding out a new project, Kubebuilder provides us with a few
basic pieces of boilerplate.

## Build Infrastructure

First up, basic infrastructure for building your project:

<details> <summary>`go.mod`: A new Go module matching our project, with
basic dependencies</summary>

```go
{{#include ./testdata/project/go.mod}}
```
</details>

<details><summary>`Makefile`: Make targets for building and deploying your controller</summary>

```makefile
{{#include ./testdata/project/Makefile}}
```
</details>

<details><summary>`PROJECT`: Kubebuilder metadata for scaffolding new components</summary>

```yaml
{{#include ./testdata/project/PROJECT}}
```
</details>

## Launch Configuration

We also get launch configurations under the
[`config/`](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project/config)
directory.  Right now, it just contains
[Kustomize](https://sigs.k8s.io/kustomize) YAML definitions required to
launch our controller on a cluster, but once we get started writing our
controller, it'll also hold our CustomResourceDefinitions, RBAC
configuration, and WebhookConfigurations.

[`config/default`](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project/config/default) contains a [Kustomize base](https://github.com/kubernetes-sigs/kubebuilder/blob/master/docs/book/src/cronjob-tutorial/testdata/project/config/default/kustomization.yaml) for launching
the controller in a standard configuration.

Each other directory contains a different piece of configuration,
refactored out into its own base:

- [`config/manager`](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project/config/manager): launch your controllers as pods in the
  cluster

- [`config/rbac`](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project/config/rbac): permissions required to run your
  controllers under their own service account

## The Entrypoint

Last, but certainly not least, Kubebuilder scaffolds out the basic
entrypoint of our project: `main.go`.  Let's take a look at that next...
