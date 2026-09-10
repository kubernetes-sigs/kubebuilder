# Examples of External Plugins

This page tracks examples of external plugins built by the community that integrate with
Kubebuilder. These plugins extend or complement Kubebuilder's scaffolding
engine and are maintained independently by their respective authors.

<aside class="note" role="note">
<p class="note-title">Note</p>

These plugins are community-maintained and are not officially
supported by the Kubebuilder project. Please refer to each plugin's
repository for documentation, support, and compatibility information.
</aside>

---

## What Counts as a Kubebuilder Plugin?

A Kubebuilder **plugin** implements the
[`plugin.Plugin`][plugin-interface] interface and integrates with the
Kubebuilder CLI via the [External Plugins][external-plugins] mechanism.
This allows it to be invoked as a sub-step during `kubebuilder init`,
`kubebuilder create api`, etc.

Projects that are *built on top of* Kubebuilder but do not plug into its CLI
(e.g. operators that use controller-runtime directly) are not listed here.

---

## Community Plugins

| Plugin | Language | Description |
|--------|----------|-------------|
| [rust-operator-plugins][rust-plugin] | Rust | Scaffolds Rust-based Kubernetes operators using the Kubebuilder plugin interface |
| [kubebuilder-initializer-plugin][initializer-plugin] | Go | Adds opinionated project initialization steps on top of the default Kubebuilder scaffolding |
| [operator-builder][operator-builder] | Go | A Kubebuilder plugin to accelerate the development of Kubernetes Operators |

---

## Experimental / Proof-of-Concept Plugins

The following repositories demonstrate the Kubebuilder plugin interface and
may serve as useful references when building your own plugin, but are not
actively maintained for production use.

| Plugin | Language | Description |
|--------|----------|-------------|
| [kb-js-plugin][js-plugin] | JavaScript | PoC for scaffolding JavaScript-based operators |
| [POC-Phase2-Plugins][poc-phase2] | Go | Early proof-of-concept for the Kubebuilder plugin system |
| [plugin-testing-poc][plugin-testing-poc] | Go | PoC for plugin testing infrastructure |

---

[plugin-interface]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin#Plugin
[external-plugins]: ./external-plugins.md
[rust-plugin]: https://github.com/SystemCraftsman/rust-operator-plugins
[initializer-plugin]: https://github.com/astrokube/kubebuilder-initializer-plugin
[operator-builder]: https://github.com/vmware-tanzu-labs/operator-builder
[js-plugin]: https://github.com/Eileen-Yu/kb-js-plugin
[poc-phase2]: https://github.com/rashmigottipati/POC-Phase2-Plugins
[plugin-testing-poc]: https://github.com/everettraven/plugin-testing-poc
