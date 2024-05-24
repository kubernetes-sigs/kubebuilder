# go/v3 vs go/v4

This document covers all breaking changes when migrating from projects built using the plugin go/v3 (default for any scaffold done since `28 Apr 2021`) to the next version of the Golang plugin `go/v4`.

The details of all changes (breaking or otherwise) can be found in:

- [controller-runtime][controller-runtime]
- [controller-tools][controller-tools]
- [kustomize][kustomize-release]
- [kb-releases][kb-releases] release notes.

## Common changes

- `go/v4` projects use Kustomize v5x (instead of v3x)
- note that some manifests under `config/` directory have been changed in order to no longer use the deprecated Kustomize features
  such as env vars.
- A `kustomization.yaml` is scaffolded under `config/samples`. This helps simply and flexibly generate sample manifests: `kustomize build config/samples`.
- adds support for Apple Silicon M1 (darwin/arm64)
- remove support to CRD/WebHooks Kubernetes API v1beta1 version which are no longer supported since k8s 1.22
- no longer scaffold webhook test files with `"k8s.io/api/admission/v1beta1"` the k8s API which is no longer served since k8s `1.25`. By default
  webhooks test files are scaffolding using `"k8s.io/api/admission/v1"` which is support from k8s `1.20`
- no longer provide backwards compatible support with k8s versions < `1.16`
- change the layout to accommodate the community request to follow the [Standard Go Project Layout][standard-go-project]
by moving the api(s) under a new directory called `api`, controller(s) under a new directory called `internal` and the `main.go` under a new directory named `cmd`

<aside class="note">
<H1> TL;DR of the New `go/v4` Plugin </H1>

Further details can be found in the [go/v4 plugin section][go/v4-doc]

</aside>

## TL;DR of the New `go/v4` Plugin

**_More details on this can be found at [here][kb-releases], but for the highlights, check below_**

<aside class="note warning">
<h1>Project customizations</h1>

After using the CLI to create your project, you are free to customize how you see fit. Bear in mind, that it is not recommended to deviate from the proposed layout unless you know what you are doing.

For example, you should refrain from moving the scaffolded files, doing so will make it difficult in upgrading your project in the future. You may also lose the ability to use some of the CLI features and helpers. For further information on the project layout, see the doc [What's in a basic project?][basic-project-doc]

</aside>

## Migrating to Kubebuilder go/v4

If you want to upgrade your scaffolding to use the latest and greatest features then, follow the guide
which will cover the steps in the most straightforward way to allow you to upgrade your project to get all
latest changes and improvements.

- [Migration Guide go/v3 to go/v4][migration-guide-gov3-to-gov4] **(Recommended)**

### By updating the files manually

If you want to use the latest version of Kubebuilder CLI without changing your scaffolding then, check the following guide which will describe the steps to be performed manually to upgrade only your PROJECT version and start using the plugins versions.

This way is more complex, susceptible to errors, and success cannot be assured. Also, by following these steps you will not get the improvements and bug fixes in the default generated project files.

- [Migrating to go/v4 by updating the files manually][manually-upgrade]

[plugins-phase1-design-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md
[plugins-phase1-design-doc-1.5]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1-5.md
[plugins-phase2-design-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-2.md
[plugins-section]: ./../plugins/plugins.md
[kustomize]: https://github.com/kubernetes-sigs/kustomize/releases/tag/kustomize%2Fv4.0.0
[go/v4-doc]: ./../plugins/go-v4-plugin.md
[migration-guide-gov3-to-gov4]: migration_guide_gov3_to_gov4.md
[manually-upgrade]: manually_migration_guide_gov3_to_gov4.md
[basic-project-doc]: ./../cronjob-tutorial/basic-project.md
[standard-go-project]: https://github.com/golang-standards/project-layout
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[controller-tools]: https://github.com/kubernetes-sigs/controller-tools
[kustomize-release]: https://github.com/kubernetes-sigs/kustomize/releases/tag/kustomize%2Fv5.0.0
[kb-releases]: https://github.com/kubernetes-sigs/kubebuilder/releases
