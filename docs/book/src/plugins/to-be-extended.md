## To help projects using Kubebuilder as Lib to composite new solutions and plugins

<aside class="note">

<h1>You can also create your own plugins, see:</h1>

- [Creating your own plugins][create-plugins].

</aside>

Then, see that you can use the kustomize plugin, which is responsible for to scaffold the kustomize files under `config/`, as
the base language plugins which are responsible for to scaffold the Golang files to create your own plugins to work with
another languages (i.e. [Operator-SDK][sdk] does to allow users work with Ansible/Helm) or to add
helpers on top, such as [Operator-SDK][sdk] does to add their features to integrate the projects with [OLM][olm].

| Plugin                                                                             | Key                         | Description                                                                                                                                     |
| ---------------------------------------------------------------------------------- |-----------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|
| [kustomize.common.kubebuilder.io/v2](kustomize-v2.md)                  | `kustomize/v2` | Responsible for scaffolding all [kustomize][kustomize] files under the `config/` directory                                                      |
| `base.go.kubebuilder.io/v4`                                 | `base/v4`      | Responsible for scaffolding all files which specifically requires Golang. This plugin is used in the composition to create the plugin (`go/v4`) |

[create-plugins]: creating-plugins.md
[kustomize]: https://kustomize.io/
[sdk]: https://github.com/operator-framework/operator-sdk
[olm]: https://olm.operatorframework.io/

