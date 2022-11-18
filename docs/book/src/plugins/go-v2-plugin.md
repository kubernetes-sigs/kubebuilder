# [Deprecated] go/v2 (go.kubebuilder.io/v2 - "Kubebuilder 2.x" layout)

<aside class="note warning">
<h1>Deprecated</h1>

The `go/v2` plugin cannot scaffold projects in which CRDs and/or Webhooks have a `v1` API version.
The `go/v2` plugin scaffolds with the `v1beta1` API version which was deprecated in Kubernetes `1.16` and removed in `1.22`.
This plugin was kept to ensure backwards compatibility with projects that were scaffolded with the old `"Kubebuilder 2.x"` layout and does not work with the new plugin ecosystem that was introduced with Kubebuilder `3.0.0` [More info](plugins.md)

Since `28 Apr 2021`, the default layout produced by Kubebuilder changed and is done via the `go/v3`.
We encourage you migrate your project to the latest version if your project was built with a Kubebuilder
versions < `3.0.0`.

The recommended way to migrate a `v2` project is to create a new `v3` project and copy over the API
and the reconciliation code. The conversion will end up with a project that looks like a native `v3` project.
For further information check the [Migration guide](../migration/legacy/manually_migration_guide_v2_v3.md)

</aside>

The `go/v2` plugin has the purpose to scaffold Golang projects to help users
to build projects with [controllers][controller-runtime] and keep the backwards compatibility
with the default scaffold made using Kubebuilder CLI `2.x.z` releases.

<aside class="note">

You can check samples using this plugin by looking at the `project-v2-<options>` directories under the [testdata][testdata] projects on the root directory of the Kubebuilder project.

</aside>

## When should I use this plugin ?

Only if you are looking to scaffold a project with the legacy layout. Otherwise, it is recommended you to use the default Golang version plugin.

<aside class="note warning">

<h1>Note</h1>

Be aware that this plugin version does not provide a scaffold compatible with the latest versions of the dependencies used in order to keep its backwards compatibility.

</aside>

## How to use it ?

To initialize a Golang project using the legacy layout and with this plugin run, e.g.:

```sh
kubebuilder init --domain tutorial.kubebuilder.io --repo tutorial.kubebuilder.io/project --plugins=go/v2
```
<aside class="note">

<h1>Note</h1>

By creating a project with this plugin, the `PROJECT` file scaffold will be using the previous
schema (_project version 2_), so that Kubebuilder CLI knows what plugin version was used and will
call its subcommands such as `create api` and `create webhooks`.

Note that further Golang plugins versions use the new Project file schema, which tracks the
information about what plugins and versions have been used so far.

</aside>

## Subcommands supported by the plugin ?

-  Init -  `kubebuilder init [OPTIONS]`
-  Edit -  `kubebuilder edit [OPTIONS]`
-  Create API -  `kubebuilder create api [OPTIONS]`
-  Create Webhook - `kubebuilder create webhook [OPTIONS]`

## Further resources

- Check the code implementation of the [go/v2 plugin][v2-plugin].

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata
[v2-plugin]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/golang/v2