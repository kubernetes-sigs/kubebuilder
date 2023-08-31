# [Deprecated] Declarative Plugin

<aside class="note warning">
<h1>Notice of Deprecation</h1>

The Declarative plugin is an implementation derived from the [kubebuilder-declarative-pattern][kubebuilder-declarative-pattern] project. 
As the project maintainers possess the most comprehensive knowledge about its changes and Kubebuilder allows 
the creation of custom plugins using its library, it has been decided that this plugin will be better 
maintained within the [kubebuilder-declarative-pattern][kubebuilder-declarative-pattern] project itself, 
which falls under its domain of responsibility. This decision aims to improve the maintainability of both the 
plugin and Kubebuilder, ultimately providing an enhanced user experience. To follow up on this work, please refer 
to [Issue #293](https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/issues/293) in the 
kubebuilder-declarative-pattern repository.

</aside>

The declarative plugin allows you to create [controllers][controller-runtime] using the [kubebuilder-declarative-pattern][kubebuilder-declarative-pattern].
By using the declarative plugin, you can make the required changes on top of what is scaffolded by default when you create a Go project with Kubebuilder and the Golang plugins (i.e. go/v2, go/v3).

<aside class="note">
<h1>Examples</h1>

You can check samples using this plugin by looking at the "addon" samples inside the [testdata][testdata] directory of the Kubebuilder project.

</aside>

## When to use it ?

- If you are looking to scaffold one or more [controllers][controller-runtime] following [the pattern][kubebuilder-declarative-pattern] ( See an e.g. of the reconcile method implemented [here][addon-v3-controller])
- If you want to have manifests shipped inside your Manager container. The declarative plugin works with channels, which allow you to push manifests. [More info][addon-channels-info]

## How to use it ?

The declarative plugin requires to be used with one of the available Golang plugins
If you want that any API(s) and its respective controller(s) generate to reconcile them of your project adopt this partner then:

```sh
kubebuilder init --plugins=go/v3,declarative/v1 --domain example.org --repo example.org/guestbook-operator
```

If you want to adopt this pattern for specific API(s) and its respective controller(s) (not for any API/controller scaffold using Kubebuilder CLI) then:

```sh
kubebuilder create api --plugins=go/v3,declarative/v1 --version v1 --kind Guestbook
```

## Subcommands

The declarative plugin implements the following subcommands:

- init (`$ kubebuilder init [OPTIONS]`)
- create api (`$ kubebuilder create api [OPTIONS]`)

## Affected files

The following scaffolds will be created or updated by this plugin:

- `controllers/*_controller.go`
- `api/*_types.go`
- `channels/packages/<packagename>/<version>/manifest.yaml`
- `channels/stable`
- `Dockerfile`

## Further resources

- Read more about the [declarative pattern][kubebuilder-declarative-pattern]
- Watch the KubeCon 2018 Video [Managing Addons with Operators][kubecon-video]
- Check the [plugin implementation][plugin-implementation]

[addon-channels-info]: https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/blob/master/docs/addon/walkthrough/README.md#adding-a-manifest
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[kubebuilder-declarative-pattern]: https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/
[kubecon-video]: https://www.youtube.com/watch?v=LPejvfBR5_w
[plugin-implementation]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/golang/declarative
[addon-v3-controller]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/project-v3-declarative-v1

