# go/v4-alpha (go.kubebuilder.io/v4-alpha)

Kubebuilder will scaffold using the go/v4-alpha plugin only if specified when initializing the project. 
This plugin is a composition of the plugins ` kustomize.common.kubebuilder.io/v2-alpha` and `base.go.kubebuilder.io/v4`. 
It scaffolds a project template that helps in constructing sets of [controllers][controller-runtime]. 

It scaffolds boilerplate code to create and design controllers. 
Note that by following the [quickstart][quickstart] you will be using this plugin.
<aside class="note">

<h1>Examples</h1>

You can check samples using this plugin by looking at the `project-v4-<options>` projects 
under the [testdata][testdata] directory on the root directory of the Kubebuilder project.

</aside>

## When to use it ?

- If you are looking to scaffold Golang projects to develop projects using [controllers][controller-runtime]
- If you are looking to experiment with the future default scaffold that will be provided by Kubebuilder CLI
- If your local environment is Apple Silicon (`darwin/arm64`)
- If you are looking to use [kubernetes-sigs/kustomize][kustomize] v4
- If you are looking for the new default layout where the "apis" and "controllers" are scaffold under the `pkg/` dir.

## How to use it ?

To create a new project with the `go/v4-alpha` plugin the following command can be used:

```sh
kubebuilder init --domain tutorial.kubebuilder.io --repo tutorial.kubebuilder.io/project --plugins=go/v4-alpha
```

## Subcommands supported by the plugin

-  Init -  `kubebuilder init [OPTIONS]`
-  Edit -  `kubebuilder edit [OPTIONS]`
-  Create API -  `kubebuilder create api [OPTIONS]`
-  Create Webhook - `kubebuilder create webhook [OPTIONS]`

## Further resources

- To see the composition of plugins, you can check the source code for the Kubebuilder [main.go][plugins-main].
- Check the code implementation of the [base Golang plugin `base.go.kubebuilder.io/v3`][v3-plugin].
- Check the code implementation of the [Kustomize/v2-alpha plugin][kustomize-plugin].
- Check [controller-runtime][controller-runtime] to know more about controllers.

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[quickstart]: ../quick-start.md
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata
[plugins-main]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/cmd/main.go
[kustomize-plugin]: ../plugins/kustomize-v2-alpha.md
[kustomize]: https://github.com/kubernetes-sigs/kustomize