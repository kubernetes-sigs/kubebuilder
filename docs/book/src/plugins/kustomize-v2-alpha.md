# Kustomize v2-alpha 

The kustomize plugin allows you to scaffold all kustomize manifests used to work with the language base plugin `base.go.kubebuilder.io/v3`.

Note that projects such as [Operator-sdk][sdk] consume the Kubebuilder project as a lib and provide options to work with other languages
like Ansible and Helm. The kustomize plugin allows them to easily keep a maintained configuration and ensure that all languages have
the same configuration. It is also helpful if you are looking to provide nice plugins which will perform changes on top of
what is scaffolded by default. With this approach we do not need to keep manually updating this configuration in all possible language plugins
which uses the same and we are also
able to create "helper" plugins which can work with many projects and languages.

<aside class="note">
<h1>Examples</h1>

You can check the kustomize content by looking at the `config/` directory provide on the sample `project-v3-with-kustomize-v2` under the [testdata][testdata]
directory of the Kubebuilder project.

</aside> 

## When to use it

- If you are looking to scaffold the kustomize configuration manifests for your own language plugin
- If you are looking for support on Apple Silicon (`darwin/arm64`). (_Before kustomize `4.x` the binary for this plataform is not provided_)
- If you are looking for to begin to try out the new syntax and features provide by kustomize v4
- If you are NOT looking to build projects which will be used on Kubernetes cluster versions < `1.22` (_The new features provides by kustomize v4 are not officially supported and might not work with kubectl < `1.22`_)
- If you are NOT looking to rely on special URLs in resource fields
- If you want to use [replacements][https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/replacements/] since [vars][https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/vars/] are deprecated and might be removed soon

<aside class="note">
<h1>Supportability</h1>

You can use `kustomize/v1` plugin which is the default configuration adopted by the `go/v3` plugin if you are not prepared to began to experiment kustomize `v4`.
Also, be aware that the `base.go.kubebuilder.io/v3` is prepared to work with this plugin. 

</aside> 


## How to use it

If you are looking to define that your language plugin should use kustomize use the [Bundle Plugin][bundle]
to specify that your language plugin is a composition with your plugin responsible for scaffold
all that is language specific and kustomize for its configuration, see:

```go
import (
...
   kustomizecommonv2 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v2"
   golangv3 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3"
...
)

	// Bundle plugin which built the golang projects scaffold by Kubebuilder go/v3
	// The follow code is creating a new plugin with its name and version via composition
	// You can define that one plugin is composite by 1 or Many others plugins
	gov3Bundle, _ := plugin.NewBundle(golang.DefaultNameQualifier, plugin.Version{Number: 3},
		kustomizecommonv2.Plugin{}, // scaffold the config/ directory and all kustomize files
		golangv3.Plugin{}, // Scaffold the Golang files and all that specific for the language e.g. go.mod, apis, controllers
	)
```

Also, with Kubebuilder, you can use kustomize/v2-alpha alone via:

```sh
kubebuilder init --plugins=kustomize/v2-alpha 
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
# Provides the same scaffold of go/v3 plugin which is composition but with kustomize/v2-alpha 
kubebuilder init --plugins=kustomize/v2-alpha,base.go.kubebuilder.io/v3 --domain example.org --repo example.org/guestbook-operator 
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
* To know more about the changes between kustomize v3 and v3 see its [release notes][release-notes]
* Also, you can compare the `config/` directory between the samples `project-v3` and `project-v3-with-kustomize-v2` to check the difference in the syntax of the manifests provided by default

[sdk]:https://github.com/operator-framework/operator-sdk
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/
[bundle]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugin/bundle.go
[kustomize-create-api]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugins/common/kustomize/v2/scaffolds/api.go#L72-L84
[kustomize-docs]: https://kustomize.io/
[kustomize-github]: https://github.com/kubernetes-sigs/kustomize
[release-notes]: https://github.com/kubernetes-sigs/kustomize/releases/tag/kustomize%2Fv4.0.0