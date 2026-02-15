# Migrating to Namespace-Scoped Manager

This guide covers converting **existing cluster-scoped projects** to namespace-scoped deployment.

<aside class="note">
<h1>Creating New Namespace-Scoped Projects</h1>

If you're creating a **new project**, simply use:

```bash
kubebuilder init --domain example.com --namespaced
```

All files including `cmd/main.go` and RBAC configurations will be scaffolded correctly. All controllers created with `kubebuilder create api` will automatically have the `namespace=` parameter in their RBAC markers. No manual changes or migration steps are needed.
</aside>

By default, Kubebuilder scaffolds cluster-scoped managers that watch and manage resources across all namespaces. This guide shows how to convert an existing cluster-scoped project to namespace-scoped deployment, limiting the manager to watch only specific namespace(s).

## When to Use Namespace-Scoped

**Use namespace-scoped when:**
- Building tenant-specific managers in multi-tenant clusters
- Security policies require least-privilege (no cluster-wide permissions)
- Need multiple manager instances in different namespaces
- Managing only namespace-scoped resources (Deployments, Services, ConfigMaps, etc.)

**Use cluster-scoped (default) when:**
- Managing cluster-scoped resources (Nodes, ClusterRoles, Namespaces, etc.)
- Single manager instance managing resources across all namespaces

<aside class="note">
<h1>AI-Assisted Migration</h1>

This migration involves updating RBAC markers across multiple controller files. If you're using an AI coding assistant, see the [AI-Assisted Migration](#ai-assisted-migration) section for ready-to-use instructions.

</aside>

## Migration Steps

**Quick Summary:**
1. Run `kubebuilder edit --namespaced --force` - scaffolds Role/RoleBinding and updates manager.yaml
2. Update cmd/main.go to configure namespace-scoped cache
3. Add `namespace=` parameter to RBAC markers in existing controller files
4. Run `make manifests` - regenerate RBAC from updated markers
5. Verify and deploy

<aside class="warning">
<h2>Manual Steps Required</h2>

The `edit` command scaffolds RBAC files and updates manager.yaml automatically (with `--force`), but **cannot** update existing controller files or cmd/main.go.

You must manually update:
- cmd/main.go namespace-scoped cache configuration
- Existing controller RBAC markers

**Note:** New controllers created after enabling namespaced mode will have correct RBAC markers automatically.
</aside>

## Detailed Steps:

### 1. Enable namespace-scoped mode

```bash
kubebuilder edit --namespaced --force
```

This command automatically:
- Sets `namespaced: true` in your PROJECT file
- Scaffolds `config/rbac/role.yaml` with `kind: Role` (namespace-scoped)
- Scaffolds `config/rbac/role_binding.yaml` with `kind: RoleBinding`
- Regenerates `config/manager/manager.yaml` with WATCH_NAMESPACE environment variable
- Regenerates admin/editor/viewer roles with `kind: Role` (namespace-scoped) for all existing APIs

**Note:** The `--force` flag regenerates config/manager/manager.yaml. Without `--force`, you must manually add WATCH_NAMESPACE (see below).

### 2. Update cmd/main.go (Required Manual Step)

The edit command cannot update cmd/main.go automatically. You must manually add namespace-scoped configuration.

**a. Add import:**
```go
import (
    // ... existing imports ...
    "sigs.k8s.io/controller-runtime/pkg/cache"
)
```

**b. Add helper functions (after `init()` and before `main()`):**
```go
// getWatchNamespace returns the namespace(s) the manager should watch for changes.
// It reads the value from the WATCH_NAMESPACE environment variable.
func getWatchNamespace() (string, error) {
    watchNamespaceEnvVar := "WATCH_NAMESPACE"
    ns, found := os.LookupEnv(watchNamespaceEnvVar)
    if !found {
        return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
    }
    return ns, nil
}

// setupCacheNamespaces configures the cache to watch specific namespace(s).
func setupCacheNamespaces(namespaces string) cache.Options {
    defaultNamespaces := make(map[string]cache.Config)
    for _, ns := range strings.Split(namespaces, ",") {
        defaultNamespaces[strings.TrimSpace(ns)] = cache.Config{}
    }
    return cache.Options{
        DefaultNamespaces: defaultNamespaces,
    }
}
```

**c. In `main()` function, before `ctrl.NewManager()`, add:**
```go
// Get the namespace(s) for namespace-scoped mode from WATCH_NAMESPACE environment variable.
watchNamespace, err := getWatchNamespace()
if err != nil {
    setupLog.Error(err, "Unable to get WATCH_NAMESPACE")
    os.Exit(1)
}
```

**d. Update manager creation to use namespace-scoped cache:**
```go
mgrOptions := ctrl.Options{
    Scheme:                 scheme,
    Metrics:                metricsServerOptions,
    WebhookServer:          webhookServer,
    HealthProbeBindAddress: probeAddr,
    LeaderElection:         enableLeaderElection,
    LeaderElectionID:       "your-leader-election-id",
    // ... other existing options ...
}

// Configure cache to watch namespace(s) specified in WATCH_NAMESPACE
mgrOptions.Cache = setupCacheNamespaces(watchNamespace)
setupLog.Info("Watching namespace(s)", "namespaces", watchNamespace)

mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
if err != nil {
    setupLog.Error(err, "Failed to start manager")
    os.Exit(1)
}
```

<aside class="note">
<h1>If You Didn't Use --force</h1>

If you ran `kubebuilder edit --namespaced` without `--force`, manually add WATCH_NAMESPACE to `config/manager/manager.yaml`:

```yaml
spec:
  template:
    spec:
      containers:
      - name: manager
        env:
        - name: WATCH_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
```

With `--force`, this is done automatically. Skip if you used `--force`.
</aside>

### 3. Update RBAC markers in existing controllers

For each **existing controller file**, add the `namespace=` parameter to RBAC markers.

**Find controller files:**
- Look for files containing `func (r *SomeReconciler) Reconcile(`
- Common locations: `internal/controller/*_controller.go`

In `internal/controller/cronjob_controller.go`:

**Before (cluster-scoped):**
```go
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
```

**After (namespace-scoped):**
```go
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,namespace=<project-name>-system,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,namespace=<project-name>-system,resources=cronjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,namespace=<project-name>-system,resources=cronjobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
```

Replace `project-system` with your namespace (found in `config/default/kustomization.yaml` under the `namespace:` field).

<aside class="note">
<h1>New Controllers Get This Automatically</h1>

After running `kubebuilder edit --namespaced --force`, any new controllers created will automatically have the `namespace=` parameter:

```bash
kubebuilder create api --group myapp --version v1 --kind MyNewKind --controller=true --resource=true
```

Generated controller will include:
```go
// +kubebuilder:rbac:groups=myapp.example.com,namespace=<project-name>-system,resources=mynewkinds,verbs=...
```

Only existing controllers need manual updates!
</aside>

### 4. Regenerate RBAC manifests

After updating RBAC markers in Step 3, regenerate the RBAC manifests:

```bash
make manifests      # Regenerate RBAC from updated controller markers
```

Verify the generated files show `kind: Role` instead of `kind: ClusterRole`:

**config/rbac/role.yaml:**
```yaml
kind: Role
metadata:
  name: manager-role
  # Note: namespace is added by kustomize during build, not in source
```

**config/rbac/*_editor_role.yaml, *_viewer_role.yaml, *_admin_role.yaml:**
```yaml
kind: Role
metadata:
  name: cronjob-editor-role
  # Note: namespace is added by kustomize during build, not in source
```

<aside class="note">
<h1>Metrics Auth Role Stays Cluster-Scoped</h1>

The `config/rbac/metrics_auth_role.yaml` will remain `kind: ClusterRole` - this is correct. The metrics authentication uses cluster-scoped APIs (TokenReview, SubjectAccessReview) and must stay cluster-scoped even in namespace-scoped projects.

</aside>

### 5. Verify and deploy

Run tests to verify everything works:

```bash
make generate       # Regenerate code
make test           # Run tests
```

Deploy and verify:

```bash
make deploy IMG=<your-image>

# Verify RBAC is namespace-scoped (not cluster-scoped)
kubectl get role,rolebinding -n <manager-namespace>

# Test: Create a resource in the manager's namespace - should be reconciled
kubectl apply -f config/samples/ -n <manager-namespace>

# Test: Create a resource in a different namespace - should NOT be reconciled
kubectl apply -f config/samples/ -n other-namespace
```

<aside class="warning">

<h1>Webhooks and Namespace-Scoped Mode</h1>

If your project has webhooks, the manager cache is restricted to `WATCH_NAMESPACE`, but webhooks receive requests from all namespaces by default.

**The Problem:**

Your webhook server receives admission requests from all namespaces, but the cache only has data from `WATCH_NAMESPACE`. If a webhook handler queries the cache for an object outside the watched namespaces, the lookup fails.

**Solution:**

Configure `namespaceSelector` or `objectSelector` on your webhooks to align webhook scope with the cache. Currently, controller-gen does not have markers for this. You must add these manually using Kustomize patches.

See the [Webhook Bootstrap Problem](../reference/webhook-bootstrap-problem.html) guide for detailed steps on creating and applying namespace selector patches.

</aside>

## AI-Assisted Migration

If you're using an AI coding assistant (Cursor, GitHub Copilot, etc.), you can automate the manual migration steps.

<aside class="note">

<h1>AI Migration Instructions</h1>

**Instructions to provide to your AI assistant:**

```
I need to migrate this Kubebuilder project from cluster-scoped to namespace-scoped.

First, get the namespace value:
- Read config/default/kustomization.yaml and find the "namespace:" field
- Use that value for all namespace= parameters in RBAC markers

Context:
By default, Kubebuilder projects are cluster-scoped. Namespace-scoped projects watch only
specific namespace(s) via the WATCH_NAMESPACE environment variable.

References:
- Kubebuilder Book: https://book.kubebuilder.io/reference/manager-scope.html

Steps to execute:

1. Enable namespace-scoped mode:
   Run: kubebuilder edit --namespaced

   This automatically:
   - Updates PROJECT file with namespaced: true
   - Scaffolds Role/RoleBinding (instead of ClusterRole/ClusterRoleBinding)
   - Regenerates admin/editor/viewer roles with kind: Role

2. Add WATCH_NAMESPACE to config/manager/manager.yaml:

   Find the manager container under spec.template.spec.containers (name: manager)
   and add the env section:

   spec:
     template:
       spec:
         containers:
         - name: manager
           env:
           - name: WATCH_NAMESPACE
             valueFrom:
               fieldRef:
                 fieldPath: metadata.namespace

3. Update cmd/main.go:

   a. Add import:

   import (
       // ... existing imports ...
       "sigs.k8s.io/controller-runtime/pkg/cache"
   )

   b. Add these two helper functions after init() and before main():

   // getWatchNamespace returns the namespace(s) the manager should watch for changes.
   // It reads the value from the WATCH_NAMESPACE environment variable.
   func getWatchNamespace() (string, error) {
       watchNamespaceEnvVar := "WATCH_NAMESPACE"
       ns, found := os.LookupEnv(watchNamespaceEnvVar)
       if !found {
           return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
       }
       return ns, nil
   }

   // setupCacheNamespaces configures the cache to watch specific namespace(s).
   func setupCacheNamespaces(namespaces string) cache.Options {
       defaultNamespaces := make(map[string]cache.Config)
       for _, ns := range strings.Split(namespaces, ",") {
           defaultNamespaces[strings.TrimSpace(ns)] = cache.Config{}
       }
       return cache.Options{
           DefaultNamespaces: defaultNamespaces,
       }
   }

   c. In main() function, find ctrl.SetLogger() and add right after it:

   // Get the namespace(s) for namespace-scoped mode from WATCH_NAMESPACE environment variable.
   watchNamespace, err := getWatchNamespace()
   if err != nil {
       setupLog.Error(err, "Unable to get WATCH_NAMESPACE")
       os.Exit(1)
   }

   d. Find the ctrl.NewManager() call and replace it with:

   mgrOptions := ctrl.Options{
       Scheme:                 scheme,
       Metrics:                metricsServerOptions,
       WebhookServer:          webhookServer,
       HealthProbeBindAddress: probeAddr,
       LeaderElection:         enableLeaderElection,
       LeaderElectionID:       "your-leader-election-id",
       // ... keep all other existing options from the original ctrl.NewManager call ...
   }

   // Configure cache to watch namespace(s) specified in WATCH_NAMESPACE
   mgrOptions.Cache = setupCacheNamespaces(watchNamespace)
   setupLog.Info("Watching namespace(s)", "namespaces", watchNamespace)

   mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
   if err != nil {
       setupLog.Error(err, "Failed to start manager")
       os.Exit(1)
   }

5. Update RBAC markers in existing controller files:

   Important: Only update RBAC markers in controller files (files containing "Reconcile" function).
   Do not modify webhook files (files in internal/webhook/ or api/*/webhook.go).

   How to find controller files in this project:
   - Search for all Go files containing "func (r *" and "Reconcile("
   - Common locations: internal/controller/, internal/controller/*/, controllers/
   - File pattern: *_controller.go (but verify by checking for Reconcile function)

   For EACH controller file found:
   - Locate ALL +kubebuilder:rbac markers in that file
   - Add namespace=<value-from-kustomization> parameter to each marker

   Example transformation:

   Before:
   // +kubebuilder:rbac:groups=myapp.example.com,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
   // +kubebuilder:rbac:groups=myapp.example.com,resources=mykinds/status,verbs=get;update;patch
   // +kubebuilder:rbac:groups=myapp.example.com,resources=mykinds/finalizers,verbs=update
   // +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
   // +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

   After:
   // +kubebuilder:rbac:groups=myapp.example.com,namespace=<value-from-kustomization>,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
   // +kubebuilder:rbac:groups=myapp.example.com,namespace=<value-from-kustomization>,resources=mykinds/status,verbs=get;update;patch
   // +kubebuilder:rbac:groups=myapp.example.com,namespace=<value-from-kustomization>,resources=mykinds/finalizers,verbs=update
   // +kubebuilder:rbac:groups=core,namespace=<value-from-kustomization>,resources=events,verbs=create;patch
   // +kubebuilder:rbac:groups=apps,namespace=<value-from-kustomization>,resources=deployments,verbs=get;list;watch;create;update;patch;delete

   Important rules:
   - Add namespace= after the groups= parameter
   - Use the namespace value from config/default/kustomization.yaml
   - Update all +kubebuilder:rbac markers in each controller file
   - Do not modify webhook files - webhooks use certificate-based auth, not RBAC
   - Do not add namespace= to metrics-auth-role markers (those stay cluster-scoped)

6. Regenerate RBAC manifests:
   Run: make manifests

   This regenerates config/rbac/role.yaml from the updated controller markers.
   Verify it shows kind: Role (not ClusterRole).

7. Verify the migration:
   Run: make generate

   Verify files were updated correctly:
   - config/rbac/role.yaml - should be kind: Role
   - config/manager/manager.yaml - should have WATCH_NAMESPACE env var
   - cmd/main.go - should have getWatchNamespace() and setupCacheNamespaces() functions
   - All controller files - should have namespace= in RBAC markers

Done! After this migration:
- The project is now namespace-scoped
- Existing controllers have been updated with namespace= RBAC markers
- Future controllers created with `kubebuilder create api` will automatically include
  namespace= in their RBAC markers - no manual updates needed!

```

</aside>


## Multi-Namespace Support

The `WATCH_NAMESPACE` environment variable supports comma-separated values to watch multiple specific namespaces:

```yaml
env:
- name: WATCH_NAMESPACE
  value: "namespace-1,namespace-2,namespace-3"
```

Note: You'll need to create Role/RoleBinding in each namespace for proper RBAC.

## Reverting to Cluster-Scoped

To revert back to cluster-scoped:

```bash
kubebuilder edit --namespaced=false --force
```

This command automatically:
- Sets `namespaced: false` in your PROJECT file
- Scaffolds `config/rbac/role.yaml` with `kind: ClusterRole`
- Scaffolds `config/rbac/role_binding.yaml` with `kind: ClusterRoleBinding`
- With `--force`: Regenerates `config/manager/manager.yaml` without WATCH_NAMESPACE env var

**Manual steps required:**
1. Remove `namespace=` parameter from RBAC markers in all controller files
2. Run `make manifests` to regenerate cluster-scoped RBAC
3. Remove namespace-scoped code from `cmd/main.go`:
   - Remove `getWatchNamespace()` function
   - Remove `setupCacheNamespaces()` function
   - Remove namespace retrieval and cache configuration
   - Remove added imports (`fmt`, `strings`, `cache`) if not used elsewhere
4. If you didn't use `--force`, manually remove `WATCH_NAMESPACE` from `config/manager/manager.yaml`

## Important Notes

- **Only controllers need RBAC updates**: Only update `+kubebuilder:rbac` markers in controller files (files with `Reconcile` function). Webhook files do NOT use RBAC markers - webhooks use certificate-based authentication with the API server.
- **RBAC markers control scope**: The `namespace=` parameter in controller RBAC markers determines whether controller-gen generates `Role` (namespace-scoped) or `ClusterRole` (cluster-scoped). Without the `namespace=` parameter, controller-gen always generates `ClusterRole`.
- **Controller-gen regenerates role.yaml**: After running `make manifests`, controller-gen will regenerate `config/rbac/role.yaml` based on your controller RBAC markers. The initial `Role` scaffold from `kubebuilder edit --namespaced=true` serves as a template, but controller-gen manages the actual content.
- **Namespace parameter format**: Use `namespace=<your-namespace>` in controller RBAC markers, typically `namespace=<project-name>-system` to match your deployment namespace.
- **Metrics auth role stays cluster-scoped**: The `metrics-auth-role` uses cluster-scoped APIs (TokenReview, SubjectAccessReview) and correctly remains a ClusterRole without namespace parameter.
- **Webhooks require manual configuration**: Currently, controller-gen does not support `namespaceSelector` or `objectSelector` markers for webhooks. See the webhook section above for details.

## See Also

- [Manager Scope](../reference/manager-scope.md) - Detailed explanation of manager scope concepts
- [Project Config](../reference/project-config.md) - PROJECT file configuration reference
