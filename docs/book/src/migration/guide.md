# Migration from v1 to v2

Make sure you understand the [differences between Kubebuilder v1 and v2](./v1vsv2.md)
before continuing

Please ensure you have followed the [installation guide](/quick-start.md#installation)
to install the required components.

The recommended way to migrate a v1 project is to create a new v2 project and
copy over the API and the reconciliation code. The conversion will end up with a
project that looks like a native v2 project. However, in some cases, it's
possible to do an in-place upgrade (i.e. reuse the v1 project layout, upgrading
controller-runtime and controller-tools.  

Let's take the [example v1 project][v1-project] and migrate it to Kubebuilder
v2. At the end, we should have something that looks like the
[example v2 project][v2-project].

## Preparation

We'll need to figure out what the group, version, kind and domain are.

Let's take a look at our current v1 project structure:

```
pkg/
├── apis
│   ├── addtoscheme_batch_v1.go
│   ├── apis.go
│   └── batch
│       ├── group.go
│       └── v1
│           ├── cronjob_types.go
│           ├── cronjob_types_test.go
│           ├── doc.go
│           ├── register.go
│           ├── v1_suite_test.go
│           └── zz_generated.deepcopy.go
├── controller
└── webhook
```

All of our API information is stored in `pkg/apis/batch`, so we can look
there to find what we need to know.

In `cronjob_types.go`, we can find

```go
type CronJob struct {...}
```

In `register.go`, we can find

```go
SchemeGroupVersion = schema.GroupVersion{Group: "batch.tutorial.kubebuilder.io", Version: "v1"}
```

Putting that together, we get `CronJob` as the kind, and `batch.tutorial.kubebuilder.io/v1` as the group-version

## Initialize a v2 Project

Now, we need to initialize a v2 project.  Before we do that, though, we'll need
to initialize a new go module if we're not on the `gopath`:

```bash
go mod init tutorial.kubebuilder.io/project
```

Then, we can finish initializing the project with kubebuilder:

```bash
kubebuilder init --domain tutorial.kubebuilder.io
```

## Migrate APIs and Controllers

Next, we'll re-scaffold out the API types and controllers. Since we want both,
we'll say yes to both the API and controller prompts when asked what parts we
want to scaffold:

```bash
kubebuilder create api --group batch --version v1 --kind CronJob
```

If you're using multiple groups, some manual work is required to migrate.
Please follow [this](/migration/multi-group.md) for more details.

### Migrate the APIs

Now, let's copy the API definition from `pkg/apis/batch/v1/cronjob_types.go` to
`api/v1/cronjob_types.go`. We only need to copy the implementation of the `Spec`
and `Status` fields.

We can replace the `+k8s:deepcopy-gen:interfaces=...` marker (which is
[deprecated in kubebuilder](/reference/markers/object.md)) with
`+kubebuilder:object:root=true`.

We don't need the following markers any more (they're not used anymore, and are
relics from much older versions of KubeBuilder):

```go
// +genclient
// +k8s:openapi-gen=true
```

Our API types should look like the following:

```go
// +kubebuilder:object:root=true

// CronJob is the Schema for the cronjobs API
type CronJob struct {...}

// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob
type CronJobList struct {...}
```

### Migrate the Controllers

Now, let's migrate the controller reconciler code from
`pkg/controller/cronjob/cronjob_controller.go` to
`controllers/cronjob_controller.go`.

We'll need to copy
- the fields from the `ReconcileCronJob` struct to `CronJobReconciler`
- the contents of the `Reconcile` function
- the [rbac related markers](/reference/markers/rbac.md) to the new file.
- the code under `func add(mgr manager.Manager, r reconcile.Reconciler) error`
to `func SetupWithManager`

## Migrate the Webhooks

If you don't have a webhook, you can skip this section.

### Webhooks for Core Types and External CRDs

If you are using webhooks for Kubernetes core types (e.g. Pods), or for an
external CRD that is not owned by you, you can refer the
[controller-runtime example for builtin types][builtin-type-example]
and do something similar. Kubebuilder doesn't scaffold much for these cases, but
you can use the library in controller-runtime.

### Scaffold Webhooks for our CRDs

Now let's scaffold the webhooks for our CRD (CronJob). We'll need to run the
following command with the `--defaulting` and `--programmatic-validation` flags
(since our test project uses defaulting and validating webhooks):

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation
```

Depending on how many CRDs need webhooks, we may need to run the above command
multiple times with different Group-Version-Kinds.

Now, we'll need to copy the logic for each webhook. For validating webhooks, we
can copy the contents from
`func validatingCronJobFn` in `pkg/default_server/cronjob/validating/cronjob_create_handler.go`
to `func ValidateCreate` in `api/v1/cronjob_webhook.go` and then the same for `update`.

Similarly, we'll copy from `func mutatingCronJobFn` to `func Default`.

### Webhook Markers

When scaffolding webhooks, Kubebuilder v2 adds the following markers:

```
// These are v2 markers

// This is for the mutating webhook
// +kubebuilder:webhook:path=/mutate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=true,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=mcronjob.kb.io

...

// This is for the validating webhook
// +kubebuilder:webhook:path=/validate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=false,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=vcronjob.kb.io
```

The default verbs are `verbs=create;update`. We need to ensure `verbs` matches
what we need. For example, if we only want to validate creation, then we would
change it to `verbs=create`.

We also need to ensure `failure-policy` is still the same.

Markers like the following are no longer needed (since they deal with
self-deploying certificate configuration, which was removed in v2):

```go
// v1 markers
// +kubebuilder:webhook:port=9876,cert-dir=/tmp/cert
// +kubebuilder:webhook:service=test-system:webhook-service,selector=app:webhook-server
// +kubebuilder:webhook:secret=test-system:webhook-server-secret
// +kubebuilder:webhook:mutating-webhook-config-name=test-mutating-webhook-cfg
// +kubebuilder:webhook:validating-webhook-config-name=test-validating-webhook-cfg
```

In v1, a single webhook marker may be split into multiple ones in the same
paragraph. In v2, each webhook must be represented by a single marker.

## Others

If there are any manual updates in `main.go` in v1, we need to port the changes
to the new `main.go`. We'll also need to ensure all of the needed schemes have
been registered.

If there are additional manifests added under `config` directory, port them as
well.

Change the image name in the Makefile if needed.

## Verification

Finally, we can run `make` and `make docker-build` to ensure things are working
fine.

[v1-project]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/migration/testdata/gopath/project-v1
[v2-project]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project
[builtin-type-example]: https://sigs.k8s.io/controller-runtime/examples/builtins
