# Extending Kubebuilder

Kubebuilder provides an extensible architecture to scaffold
projects using plugins. These plugins allow you to customize the CLI
behavior or integrate new features.

## Overview

Kubebuilder’s CLI can be extended through custom plugins, allowing you to:

- Build new scaffolds.
- Enhance existing ones.
- Add new commands and functionality to Kubebuilder’s scaffolding.

This flexibility enables you to create custom project
setups tailored to specific needs.

<aside class="note">
<h1>Why use the Kubebuilder style?</h1>

Kubebuilder and SDK are both broadly adopted projects which leverage the [controller-runtime][controller-runtime] project. They both allow users to build solutions using the [Operator Pattern][operator-pattern] and follow common standards.

Adopting these standards can bring significant benefits, such as joining forces on maintaining the common standards as the features provided by Kubebuilder and take advantage of the contributions made by the community. This allows you to focus on the specific needs and requirements for your plugin and use-case.

And then, you will also be able to use custom plugins and options currently or in the future which might to be provided by these projects as any other which decides to persuade the same standards.

</aside>

## Options to Extend

Extending Kubebuilder can be achieved in two main ways:

1. **Extending CLI features and Plugins**:
   You can import and build upon existing Kubebuilder plugins to [extend
   its features and plugins][extending-cli]. This is useful when you need to add specific
   features to a tool that already benefits from Kubebuilder's scaffolding system.
   For example, [Operator SDK][sdk] leverages the [kustomize plugin][kustomize-plugin]
   to provide language support for tools like Ansible or Helm. So that the project
   can be focused to keep maintained only what is specific language based.

2. **Creating External Plugins**:
   You can build standalone, independent plugins as binaries. These plugins can be written in any
   language and should follow an execution pattern that Kubebuilder recognizes. For more information,
   see [Creating external plugins][external-plugins].

For further details on how to extend Kubebuilder, explore the following sections:

- [CLI and Plugins](./extending/extending_cli_features_and_plugins.md) to learn how to extend CLI features and plugins.
- [External Plugins](./extending/external-plugins.md) for creating standalone plugins.
- [E2E Tests](./extending/testing-plugins.md) to ensure your plugin functions as expected.

[extending-cli]: ./extending/extending_cli_features_and_plugins.md
[external-plugins]: ./extending/external-plugins.md
[sdk]: https://github.com/operator-framework/operator-sdk
[kustomize-plugin]: ./available/kustomize-v2.md
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/