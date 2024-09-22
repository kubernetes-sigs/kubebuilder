## To be extended

The following plugins are useful for other tools and [External Plugins][external-plugins] which are looking to extend the
Kubebuilder functionality.

You can use the kustomize plugin, which is responsible for scaffolding the
kustomize files under `config/`. The base language plugins are responsible
for scaffolding the necessary Golang files, allowing you to create your
own plugins for other languages (e.g., [Operator-SDK][sdk] enables
users to work with Ansible/Helm) or add additional functionality.

For example, [Operator-SDK][sdk] has a plugin which integrates the
projects with [OLM][olm] by adding its own features on top.

| Plugin                                                 | Key                         | Description                                                                                                                                     |
|--------------------------------------------------------|-----------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|
| [kustomize.common.kubebuilder.io/v2][kustomize-plugin] | `kustomize/v2` | Responsible for scaffolding all [kustomize][kustomize] files under the `config/` directory                                                      |
| `base.go.kubebuilder.io/v4`                            | `base/v4`      | Responsible for scaffolding all files which specifically requires Golang. This plugin is used in the composition to create the plugin (`go/v4`) |

[kustomize]: https://kustomize.io/
[sdk]: https://github.com/operator-framework/operator-sdk
[olm]: https://olm.operatorframework.io/
[kustomize-plugin]: ./available/kustomize-v2.md
[external-plugins]: ./extending/external-plugins.md
