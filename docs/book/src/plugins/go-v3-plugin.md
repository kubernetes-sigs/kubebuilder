# [Deprecated] go/v3 (go.kubebuilder.io/v3)

<aside class="note warning">
<h1>Deprecated</h1>

The `go/v3` cannot fully support Kubernetes 1.25+ and work with Kustomize versions > v3.

The recommended way to migrate a `v3` project is to create a new `v4` project and copy over the API
and the reconciliation code. The conversion will end up with a project that looks like a native `v4` project.
For further information check the [Migration guide](../migration/migration_guide_gov3_to_gov4.md)

</aside>


Kubebuilder tool will scaffold the go/v3 plugin by default. This plugin is a composition of the plugins ` kustomize.common.kubebuilder.io/v1` and `base.go.kubebuilder.io/v3`. By using you can scaffold the default project which is a helper to construct sets of [controllers][controller-runtime]. 

It basically scaffolds all the boilerplate code required to create and design controllers. Note that by following the [quickstart][quickstart] you will be using this plugin. 

<aside class="note">

<h1>Examples</h1>

Samples are provided under the [testdata][testdata] directory of the Kubebuilder project. You can check samples using this plugin by looking at the `project-v3-<options>` projects under the [testdata][testdata] directory on the root directory of the Kubebuilder project.

</aside>

## When to use it ?

If you are looking to scaffold Golang projects to develop projects using [controllers][controller-runtime]

## How to use it ?

As `go/v3` is the default plugin there is no need to explicitly mention to Kubebuilder to use this plugin. 

To create a new project with the `go/v3` plugin the following command can be used:

```sh
kubebuilder init --plugins=`go/v3` --domain tutorial.kubebuilder.io --repo tutorial.kubebuilder.io/project
```

All the other subcommands supported by the go/v3 plugin can be executed similarly.

<aside class="note">

<h1>Note</h1>

Also, if you need you can explicitly inform the plugin via the option provided `--plugins=go/v3`.

</aside> 

## Subcommands supported by the plugin

-  Init -  `kubebuilder init [OPTIONS]`
-  Edit -  `kubebuilder edit [OPTIONS]`
-  Create API -  `kubebuilder create api [OPTIONS]`
-  Create Webhook - `kubebuilder create webhook [OPTIONS]`

## Further resources

- To check how plugins are composited by looking at this definition in the [main.go][plugins-main].
- Check the code implementation of the [base Golang plugin `base.go.kubebuilder.io/v3`][v3-plugin].
- Check the code implementation of the [Kustomize/v1 plugin][kustomize-plugin].
- Check [controller-runtime][controller-runtime] to know more about controllers.

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[quickstart]: ../quick-start.md
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata
[plugins-main]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/cmd/main.go
[v3-plugin]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/golang/v3
[kustomize-plugin]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/common/kustomize/v1