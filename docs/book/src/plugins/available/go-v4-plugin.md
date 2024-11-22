# go/v4 (go.kubebuilder.io/v4)

**(Default Scaffold)**

Kubebuilder will scaffold using the `go/v4` plugin only if specified when initializing the project.
This plugin is a composition of the `kustomize.common.kubebuilder.io/v2` and `base.go.kubebuilder.io/v4` plugins
using the [Bundle Plugin][bundle]. It scaffolds a project template
that helps in constructing sets of [controllers][controller-runtime].

By following the [quickstart][quickstart] and creating any project,
you will be using this plugin by default.

<aside class="note">
<h1>Examples</h1>

You can check samples using this plugin by looking at the `project-v4-<options>` projects under the [testdata][testdata]
directory on the root directory of the Kubebuilder project.

</aside>

## How to use it ?

To create a new project with the `go/v4` plugin the following command can be used:

```sh
kubebuilder init --domain tutorial.kubebuilder.io --repo tutorial.kubebuilder.io/project --plugins=go/v4
```

## Subcommands supported by the plugin

-  Init -  `kubebuilder init [OPTIONS]`
-  Edit -  `kubebuilder edit [OPTIONS]`
-  Create API -  `kubebuilder create api [OPTIONS]`
-  Create Webhook - `kubebuilder create webhook [OPTIONS]`

## Further resources

- To see the composition of plugins, you can check the source code for the Kubebuilder [main.go][plugins-main].
- Check the code implementation of the [base Golang plugin `base.go.kubebuilder.io/v4`][v4-plugin].
- Check the code implementation of the [Kustomize/v2 plugin][kustomize-plugin].
- Check [controller-runtime][controller-runtime] to know more about controllers.

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[quickstart]: ./../../quick-start.md
[testdata]: ./../../../../../testdata
[plugins-main]: ./../../../../../cmd/main.go
[kustomize-plugin]: ./../../plugins/available/kustomize-v2.md
[kustomize]: https://github.com/kubernetes-sigs/kustomize
[standard-go-project]: https://github.com/golang-standards/project-layout
[v4-plugin]: ./../../../../../pkg/plugins/golang/v4
[migration-guide-doc]: ./../../migration/migration_guide_gov3_to_gov4.md
[project-doc]: ./../../reference/project-config.md
[bundle]: ./../../../../../pkg/plugin/bundle.go
