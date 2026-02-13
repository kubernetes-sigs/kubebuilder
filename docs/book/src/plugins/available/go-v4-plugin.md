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

## Multiple Controllers for the Same API

The `create api` subcommand supports a `--controller-name` flag that allows scaffolding multiple controllers for the same Group/Version/Kind. This is useful when you need to split reconciliation responsibilities, run different modes of reconciliation, or manage migration scenarios.

```sh
# Create the initial API and controller
kubebuilder create api --group cache --version v1alpha1 --kind Memcached

# Add a second controller for the same Memcached Kind
kubebuilder create api --group cache --version v1alpha1 --kind Memcached \
  --controller-name memcached-backup --resource=false
```

When `--controller-name` is provided:
- The controller file is named after the controller name (e.g. `internal/controller/memcached_backup_controller.go`)
- The reconciler struct uses the PascalCase form (e.g. `MemcachedBackupReconciler`)
- The controller is registered with `Named("memcached-backup")` for unique metrics and logging
- The name is tracked in the [PROJECT file][project-doc] via the `controllerName` field

The `--controller-name` value must be a DNS-1035 label (lowercase alphanumeric and hyphens, starting with a letter).

## Further resources

- To see the composition of plugins, you can check the source code for the Kubebuilder [main.go][plugins-main].
- Check the code implementation of the [base Golang plugin `base.go.kubebuilder.io/v4`][v4-plugin].
- Check the code implementation of the [Kustomize/v2 plugin][kustomize-plugin].
- Check [controller-runtime][controller-runtime] to know more about controllers.

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[quickstart]: ./../../quick-start.md
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata
[plugins-main]: ./../../../../../cmd/main.go
[kustomize-plugin]: ./../../plugins/available/kustomize-v2.md
[kustomize]: https://github.com/kubernetes-sigs/kustomize
[standard-go-project]: https://github.com/golang-standards/project-layout
[v4-plugin]: ./../../../../../pkg/plugins/golang/v4
[migration-guide-doc]: ./../../migration/migration_guide_gov3_to_gov4.md
[project-doc]: ./../../reference/project-config.md
[bundle]: ./../../../../../pkg/plugin/bundle.go
