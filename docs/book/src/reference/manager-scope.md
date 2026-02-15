# Manager Scope

Manager scope determines which namespace(s) your manager watches and manages resources in.

<aside class="note">
<h1>Manager Scope vs CRD Scope</h1>

Manager scope is independent from CRD scope. See [Understanding Scopes](./scopes.md) for an explanation of how these two concepts differ.
</aside>

## Overview

Kubebuilder supports three types of manager scope:

| Scope | Description | Use Case |
|-------|-------------|----------|
| **Cluster-scoped (default)** | Watches all namespaces in the cluster | Single manager managing resources cluster-wide |
| **Namespace-scoped** | Watches only specific namespace(s) | Multi-tenant, least-privilege deployments |
| **Multi-namespace** | Watches multiple specific namespaces | Manager managing resources in subset of namespaces |

Manager scope is configured through:
- RBAC resources (Role vs ClusterRole)
- Cache configuration in `cmd/main.go`
- `WATCH_NAMESPACE` environment variable

## Cluster-Scoped (Default)

By default, Kubebuilder scaffolds cluster-scoped managers that watch all namespaces in the cluster.

```bash
kubebuilder init --domain example.com
```

**Characteristics:**
- Uses `ClusterRole` and `ClusterRoleBinding` for RBAC
- Manager watches all namespaces
- No cache configuration needed

**When to use:**
- Single manager instance for the entire cluster
- Managing cluster-scoped resources (Nodes, ClusterRoles, Namespaces)
- Simpler RBAC model when cluster-wide access is acceptable

## Namespace-Scoped

Namespace-scoped managers watch only specific namespace(s), configured via the `WATCH_NAMESPACE` environment variable.

```bash
# New projects
kubebuilder init --domain example.com --namespaced

# Existing projects
kubebuilder edit --namespaced=true
```

**Characteristics:**
- Uses namespace-scoped `Role` and `RoleBinding` for RBAC
- Manager watches only specified namespace(s)
- Requires cache configuration in `cmd/main.go`
- Requires `namespace=` parameter in controller RBAC markers

**When to use:**
- Multi-tenant environments (one manager per tenant/namespace)
- Security policies requiring least-privilege access
- Multiple manager instances in different namespaces

**RBAC markers:**

Controllers in namespace-scoped projects use the `namespace=` parameter in RBAC markers to generate namespace-scoped `Role` resources:

```go
// +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mykinds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mykinds/finalizers,verbs=update
```

When controller-gen sees the `namespace=` parameter, it generates `kind: Role` instead of `kind: ClusterRole`. The namespace field is added by kustomize during the build process (configured in `config/default/kustomization.yaml`).

**Cache configuration:**

Kubebuilder automatically scaffolds the cache configuration in `cmd/main.go` when using `--namespaced` flag:

```go
// setupCacheNamespaces configures the cache to watch specific namespace(s).
// It supports both single namespace ("ns1") and multi-namespace ("ns1,ns2,ns3") formats.
func setupCacheNamespaces(namespaces string) cache.Options {
    defaultNamespaces := make(map[string]cache.Config)
    for ns := range strings.SplitSeq(namespaces, ",") {
        defaultNamespaces[strings.TrimSpace(ns)] = cache.Config{}
    }
    return cache.Options{
        DefaultNamespaces: defaultNamespaces,
    }
}

// In main()
watchNamespace, err := getWatchNamespace()
if err != nil {
    setupLog.Error(err, "Unable to get WATCH_NAMESPACE")
    os.Exit(1)
}

mgrOptions := ctrl.Options{
    Scheme:                 scheme,
    Metrics:                metricsServerOptions,
    WebhookServer:          webhookServer,
    HealthProbeBindAddress: probeAddr,
    LeaderElection:         enableLeaderElection,
    LeaderElectionID:       "your-leader-election-id",
}

// Configure cache to watch namespace(s) specified in WATCH_NAMESPACE
mgrOptions.Cache = setupCacheNamespaces(watchNamespace)
setupLog.Info("Watching namespace(s)", "namespaces", watchNamespace)

mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
```

This configuration works for both single namespace (`WATCH_NAMESPACE=my-namespace`) and multi-namespace (`WATCH_NAMESPACE=ns1,ns2,ns3`) scenarios.

## Multi-Namespace

Managers can watch multiple specific namespaces using comma-separated values in `WATCH_NAMESPACE`.

**Characteristics:**
- Requires `Role` and `RoleBinding` in each watched namespace
- Uses the same `setupCacheNamespaces` helper function
- Same code as single-namespace mode (KISS principle)

**Example:**

```bash
# Deploy manager to watch multiple namespaces
export WATCH_NAMESPACE=namespace1,namespace2,namespace3
kubectl apply -f dist/install.yaml
```

The `setupCacheNamespaces` helper function automatically handles both single and multiple namespaces without conditional logic.

<aside class="note">
<h1>Example</h1>

The `testdata/project-v4-with-plugins` in the Kubebuilder repository demonstrates a complete namespace-scoped manager configuration.

See: [testdata/project-v4-with-plugins](https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/project-v4-with-plugins)
</aside>

<aside class="warning">

<h1>Webhooks and Namespace-Scoped Mode</h1>

If your project has webhooks, the manager cache is restricted to `WATCH_NAMESPACE`, but webhooks receive requests from all namespaces by default.

**The Problem:**

Your webhook server receives admission requests from all namespaces, but the cache only has data from `WATCH_NAMESPACE`. If a webhook handler queries the cache for an object outside the watched namespaces, the lookup fails.

**Solution:**

Configure `namespaceSelector` or `objectSelector` on your webhooks to align webhook scope with the cache. Currently, controller-gen does not have markers for this. You must add these manually using Kustomize patches.

See the [Webhook Bootstrap Problem](../reference/webhook-bootstrap-problem.html) guide for detailed steps on creating and applying namespace selector patches.

</aside>

## See Also

- [Understanding Scopes](./scopes.md) - Overview of manager and CRD scopes
- [CRD Scope](./crd-scope.md) - Configuring CustomResourceDefinition scope
- [Namespace-Scoped Migration](../migration/namespace-scoped.md) - Detailed implementation guide
- [Project Config](./project-config.md) - PROJECT file configuration
