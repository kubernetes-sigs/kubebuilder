# go/v2 plugin

The `go/v2` plugin has the purpose to scaffold Golang projects to help users to build projects with [controllers][controller-runtime] and keep the backwards compatibility with the default scaffold made using Kubebuilder CLI `2.x.z` releases.   

<node>

You can check samples using this plugin by looking at the `project-v2-<options>` directories under the [testdata][testdata] projects on the root directory of the Kubebuilder project.

</node>  

## When should I use this plugin

Only if you are looking to scaffold a project with the legacy layout. Otherwise, it is recommended you to use the default Golang version plugin. 

<aside class="note warning">

<h1> Note </h1>

Be aware that this plugin version does not provide a scaffold compatible with the latest versions of the dependencies used in order to keep its backwards compatibility. 

</aside>

## How to use it ?

To initialize a Golang project using the legacy layout and with this plugin run, e.g.:

```sh
kubebuilder init --domain tutorial.kubebuilder.io --repo tutorial.kubebuilder.io/project --plugins=go/v2
```
<aside class="note">

<h1> Note </h1>

By creating a project with this plugin, the PROJECT file scaffold will be using the previous schema (_project version 2_).  So that Kubebuilder CLI knows what plugin version was used and will call its subcommands such as `create api` and `create webhooks`.  Note that further Golang plugins versions use the new Project file schema, which tracks the information about what plugins and versions have been used so far. 

</aside>

## Subcommands supported by the plugin

-  Init -  `kubebuilder init [OPTIONS]`
-  Edit -  `kubebuilder edit [OPTIONS]`
-  Create API -  `kubebuilder create api [OPTIONS]`
-  Create Webhook - `kubebuilder create webhook [OPTIONS]`

## Further resources

- Check the code implementation of the [go/v2 plugin][v2-plugin].

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata
[v2-plugin]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/golang/v2