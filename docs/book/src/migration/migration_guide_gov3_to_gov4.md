# Migration from go/v3 to go/v4

Make sure you understand the [differences between Kubebuilder go/v3 and go/v4][v3vsv4]
before continuing.

Please ensure you have followed the [installation guide][quick-start]
to install the required components.

The recommended way to migrate a `go/v3` project is to create a new `go/v4` project and
copy over the API and the reconciliation code. The conversion will end up with a
project that looks like a native go/v4 project layout (latest version).

<aside class="note warning">
<h1>Your Upgrade Assistant: The `alpha generate` command</h1>

To upgrade your project you might want to use the command `kubebuilder alpha generate [OPTIONS]`.
This command will re-scaffold the project using the current Kubebuilder version.
You can run `kubebuilder alpha generate --plugins=go/v4` to regenerate your project using `go/v4`
based in your [PROJECT][project-file] file config. ([More info](./../reference/rescaffold.md))

</aside>

However, in some cases, it's possible to do an in-place upgrade (i.e. reuse the go/v3 project layout, upgrading
the PROJECT file, and scaffolds manually). For further information see [Migration from go/v3 to go/v4 by updating the files manually][manually-upgrade]

## Initialize a go/v4 Project

<aside class="note">
<h1>Project name</h1>

For the rest of this document, we are going to use `migration-project` as the project name and `tutorial.kubebuilder.io` as the domain. Please, select and use appropriate values for your case.

</aside>

Create a new directory with the name of your project. Note that
this name is used in the scaffolds to create the name of your manager Pod and of the Namespace where the Manager is deployed by default.

```bash
$ mkdir migration-project-name
$ cd migration-project-name
```

Now, we need to initialize a go/v4 project.  Before we do that, we'll need
to initialize a new go module if we're not on the `GOPATH`. While technically this is
not needed inside `GOPATH`, it is still recommended.

```bash
go mod init tutorial.kubebuilder.io/migration-project
```

<aside class="note">
<h1>The module of your project can found in the `go.mod` file at the root of your project:</h1>

```
module tutorial.kubebuilder.io/migration-project
```

</aside>

Now, we can finish initializing the project with kubebuilder.

```bash
kubebuilder init --domain tutorial.kubebuilder.io --plugins=go/v4
```

<aside class="note">
<h1>The domain of your project can be found in the PROJECT file:</h1>

```yaml
...
domain: tutorial.kubebuilder.io
...
```
</aside>

## Migrate APIs and Controllers

Next, we'll re-scaffold out the API types and controllers.

<aside class="note">
<h1>Scaffolding both the API types and controllers</h1>

For this example, we are going to consider that we need to scaffold both the API types and the controllers, but remember that this depends on how you scaffolded them in your original project.

</aside>

```bash
kubebuilder create api --group batch --version v1 --kind CronJob
```

### Migrate the APIs

<aside class="note">
<h1>If you're using multiple groups</h1>

Please run `kubebuilder edit --multigroup=true` to enable multi-group support before migrating the APIs and controllers. Please see [this][multi-group] for more details.

</aside>

Now, let's copy the API definition from `api/v1/<kind>_types.go` in our old project to the new one.

These files have not been modified by the new plugin, so you should be able to replace your freshly scaffolded files by your old one. There may be some cosmetic changes. So you can choose to only copy the types themselves.

### Migrate the Controllers

Now, let's migrate the controller code from `controllers/cronjob_controller.go` in our old project to `internal/controller/cronjob_controller.go` in the new one.

## Migrate the Webhooks

<aside class="note">
<h1>Skip</h1>

If you don't have any webhooks, you can skip this section.

</aside>

Now let's scaffold the webhooks for our CRD (CronJob). We'll need to run the
following command with the `--defaulting` and `--programmatic-validation` flags
(since our test project uses defaulting and validating webhooks):

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation
```

Now, let's copy the webhook definition from `api/v1/<kind>_webhook.go` from our old project to the new one.

## Others

If there are any manual updates in `main.go` in v3, we need to port the changes to the new `main.go`. We’ll also need to ensure all of needed controller-runtime `schemes` have been registered.

If there are additional manifests added under config directory, port them as well. Please, be aware that
the new version go/v4 uses Kustomize v5x and no longer Kustomize v4. Therefore, if added customized
implementations in the config you need to ensure that they can work with Kustomize v5 and if not
update/upgrade any breaking change that you might face.

In v4, installation of Kustomize has been changed from bash script to `go get`. Change the `kustomize` dependency in Makefile to
```
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)
```

Change the image name in the Makefile if needed.

## Verification

Finally, we can run `make` and `make docker-build` to ensure things are working
fine.

[v3vsv4]: v3vsv4.md
[quick-start]: ./../quick-start.md#installation
[controller-tools]: https://github.com/kubernetes-sigs/controller-tools/releases
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime/releases
[multi-group]: multi-group.md
[manually-upgrade]: manually_migration_guide_gov3_to_gov4.md
[project-file]: ../reference/project-config.md
