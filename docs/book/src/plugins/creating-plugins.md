# Creating your own plugins

[extending-cli]: extending-cli.md
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[creating-external-plugins]: external-plugins.md
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator
[sdk-ansible]: https://sdk.operatorframework.io/docs/building-operators/ansible/
[sdk-cli-pkg]: https://pkg.go.dev/github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/cli
[sdk-helm]: https://sdk.operatorframework.io/docs/building-operators/helm/
[sdk]: https://github.com/operator-framework/operator-sdk

<aside class="note warning">

<h1>Note</h1>

Extending Kubebuilder can be accomplished in two primary ways:

`By re-using the existing plugins`: In this approach, you use Kubebuilder as a library.

This enables you to import existing Kubebuilder plugins and extend them, leveraging their features to build upon.

It is particularly useful if you want to add functionalities that are closely tied with the existing Kubebuilder features.


`By Creating an External Plugin`: This method allows you to create an independent, standalone plugin as a binary.

The plugin can be written in any language and should implement an execution pattern that Kubebuilder knows how to interact with.

You can see [Creating external plugins][creating-external-plugins] for more info.

</aside>

## Overview

You can extend the Kubebuilder API to create your own plugins. If [extending the CLI][extending-cli], your plugin will be implemented in your project and registered to the CLI as has been done by the [SDK][sdk] project. See its [CLI code][sdk-cli-pkg] as an example.

## When it is useful?

- If you are looking to create plugins which support and work with another language.
- If you would like to create helpers and integrations on top of the scaffolds done by the plugins provided by Kubebuiler.
- If you would like to have customized layouts according to your needs.

## How the plugins can be used?

Kubebuilder provides a set of plugins to scaffold the projects, to help you extend and re-use its implementation to provide additional features.
For further information see [Available Plugins][available-plugins].

Therefore, if you have a need you might want to propose a solution by adding a new plugin
which would be shipped with Kubebuilder by default.

However, you might also want to have your own tool to address your specific scenarios and by taking advantage of what is provided by Kubebuilder as a library.
That way, you can focus on addressing your needs and keep your solutions easier to maintain.

Note that by using Kubebuilder as a library, you can import its plugins and then create your own plugins that do customizations on top.
For instance, `Operator-SDK` does with the plugins [manifest][operator-sdk-manifest] and [scorecard][operator-sdk-scorecard] to add its features.
Also see [here][operator-sdk-plugin-ref].

Another option implemented with the [Extensible CLI and Scaffolding Plugins - Phase 2][plugins-phase2-design-doc] is
to extend Kibebuilder as a LIB to create only a specific plugin that can be called and used with
Kubebuilder as well.

<aside class="note">
<H1> Plugins proposal docs</H1>

You can check the proposal documentation for better understanding its motivations. See the [Extensible CLI and Scaffolding Plugins: phase 1][plugins-phase1-design-doc],
the [Extensible CLI and Scaffolding Plugins: phase 1.5][plugins-phase1-design-doc-1.5] and the [Extensible CLI and Scaffolding Plugins - Phase 2][plugins-phase2-design-doc]
design docs. Also, you can check the [Plugins section][plugins-section].

</aside>

## Language-based Plugins

Kubebuilder offers the Golang-based operator plugins, which will help its CLI tool users create projects following the [Operator Pattern][operator-pattern].

The [SDK][sdk] project, for example, has language plugins for [Ansible][sdk-ansible] and [Helm][sdk-helm], which are similar options but for users who would like to work with these respective languages and stacks instead of Golang.

Note that Kubebuilder provides the `kustomize.common.kubebuilder.io` to help in these efforts. This plugin will scaffold the common base without any specific language scaffold file to allow you to extend the Kubebuilder style for your plugins.

In this way, currently, you can [Extend the CLI][extending-cli] and use the `Bundle Plugin` to create your language plugins such as:

```go
  mylanguagev1Bundle, _ := plugin.NewBundle(plugin.WithName(language.DefaultNameQualifier), 
    plugin.WithVersion(plugin.Version{Number: 1}),
		plugin.WithPlugins(kustomizecommonv1.Plugin{}, mylanguagev1.Plugin{}), // extend the common base from Kubebuilder
		// your plugin language which will do the scaffolds for the specific language on top of the common base
	)
```

If you do not want to develop your plugin using Golang, you can follow its standard by using the binary as follows:

```sh
kubebuilder init --plugins=kustomize
```

Then you can, for example, create your implementations for the sub-commands `create api` and `create webhook` using your language of preference.

<aside class="note">
<h1>Why use the Kubebuilder style?</h1>

Kubebuilder and SDK are both broadly adopted projects which leverage the [controller-runtime][controller-runtime] project. They both allow users to build solutions using the [Operator Pattern][operator-pattern] and follow common standards.

Adopting these standards can bring significant benefits, such as joining forces on maintaining the common standards as the features provided by Kubebuilder and take advantage of the contributions made by the community. This allows you to focus on the specific needs and requirements for your plugin and use-case.

And then, you will also be able to use custom plugins and options currently or in the future which might to be provided by these projects as any other which decides to persuade the same standards.

</aside>

## Custom Plugins

Note that users are also able to use plugins to customize their scaffolds and address specific needs.

See that Kubebuilder provides the [`deploy-image`][deploy-image] plugin that allows the user to create the controller & CRs which will deploy and manage an image on the cluster:

```sh
kubebuilder create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:1.6.15-alpine --image-container-command="memcached,-m=64,modern,-v" --image-container-port="11211" --run-as-user="1001" --plugins="deploy-image/v1-alpha"
```

This plugin will perform a custom scaffold following the [Operator Pattern][operator-pattern].

Another example is the [`grafana`][grafana] plugin that scaffolds a new folder container manifests to visualize operator status on Grafana Web UI:

```sh
kubebuilder edit --plugins="grafana.kubebuilder.io/v1-alpha"
```

In this way, by [Extending the Kubebuilder CLI][extending-cli], you can also create custom plugins such this one.

Feel free to check the implementation under:

- deploy-image: <https://github.com/kubernetes-sigs/kubebuilder/tree/v3.7.0/pkg/plugins/golang/deploy-image/v1alpha1>
- grafana: <https://github.com/kubernetes-sigs/kubebuilder/tree/v3.7.0/pkg/plugins/optional/grafana/v1alpha>

## Plugin Scaffolding

Your plugin may add code on top of what is scaffolded by default with Kubebuilder sub-commands(`init`, `create`, ...).
This is common as you may expect your plugin to:

- Create API
- Update controller manager logic
- Generate corresponding manifests

### Boilerplates

The Kubebuilder internal plugins use boilerplates to generate the files of code.

For instance, the go/v3 scaffolds the `main.go` file by defining an object that [implements the machinery interface][kubebuilder-machinery].
In the [implementation][go-v3-settemplatedefault] of `Template.SetTemplateDefaults`, the [raw template][go-v3-main-template] is set to the body.
Such object that implements the machinery interface will later pass to the [execution of scaffold][go-v3-scaffold-execute].

Similar, you may also design your code of plugin implementation by such reference.
You can also view the other parts of the code file given by the links above.

If your plugin is expected to modify part of the existing files with its scaffold, you may use functions provided by [sigs.k8s.io/kubebuilder/v3/pkg/plugin/util][kb-util].
See [example of deploy-image][example-of-deploy-image-2].
In brief, the util package helps you customize your scaffold in a lower level.

### Use Kubebuilder Machinery Lib

Notice that Kubebuilder also provides [machinery pkg][kubebuilder-machinery-pkg] where you can:

- Define file I/O behavior.
- Add markers to the scaffolded file.
- Define the template for scaffolding.

#### Overwrite A File

You might want for example to overwrite a scaffold done by using the option:

```go
	f.IfExistsAction = machinery.OverwriteFile
```

Let's imagine that you would like to have a helper plugin that would be called in a chain with `go/v4` to add customizations on top.
Therefore after we generate the code calling the subcommand to `init` from `go/v4` we would like to overwrite the Makefile to change this scaffold via our plugin.
In this way, we would implement the Bollerplate for our Makefile and then use this option to ensure that it would be overwritten.

See [example of deploy-image][example-of-deploy-image-1].

### A Combination of Multiple Plugins

Since your plugin may work frequently with other plugins, the executing command for scaffolding may become cumbersome, e.g:

```shell
kubebuilder create api --plugins=go/v3,kustomize/v1,yourplugin/v1
```

You can probably define a method to your scaffolder that calls the plugin scaffolding method in order.
See [example of deploy-image][example-of-deploy-image-3].

#### Define Plugin Bundles

Alternatively, you can create a plugin bundle to include the target plugins. For instance:

```go
  mylanguagev1Bundle, _ := plugin.NewBundle(plugin.WithName(language.DefaultNameQualifier), 
        plugin.WithVersion(plugin.Version{Number: 1}),
        plugin.WithPlugins(kustomizecommonv1.Plugin{}, mylanguagev1.Plugin{}), // extend the common base from Kuebebuilder
        // your plugin language which will do the scaffolds for the specific language on top of the common base
    )
```

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[deploy-image]: https://github.com/kubernetes-sigs/kubebuilder/tree/v3.7.0/pkg/plugins/golang/deploy-image/v1alpha1
[grafana]: https://github.com/kubernetes-sigs/kubebuilder/tree/v3.7.0/pkg/plugins/optional/grafana/v1alpha
[extending-cli]: ./extending-cli.md
[kb-util]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/plugin/util
[example-of-deploy-image-1]: https://github.com/kubernetes-sigs/kubebuilder/blob/df1ed6ccf19df40bd929157a91eaae6a9215bfc6/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/api/types.go#L58
[example-of-deploy-image-2]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/api.go#L170-L266
[example-of-deploy-image-3]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/api.go#L77-L98
[available-plugins]: ./available-plugins.md
[operator-sdk-manifest]: https://github.com/operator-framework/operator-sdk/tree/v1.23.0/internal/plugins/manifests/v2
[operator-sdk-scorecard]: https://github.com/operator-framework/operator-sdk/tree/v1.23.0/internal/plugins/scorecard/v2
[operator-sdk-plugin-ref]: https://github.com/operator-framework/operator-sdk/blob/v1.23.0/internal/cmd/operator-sdk/cli/cli.go#L78-L160
[plugins-section]: ???
[plugins-phase1-design-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/designs/extensible-cli-and-scaffolding-plugins-phase-1.md#extensible-cli-and-scaffolding-plugins
[plugins-phase1-design-doc-1.5]: https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/designs/extensible-cli-and-scaffolding-plugins-phase-1-5.md#extensible-cli-and-scaffolding-plugins---phase-15
[plugins-phase2-design-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/designs/extensible-cli-and-scaffolding-plugins-phase-2.md#extensible-cli-and-scaffolding-plugins---phase-2
[go-v3-main-template]: https://github.com/kubernetes-sigs/kubebuilder/blob/3bfc84ec8767fa760d1771ce7a0cb05a9a8f6286/pkg/plugins/golang/v3/scaffolds/internal/templates/main.go#L183
[kubebuilder-machinery]: https://github.com/kubernetes-sigs/kubebuilder/blob/3bfc84ec8767fa760d1771ce7a0cb05a9a8f6286/pkg/plugins/golang/v3/scaffolds/internal/templates/main.go#L28
[go-v3-settemplatedefault]: https://github.com/kubernetes-sigs/kubebuilder/blob/3bfc84ec8767fa760d1771ce7a0cb05a9a8f6286/pkg/plugins/golang/v3/scaffolds/internal/templates/main.go#L40
[go-v3-scaffold-execute]: https://github.com/kubernetes-sigs/kubebuilder/blob/3bfc84ec8767fa760d1771ce7a0cb05a9a8f6286/pkg/plugins/golang/v3/scaffolds/init.go#L120
[kubebuilder-machinery-pkg]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v3/pkg/machinery#section-documentation
