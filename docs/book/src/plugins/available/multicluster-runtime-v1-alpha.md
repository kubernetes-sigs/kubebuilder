# multicluster-runtime/v1-alpha Plugin

The `multicluster-runtime/v1-alpha` plugin adds
[sigs.k8s.io/multicluster-runtime](https://github.com/kubernetes-sigs/multicluster-runtime)
support to a Kubebuilder project. Instead of reconciling objects in a single cluster, your
controller will reconcile objects across all clusters registered with the chosen provider.

This plugin is built into the `kubebuilder` binary — no separate installation is required.

## When to use this plugin

Use `multicluster-runtime/v1-alpha` when you need to:

- Manage resources across **multiple Kubernetes clusters** from a single operator binary
- React to cluster lifecycle events (clusters joining or leaving the fleet)
- Run **fleet-wide controllers** while still using familiar controller-runtime patterns

## Prerequisites

- Kubebuilder v4+
- Go 1.22+

## Project initialization

Chain `multicluster-runtime/v1-alpha` after `go/v4`:

```bash
kubebuilder init \
  --plugins go/v4,multicluster-runtime/v1-alpha \
  --domain example.com \
  --repo github.com/example/myop \
  --provider kubeconfig
```

The plugin rewrites `cmd/main.go` to use `mcmanager.New(...)` instead of `ctrl.NewManager(...)`.

## Provider selection guide

The `--provider` flag (default: `kubeconfig`) controls how clusters are discovered.

| Provider | Flag value | Use case |
|---|---|---|
| **Kubeconfig secrets** | `kubeconfig` | Dynamic fleet — clusters join/leave at runtime by creating kubeconfig Secrets |
| **Namespace** | `namespace` | Single cluster, namespace-per-tenant; each namespace is treated as a "cluster" |
| **Cluster API** | `cluster-api` | Fleet managed by [Cluster API](https://cluster-api.sigs.k8s.io/) controllers |
| **File** | `file` | Static cluster list; one kubeconfig file per cluster in a directory (great for CI) |

### Kubeconfig provider (default)

```bash
kubebuilder init --plugins go/v4,multicluster-runtime/v1-alpha \
  --domain example.com --repo github.com/example/myop \
  --provider kubeconfig
```

Add a cluster at runtime by creating a Secret with the kubeconfig:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-cluster
  namespace: default
  labels:
    # label recognized by the kubeconfig provider
    multicluster.x-k8s.io/cluster-name: my-cluster
type: Opaque
data:
  kubeconfig: <base64-encoded-kubeconfig>
```

### Namespace provider

```bash
kubebuilder init --plugins go/v4,multicluster-runtime/v1-alpha \
  --domain example.com --repo github.com/example/myop \
  --provider namespace
```

The generated `cmd/main.go` starts the provider and manager concurrently using `errgroup`.

### File provider

```bash
kubebuilder init --plugins go/v4,multicluster-runtime/v1-alpha \
  --domain example.com --repo github.com/example/myop \
  --provider file \
  --kubeconfig-dir /etc/kubeconfig
```

Place one kubeconfig file per cluster in `--kubeconfig-dir`. Each file name becomes the
cluster name.

## Create a multicluster controller

```bash
kubebuilder create api \
  --plugins go/v4,multicluster-runtime/v1-alpha \
  --group foo --version v1 --kind Foo \
  --controller --resource
```

The generated controller (`internal/controller/foo_controller.go`) uses:

- `mcreconcile.Request` — carries `ClusterName` in addition to the usual `NamespacedName`
- `mcbuilder.ControllerManagedBy(mgr)` — watches objects across all registered clusters
- `mcmanager.Manager` — the multicluster-aware manager type

### Using `req.ClusterName`

```go
func (r *FooReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

    // Fetch the object from the correct cluster's cache.
    foo := &foov1.Foo{}
    if err := r.Get(ctx, req.NamespacedName, foo); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    log.Info("Reconciling Foo", "name", foo.Name)
    // ... your business logic ...
    return ctrl.Result{}, nil
}
```

## Webhooks

Webhooks register with the **local cluster's** API server and do not need multicluster
changes. Scaffold the webhook files with only the `go/v4` plugin:

```bash
kubebuilder create webhook \
  --plugins go/v4 \
  --group foo --version v1 --kind Foo \
  --defaulting --programmatic-validation
```

Webhook setup is wired automatically into `cmd/main.go` via the standard
`// +kubebuilder:scaffold:builder` marker, which the multicluster templates include
alongside `// +kubebuilder:scaffold:multicluster-builder`. Controllers are registered
on the multicluster manager; webhooks are registered on the local webhook server —
no manual edits needed.

## Switching providers

Use `kubebuilder edit` to replace the provider in an existing project:

```bash
kubebuilder edit --plugins multicluster-runtime/v1-alpha --provider namespace
```

This rewrites `cmd/main.go` while preserving all `// +kubebuilder:scaffold:*` markers so
that future `kubebuilder create api` and `kubebuilder create webhook` commands still work.

## Plugin chain note

This plugin is designed to run **after** `go/v4`. The plugin chain `go/v4,multicluster-runtime/v1-alpha`
means:

1. `go/v4` scaffolds the standard project structure
2. `multicluster-runtime/v1-alpha` rewrites `cmd/main.go` to use the multicluster manager

If `go/v4` is absent from the chain, the scaffolded `cmd/main.go` will not compile
because the standard project structure (`api/`, `internal/controller/`, `Makefile`, etc.)
will be missing. Always chain `go/v4` first.
