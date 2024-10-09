# Kustomize v2

**(Default Scaffold)**

The Kustomize plugin allows you to scaffold all kustomize manifests used
with the language base plugin `base.go.kubebuilder.io/v4`.
This plugin is used to generate the manifest under the `config/` directory
for projects built within the `go/v4` plugin (default scaffold).

Projects like [Operator-sdk][sdk] use the Kubebuilder project as a library
and provide options for working with other languages such as Ansible and Helm.
The Kustomize plugin helps them maintain consistent configuration across
languages. It also simplifies the creation of plugins that perform
changes on top of the default scaffold, removing the need for manual
updates across multiple language plugins. This approach allows the
creation of "helper" plugins that work with different projects
and languages.

<aside class="note">
<h1>Examples</h1>

You can check the kustomize content by looking at the `config/` directory provided in the sample `project-v4-*`
under the [testdata][testdata] directory of the Kubebuilder project.

</aside>

## How to use it

If you want your language plugin to use kustomize, use the [Bundle Plugin][bundle] to specify that your language plugin is composed of your language-specific plugin and kustomize for its configuration, as shown:

```go
import (
   ...
   kustomizecommonv2 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2"
   golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
   ...
)

// Bundle plugin for Golang projects scaffolded by Kubebuilder go/v4
gov4Bundle, _ := plugin.NewBundle(plugin.WithName(golang.DefaultNameQualifier),
    plugin.WithVersion(plugin.Version{Number: 4}),
    plugin.WithPlugins(kustomizecommonv2.Plugin{}, golangv4.Plugin{}), // Scaffold the config/ directory and all kustomize files
)
```

You can also use kustomize/v2 alone via:

```sh
kubebuilder init --plugins=kustomize/v2
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
# Provides the same scaffold of go/v4 plugin which is composition but with kustomize/v2
kubebuilder init --plugins=kustomize/v2,base.go.kubebuilder.io/v4 --domain example.org --repo example.org/guestbook-operator
```

## Subcommands

The kustomize plugin implements the following subcommands:

* init (`$ kubebuilder init [OPTIONS]`)
* create api (`$ kubebuilder create api [OPTIONS]`)
* create webhook (`$ kubebuilder create api [OPTIONS]`)

<aside class="note">
<h1>Create API and Webhook</h1>

The implementation for the `create api` subcommand scaffolds the kustomize
manifests specific to each API. See more [here][kustomize-create-api].
The same applies to `create webhook`.

</aside>

## Affected files

The following scaffolds will be created or updated by this plugin:

* `config/*`

## Further resources

* Check the kustomize [plugin implementation](https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/common/kustomize)
* Check the [kustomize documentation][kustomize-docs]
* Check the [kustomize repository][kustomize-github]

[sdk]:https://github.com/operator-framework/operator-sdk
[kustomize-docs]: https://kustomize.io/
[kustomize-github]: https://github.com/kubernetes-sigs/kustomize
[kustomize-replacements]: https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/replacements/
[kustomize-vars]: https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/vars/
[release-notes-v5]: https://github.com/kubernetes-sigs/kustomize/releases/tag/kustomize%2Fv5.0.0
[release-notes-v4]: https://github.com/kubernetes-sigs/kustomize/releases/tag/kustomize%2Fv4.0.0
[testdata]: ./../../../../../testdata/
[bundle]: ./../../../../../pkg/plugin/bundle.go
[kustomize-create-api]: ./../../../../../pkg/plugins/common/kustomize/v2/scaffolds/api.go
