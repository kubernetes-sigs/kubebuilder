# Migration from v2 to v3

Make sure you understand the [differences between Kubebuilder v2 and v3][v2vsv3]
before continuing.

Please ensure you have followed the [installation guide][quick-start]
to install the required components.

The recommended way to migrate a v2 project is to create a new v3 project and
copy over the API and the reconciliation code. The conversion will end up with a
project that looks like a native v3 project. However, in some cases, it's
possible to do an in-place upgrade (i.e. reuse the v2 project layout, upgrading
[controller-runtime][controller-runtime] and [controller-tools][controller-tools]).  

## Initialize a v3 Project

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

Now, we need to initialize a v3 project.  Before we do that, though, we'll need
to initialize a new go module if we're not on the `GOPATH`. While technically this is
not needed inside `GOPATH`, it is still recommended.

```bash
go mod init tutorial.kubebuilder.io/migration-project
```
<aside class="note warning">
<h1> Migrating to Kubebuilder v3 while staying on the go/v2 plugin </h1>

You can use `--plugins=go/v2` if you wish to continue using "`Kubebuilder 2.x`" layout and avoid dealing with the breaking changes that will be faced because of the default upper versions which will be used now. See that the [controller-tools][controller-tools] `v0.5.0` & [controller-runtime][controller-runtime] `v0.8.3` are just used by default with the `go/v3` plugin layout. 
</aside>

<aside class="note">
<h1>The module of your project can found in the in the `go.mod` file at the root of your project:</h1>

```
module tutorial.kubebuilder.io/migration-project
```

</aside>

Then, we can finish initializing the project with kubebuilder.

```bash
kubebuilder init --domain tutorial.kubebuilder.io
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

<aside class="note">
<h1>How to still keep `apiextensions.k8s.io/v1beta1` for CRDs?</h1>

From now on, the CRDs that will be created by controller-gen will be using the Kubernetes API version `apiextensions.k8s.io/v1`  by default, instead of `apiextensions.k8s.io/v1beta1`. 

The `apiextensions.k8s.io/v1beta1` was deprecated in Kubernetes `1.16` and will be removed in Kubernetes `1.22`.

So, if you would like to keep using the previous version use the flag `--crd-version=v1beta1` in the above command which is only needed if you want your operator to support Kubernetes `1.15` and earlier.

</aside>

### Migrate the APIs

<aside class="note">
<h1>If you're using multiple groups</h1>

Please run `kubebuilder edit --multigroup=true` to enable multi-group support before migrating the APIs and controllers. Please see [this][multi-group] for more details.

</aside>

Now, let's copy the API definition from `api/v1/<kind>_types.go` in our old project to the new one.

These files have not been modified by the new plugin, so you should be able to replace your freshly scaffolded files by your old one. There may be some cosmetic changes. So you can choose to only copy the types themselves.

### Migrate the Controllers

Now, let's migrate the controller code from `controllers/cronjob_controller.go` in our old project to the new one. There is a breaking change and there may be some cosmetic changes.

The new `Reconcile` method receives the context as an argument now, instead of having to create it with `context.Background()`. You can copy the rest of the code in your old controller to the scaffolded methods replacing:

```go 
func (r *CronJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
    ctx := context.Background() 
    log := r.Log.WithValues("cronjob", req.NamespacedName)
```

With:

```go 
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("cronjob", req.NamespacedName)
```

<aside class="note warning">
<h1>Controller-runtime version updated has breaking changes</h1>

Check [sigs.k8s.io/controller-runtime release docs from 0.8.0+ version][controller-runtime] for breaking changes.

</aside>

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

<aside class="note">
<h1>How to keep using `apiextensions.k8s.io/v1beta1` for Webhooks?</h1>

From now on, the Webhooks that will be created by Kubebuilder using by default the Kubernetes API version `admissionregistration.k8s.io/v1` instead of `admissionregistration.k8s.io/v1beta1` and the `cert-manager.io/v1` to replace `cert-manager.io/v1alpha2`. 

Note that `apiextensions/v1beta1` and `admissionregistration.k8s.io/v1beta1` were deprecated in Kubernetes `1.16` and will be removed  in Kubernetes `1.22`. If you use `apiextensions/v1` and `admissionregistration.k8s.io/v1` then you need to use `cert-manager.io/v1` which will be the API adopted per Kubebuilder CLI by default in this case.  

The API `cert-manager.io/v1alpha2` is not compatible with the latest Kubernetes API versions. 

So, if you would like to keep using the previous version use the flag `--webhook-version=v1beta1` in the above command which is only needed if you want your operator to support Kubernetes `1.15` and earlier.

</aside>

Now, let's copy the webhook definition from `api/v1/<kind>_webhook.go` from our old project to the new one. 

## Others

If there are any manual updates in `main.go` in v2, we need to port the changes to the new `main.go`. We’ll also need to ensure all of the needed schemes have been registered.

If there are additional manifests added under config directory, port them as well.

Change the image name in the Makefile if needed.

## Verification

Finally, we can run `make` and `make docker-build` to ensure things are working
fine.

[v2vsv3]: ./v2vsv3.md
[quick-start]: /quick-start.md#installation
[controller-tools]: https://github.com/kubernetes-sigs/controller-tools/releases
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime/releases
[multi-group]: /migration/multi-group.md