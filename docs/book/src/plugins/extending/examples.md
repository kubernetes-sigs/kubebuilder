# Examples of external plugins

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

## What counts as a Kubebuilder plugin?

A Kubebuilder **plugin** integrates with the Kubebuilder CLI either as an in-process Go plugin implementing the [`plugin.Plugin`][plugin-interface] interface, or as an external plugin executable using the [External Plugins][external-plugins] protocol.

This design allows projects to build solutions that generate code with the Kubebuilder CLI, provide new scaffolds, or customize existing scaffolding by chaining with default subcommands (such as `kubebuilder init` or `kubebuilder create api`). See the [plugins][plugins-page] page for details.

Note that Kubebuilder can also be used as a library by other tooling. For example, Kubebuilder provides core functionality for tools like Operator SDK. This is different from creating a plugin that integrates directly with the Kubebuilder CLI. Because of that distinction, projects that use Kubebuilder solely as a library are not listed here.

---

## Community plugins

| Plugin | Language | Description |
|--------|----------|-------------|
| [rust-operator-plugins][rust-plugin] | Rust | Scaffolds Rust-based Kubernetes operators using the Kubebuilder plugin interface |
| [kubebuilder-initializer-plugin][initializer-plugin] | Go | Adds opinionated project initialization steps on top of the default Kubebuilder scaffolding |
| [operator-builder][operator-builder] | Go | A Kubebuilder plugin to accelerate the development of Kubernetes Operators |

---

## Experimental / proof of concept plugins

The following repositories demonstrate the Kubebuilder plugin interface and
may serve as useful references when building your own plugin, but are not
actively maintained for production use.

| Plugin | Language | Description |
|--------|----------|-------------|
| [kb-js-plugin][js-plugin] | JavaScript | Proof of concept for scaffolding JavaScript-based operators |
| [POC-Phase2-Plugins][poc-phase2] | Go | Early proof of concept for the Kubebuilder plugin system |
| [plugin-testing-poc][plugin-testing-poc] | Go | Proof of concept for plugin testing infrastructure |

---

[plugin-interface]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugin#Plugin
[external-plugins]: ./external-plugins.md
[plugins-page]: ../plugins.md
[rust-plugin]: https://github.com/SystemCraftsman/rust-operator-plugins
[initializer-plugin]: https://github.com/astrokube/kubebuilder-initializer-plugin
[operator-builder]: https://github.com/nukleros/operator-builder
[js-plugin]: https://github.com/Eileen-Yu/kb-js-plugin
[poc-phase2]: https://github.com/rashmigottipati/POC-Phase2-Plugins
[plugin-testing-poc]: https://github.com/everettraven/plugin-testing-poc
