# [Deprecated] Kustomize (kustomize/v1)

<aside class="note warning">
<h1>Deprecated</h1>

The kustomize/v1 plugin is deprecated. If you are using this plugin, it is recommended
to migrate to the kustomize/v2 plugin which uses Kustomize v5 and provides support for
Apple Silicon (M1).

If you are using Golang projects scaffolded with `go/v3` which uses this version please, check 
the [Migration guide](../migration/v3vsv4.md) to learn how to upgrade your projects.

</aside>

The kustomize plugin allows you to scaffold all kustomize manifests used to work with the language plugins such as `go/v2` and `go/v3`. 
By using the kustomize plugin, you can create your own language plugins and ensure that you will have the same configurations 
and features provided by it. 

<aside class="note">
<h1>Supportability</h1>

This plugin uses [kubernetes-sigs/kustomize](https://github.com/kubernetes-sigs/kustomize) v3 and the architectures supported are: 
- linux/amd64
- linux/arm64
- darwin/amd64

You might want to consider using [kustomize/v2](./kustomize-v2.md) if you are looking to scaffold projects in
other architecture environments. (i.e. if you are looking to scaffold projects with Apple Silicon/M1 (`darwin/arm64`) this plugin 
will not work, more info: [kubernetes-sigs/kustomize#4612](https://github.com/kubernetes-sigs/kustomize/issues/4612)).
</aside> 

Note that projects such as [Operator-sdk][sdk] consume the Kubebuilder project as a lib and provide options to work with other languages
like Ansible and Helm. The kustomize plugin allows them to easily keep a maintained configuration and ensure that all languages have
the same configuration. It is also helpful if you are looking to provide nice plugins which will perform changes on top of
what is scaffolded by default. With this approach we do not need to keep manually updating this configuration in all possible language plugins
which uses the same and we are also
able to create "helper" plugins which can work with many projects and languages.

<aside class="note">
<h1>Examples</h1>

You can check the kustomize content by looking at the `config/` directory. Samples are provided under the [testdata][testdata]
directory of the Kubebuilder project.

</aside> 


## When to use it ?

If you are looking to scaffold the kustomize configuration manifests for your own language plugin 

## How to use it ?

If you are looking to define that your language plugin should use kustomize use the [Bundle Plugin][bundle]
to specify that your language plugin is a composition with your plugin responsible for scaffold
all that is language specific and kustomize for its configuration, see: 

```go
	// Bundle plugin which built the golang projects scaffold by Kubebuilder go/v3
	// The follow code is creating a new plugin with its name and version via composition
	// You can define that one plugin is composite by 1 or Many others plugins
	gov3Bundle, _ := plugin.NewBundle(plugin.WithName(golang.DefaultNameQualifier), 
		plugin.WithVersion(plugin.Version{Number: 3}),
		plugin.WithPlugins(kustomizecommonv1.Plugin{}, golangv3.Plugin{}), // scaffold the config/ directory and all kustomize files
		// Scaffold the Golang files and all that specific for the language e.g. go.mod, apis, controllers
	)
```

Also, with Kubebuilder, you can use kustomize alone via:

```sh
kubebuilder init --plugins=kustomize/v1 
$ ls -la 
total 24
drwxr-xr-x   6 camilamacedo86  staff  192 31 Mar 09:56 .
drwxr-xr-x  11 camilamacedo86  staff  352 29 Mar 21:23 ..
-rw-------   1 camilamacedo86  staff  129 26 Mar 12:01 .dockerignore
-rw-------   1 camilamacedo86  staff  367 26 Mar 12:01 .gitignore
-rw-------   1 camilamacedo86  staff   94 31 Mar 09:56 PROJECT
drwx------   6 camilamacedo86  staff  192 31 Mar 09:56 config
```

Or combined with the base language plugins:

```sh
# Provides the same scaffold of go/v3 plugin which is a composition (kubebuilder init --plugins=go/v3)
kubebuilder init --plugins=kustomize/v1,base.go.kubebuilder.io/v3 --domain example.org --repo example.org/guestbook-operator 
```

## Subcommands

The kustomize plugin implements the following subcommands:

* init (`$ kubebuilder init [OPTIONS]`)
* create api (`$ kubebuilder create api [OPTIONS]`)
* create webhook (`$ kubebuilder create api [OPTIONS]`)

<aside class="note">
<h1>Create API and Webhook</h1>

Its implementation for the subcommand create api will scaffold the kustomize manifests
which are specific for each API, see [here][kustomize-create-api]. The same applies
to its implementation for create webhook.

</aside> 

## Affected files

The following scaffolds will be created or updated by this plugin:

* `config/*`

## Further resources

* Check the kustomize [plugin implementation](https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/common/kustomize) 
* Check the [kustomize documentation][kustomize-docs]
* Check the [kustomize repository][kustomize-github]

[sdk]:https://github.com/operator-framework/operator-sdk
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/
[bundle]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugin/bundle.go
[kustomize-create-api]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugins/common/kustomize/v1/scaffolds/api.go#L72-L84
[kustomize-docs]: https://kustomize.io/
[kustomize-github]: https://github.com/kubernetes-sigs/kustomize