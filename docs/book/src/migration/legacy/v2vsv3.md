# Kubebuilder v2 vs v3 (Legacy Kubebuilder v2.0.0+ layout to 3.0.0+)

This document covers all breaking changes when migrating from v2 to v3.

The details of all changes (breaking or otherwise) can be found in
[controller-runtime][controller-runtime],
[controller-tools][controller-tools]
and [kb-releases][kb-releases] release notes.

## Common changes

v3 projects use Go modules and request Go 1.18+. Dep is no longer supported for dependency management.

## Kubebuilder

- Preliminary support for plugins was added. For more info see the [Extensible CLI and Scaffolding Plugins: phase 1][plugins-phase1-design-doc],
  the [Extensible CLI and Scaffolding Plugins: phase 1.5][plugins-phase1-design-doc-1.5] and the [Extensible CLI and Scaffolding Plugins - Phase 2][plugins-phase2-design-doc]
  design docs. Also, you can check the [Plugins section][plugins-section].

- The `PROJECT` file now has a new layout.  It stores more information about what resources are in use, to better enable plugins to make useful decisions when scaffolding.

    Furthermore, the PROJECT file itself is now versioned: the `version` field corresponds to the version of the PROJECT file itself, while the `layout` field indicates the scaffolding & primary plugin version in use.

- The version of the image `gcr.io/kubebuilder/kube-rbac-proxy`, which is an optional component enabled by default to secure the request made against the manager, was updated from `0.5.0` to `0.11.0` to address security concerns. The details of all changes can be found in [kube-rbac-proxy][kube-rbac-proxy].

## TL;DR of the New `go/v3` Plugin

***More details on this can be found at [here][kb-releases], but for the highlights, check below***

<aside class="note">
<h1>Default plugin</h1>
Projects scaffolded with Kubebuilder v3 will use the `go.kubebuilder.io/v3` plugin by default.
</aside>

- Scaffolded/Generated API version changes:
  * Use `apiextensions/v1` for generated CRDs (`apiextensions/v1beta1` was deprecated in Kubernetes `1.16`)
  * Use `admissionregistration.k8s.io/v1` for generated webhooks (`admissionregistration.k8s.io/v1beta1` was deprecated in Kubernetes `1.16`)
  * Use `cert-manager.io/v1` for the certificate manager when webhooks are used (`cert-manager.io/v1alpha2` was deprecated in `Cert-Manager 0.14`. More info: [CertManager v1.0 docs][cert-manager-docs])

- Code changes:
  * The manager flags `--metrics-addr` and `enable-leader-election` now are named `--metrics-bind-address` and `--leader-elect` to be more aligned with core Kubernetes Components. More info: [#1839][issue-1893]
  * Liveness and Readiness probes are now added by default using [`healthz.Ping`][healthz-ping].
  * A new option to create the projects using ComponentConfig is introduced. For more info see its [enhancement proposal][enhancement proposal] and the [Component config tutorial][component-config-tutorial]
  * Manager manifests now use `SecurityContext` to address security concerns. More info: [#1637][issue-1637]
- Misc:
  * Support for [controller-tools][controller-tools] `v0.9.0` (for `go/v2` it is `v0.3.0` and previously it was `v0.2.5`)
  * Support for [controller-runtime][controller-runtime] `v0.12.1` (for `go/v2` it is `v0.6.4` and previously it was `v0.5.0`)
  * Support for [kustomize][kustomize] `v3.8.7` (for `go/v2` it is `v3.5.4` and previously it was `v3.1.0`)
  * Required Envtest binaries are automatically downloaded
  * The minimum Go version is now `1.18` (previously it was `1.13`).

<aside class="note warning">
<h1>Project customizations</h1>

After using the CLI to create your project, you are free to customise how you see fit. Bear in mind, that it is not recommended to deviate from the proposed layout unless you know what you are doing.

For example, you should refrain from moving the scaffolded files, doing so will make it difficult in upgrading your project in the future. You may also lose the ability to use some of the CLI features and helpers. For further information on the project layout, see the doc [What's in a basic project?][basic-project-doc]

</aside>

## Migrating to Kubebuilder v3

So you want to upgrade your scaffolding to use the latest and greatest features then, follow up the following guide which will cover the steps in the most straightforward way to allow you to upgrade your project to get all latest changes and improvements.

<aside class="note warning">
<h1> Apple Silicon (M1) </h1>

The current scaffold done by the CLI (`go/v3`) uses [kubernetes-sigs/kustomize][kustomize] v3 which does not provide
a valid binary for Apple Silicon (`darwin/arm64`). Therefore, you can use the `go/v4` plugin
instead which provides support for this platform:

```bash
kubebuilder init --domain my.domain --repo my.domain/guestbook --plugins=go/v4
```

</aside>

- [Migration Guide v2 to V3][migration-guide-v2-to-v3] **(Recommended)**

### By updating the files manually

So you want to use the latest version of Kubebuilder CLI without changing your scaffolding then, check the following guide which will describe the manually steps required for you to upgrade only your PROJECT version and starts to use the plugins versions.

This way is more complex, susceptible to errors, and success cannot be assured. Also, by following these steps you will not get the improvements and bug fixes in the default generated project files.

You will check that you can still using the previous layout by using the `go/v2` plugin which will not upgrade the [controller-runtime][controller-runtime] and [controller-tools][controller-tools] to the latest version used with `go/v3` becuase of its breaking changes. By checking this guide you know also how to manually change the files to use the `go/v3` plugin and its dependencies versions.

- [Migrating to Kubebuilder v3 by updating the files manually][manually-upgrade]

[plugins-phase1-design-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md
[plugins-phase1-design-doc-1.5]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1-5.md
[plugins-phase2-design-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-2.md
[plugins-section]: ./../../plugins/plugins.md
[manually-upgrade]: manually_migration_guide_v2_v3.md
[component-config-tutorial]: ../../component-config-tutorial/tutorial.md
[issue-1893]: https://github.com/kubernetes-sigs/kubebuilder/issues/1839
[migration-guide-v2-to-v3]: migration_guide_v2tov3.md
[healthz-ping]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/healthz#CheckHandler
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime/releases
[controller-tools]: https://github.com/kubernetes-sigs/controller-tools/releases
[kustomize]: https://github.com/kubernetes-sigs/kustomize/releases
[issue-1637]: https://github.com/kubernetes-sigs/kubebuilder/issues/1637
[enhancement proposal]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/wgs
[cert-manager-docs]: https://cert-manager.io/docs/installation/upgrading/
[kb-releases]: https://github.com/kubernetes-sigs/kubebuilder/releases
[kube-rbac-proxy]: https://github.com/brancz/kube-rbac-proxy/releases
[basic-project-doc]: ../../cronjob-tutorial/basic-project.md
[kustomize]: https://github.com/kubernetes-sigs/kustomize
