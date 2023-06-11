## To help projects using Kubebuilder as Lib to composite new solutions and plugins

<aside class="note">

<h1>You can also create your own plugins, see:</h1>

- [Creating your own plugins][create-plugins].

</aside>

Then, see that you can use the kustomize plugin, which is responsible for to scaffold the kustomize files under `config/`, as
the base language plugins which are responsible for to scaffold the Golang files to create your own plugins to work with
another languages (i.e. [Operator-SDK][sdk] does to allow users work with Ansible/Helm) or to add
helpers on top, such as [Operator-SDK][sdk] does to add their features to integrate the projects with [OLM][olm].

| Plugin                                                                             | Key                         | Description                                                                                                                                                                                                                                  |
| ---------------------------------------------------------------------------------- |-----------------------------| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [kustomize.common.kubebuilder.io/v1](https://github.com/kubernetes-sigs/kubebuilder/pull/3235/kustomize-v1.md) | kustomize/v1 (Deprecated) | Responsible for scaffolding all manifests to configure projects with [kustomize(v3)][kustomize]. (create and update the `config/` directory). This plugin is used in the composition to create the plugin (`go/v3`). |
| [kustomize.common.kubebuilder.io/v2](kustomize-v2.md)                  | `kustomize/v2` | It has the same purpose of `kustomize/v1`. However, it works with [kustomize][kustomize] version `v4` and addresses the required changes for future kustomize configurations. It will probably be used with the future `go/v4-alpha` plugin. |
| `base.go.kubebuilder.io/v3`                                                        | `base/v3`                   | Responsible for scaffolding all files that specifically require Golang. This plugin is used in composition to create the plugin (`go/v3`)                                                                                                     |
| `base.go.kubebuilder.io/v4`                                 | `base/v4`      | Responsible for scaffolding all files which specifically requires Golang. This plugin is used in the composition to create the plugin (`go/v4`)                                                                                     |

[create-plugins]: creating-plugins.md
[kubebuilder-declarative-pattern]: https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern
[kustomize]: https://kustomize.io/
[sdk]: https://github.com/operator-framework/operator-sdk
[olm]: https://olm.operatorframework.io/

