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
1. Run `kubebuilder edit --namespaced=true` - scaffolds Role/RoleBinding
2. Add `namespace=` parameter to RBAC markers in existing controller files
3. Update `cmd/main.go` to configure namespace watching
4. Update `config/manager/manager.yaml` to add WATCH_NAMESPACE env var
5. Run `make manifests` - regenerate RBAC from updated markers
6. Verify and deploy

**Detailed Steps:**

### 1. Enable namespace-scoped mode

```bash
kubebuilder edit --namespaced=true
```

This command automatically:
- Sets `namespaced: true` in your PROJECT file
- Scaffolds `config/rbac/role.yaml` with `kind: Role` (namespace-scoped)
- Scaffolds `config/rbac/role_binding.yaml` with `kind: RoleBinding`
- Regenerates admin/editor/viewer roles with `kind: Role` (namespace-scoped) for all existing APIs

**Note:** For existing projects, you must manually update `cmd/main.go` and `config/manager/manager.yaml`. Projects created with `kubebuilder init --namespaced` have these files pre-configured.

<aside class="warning">
<h1>Manual Steps Required</h1>

The `edit` command cannot automatically update **existing controller RBAC markers**. You must manually add the `namespace=` parameter to RBAC markers in all existing controller files as shown in Step 2.

**Important:**
- **New controllers are automatic**: Any new APIs created with `kubebuilder create api` after running `edit --namespaced=true` will automatically have the `namespace=` parameter in their controller RBAC markers. No manual updates needed.
- **New projects are automatic**: Projects created with `kubebuilder init --namespaced` from the start don't require any manual updates at all.
- **Existing controllers need manual updates**: Only controllers that existed before running `edit --namespaced=true` need manual RBAC marker updates.
</aside>

### 2. Update RBAC markers in controllers (Required Manual Step)

For each **controller file** in your project (files containing the `Reconcile` function), update the RBAC markers to include the `namespace=` parameter. This tells controller-gen to generate namespace-scoped `Role` resources instead of cluster-scoped `ClusterRole`.

<aside class="warning">
<h1>Controllers Only - Not Webhooks</h1>

**Only update RBAC markers in controller files.** Do NOT modify webhook files.

- **Controllers** are in files like `internal/controller/*_controller.go` and contain the `Reconcile` function
- **Webhooks** are in files like `internal/webhook/*` or `api/*/webhook.go` and do NOT need RBAC changes (webhooks use certificate-based authentication, not RBAC)

</aside>

**How to find controller files:**
- Look for files containing `func (r *SomeReconciler) Reconcile(`
- Common locations: `internal/controller/`, `internal/controller/*/`, or custom paths in multi-group layouts
- File pattern typically matches `*_controller.go`

**Before (cluster-scoped):**
```go
// In a controller file (e.g., internal/controller/mykind_controller.go)
// +kubebuilder:rbac:groups=myapp.example.com,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=myapp.example.com,resources=mykinds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=myapp.example.com,resources=mykinds/finalizers,verbs=update
```

**After (namespace-scoped):**
```go
// In a controller file (e.g., internal/controller/mykind_controller.go)
// +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mykinds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mykinds/finalizers,verbs=update
```

Replace `myproject-system` with your actual namespace name (usually `<project-name>-system`).

After updating the markers in all controller files, run `make manifests` to regenerate `config/rbac/role.yaml`. You should see `kind: Role` instead of `kind: ClusterRole` (the namespace field is added by kustomize during build, not in the source YAML).

<aside class="note">
<h1>Creating New APIs After Migration</h1>

Once you've run `kubebuilder edit --namespaced=true`, any new controllers created will automatically have namespace-scoped RBAC:

```bash
# After enabling namespace-scoped mode
kubebuilder create api --group myapp --version v1 --kind MyNewKind --controller=true --resource=true
```

The generated controller will automatically include:
```go
// +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mynewkinds,verbs=...
```

No manual RBAC marker updates needed for new controllers!
</aside>

### 3. Update config/manager/manager.yaml

Add the `WATCH_NAMESPACE` environment variable to `config/manager/manager.yaml`:

```yaml
spec:
  template:
    spec:
      containers:
      - name: manager
        image: controller:latest
        env:
        - name: WATCH_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
```

This uses the Kubernetes downward API to inject the namespace where the manager pod is running.

### 4. Update cmd/main.go

Manually add namespace-scoped configuration to `cmd/main.go`:

**Add imports:**
```go
import (
    // ... existing imports ...
    "fmt"
    "strings"
    "sigs.k8s.io/controller-runtime/pkg/cache"
)
```

**Add helper functions (after `init()` and before `main()`):**
```go
// getWatchNamespace returns the namespace(s) the manager should watch.
func getWatchNamespace() (string, error) {
    watchNamespaceEnvVar := "WATCH_NAMESPACE"
    ns, found := os.LookupEnv(watchNamespaceEnvVar)
    if !found {
        return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
    }
    return ns, nil
}

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
```

**Add namespace retrieval in `main()` function (before creating the manager):**
```go
func main() {
    // ... existing flag parsing and setup ...

    ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

    // Get the namespace(s) to watch from WATCH_NAMESPACE environment variable
    watchNamespace, err := getWatchNamespace()
    if err != nil {
        setupLog.Error(err, "Unable to get WATCH_NAMESPACE")
        os.Exit(1)
    }

    // ... continue with manager creation ...
}
```

**Update manager configuration to watch namespace(s):**
```go
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
if err != nil {
    setupLog.Error(err, "Failed to start manager")
    os.Exit(1)
}
```

This simplified configuration works for both single-namespace and multi-namespace scenarios without conditional logic.

### 5. Regenerate admin/editor/viewer roles

After updating RBAC markers in controllers and running `make manifests`, verify the generated admin/editor/viewer roles in `config/rbac/*_editor_role.yaml`, `*_viewer_role.yaml`, and `*_admin_role.yaml`:

**Before (cluster-scoped):**
```yaml
kind: ClusterRole
metadata:
  name: mykind-editor-role
```

**After (namespace-scoped):**
```yaml
kind: Role
metadata:
  name: mykind-editor-role
  # Note: namespace is added by kustomize during build, not in source
```

These roles allow cluster admins to grant namespace-scoped permissions to users for managing your custom resources.

### 6. Regenerate RBAC and verify

After updating the RBAC markers in step 2, regenerate the RBAC manifests:

```bash
make manifests      # Regenerate RBAC from updated controller markers
make generate       # Regenerate code
make test           # Run tests
```

Verify that `config/rbac/role.yaml` shows `kind: Role` (the namespace is added by kustomize during build).

### 7. Deploy and test

```bash
make deploy IMG=<your-image>

# Verify RBAC is namespace-scoped (not cluster-scoped)
kubectl get role,rolebinding -n <manager-namespace>

# Test: Create a resource in the manager's namespace - should be reconciled
kubectl apply -f config/samples/ -n <manager-namespace>

# Test: Create a resource in a different namespace - should NOT be reconciled
kubectl apply -f config/samples/ -n other-namespace
```

## AI-Assisted Migration

If you're using an AI coding assistant (Cursor, GitHub Copilot, etc.), you can automate the manual migration steps.

<aside class="note">

<h1>AI Migration Instructions</h1>

**Prerequisites:**
1. Get your project name from the `PROJECT` file (field under `projectName:`)
2. Find controller files: Search for `*_controller.go` files in your project (typically in `internal/controller/` or subdirectories)

**Instructions to provide to your AI assistant:**

Give your AI assistant these instructions, replacing the project name:

```
I need to migrate this Kubebuilder project from cluster-scoped to namespace-scoped.

Project details:
- Project name: myproject
- Namespace: myproject-system

Context:
By default, Kubebuilder projects are cluster-scoped (watch all namespaces). Namespace-scoped
projects watch only specific namespace(s) via the WATCH_NAMESPACE environment variable,
providing better security isolation for multi-tenant environments.

The `kubebuilder edit --namespaced=true` command scaffolds most files automatically, but cannot
update existing controller RBAC markers. This migration focuses on updating RBAC markers in
CONTROLLER files only.

References:
- Kubebuilder Book: https://book.kubebuilder.io/reference/manager-scope.html

Steps to execute:

1. Enable namespace-scoped mode (scaffolds most files automatically):
   Run: kubebuilder edit --namespaced=true

   This automatically:
   - Updates PROJECT file with namespaced: true
   - Scaffolds Role/RoleBinding (instead of ClusterRole/ClusterRoleBinding)
   - Adds helper functions to cmd/main.go (getWatchNamespace, setupCacheNamespaces)
   - Adds WATCH_NAMESPACE env var to config/manager/manager.yaml
   - Regenerates admin/editor/viewer roles with kind: Role

2. Update RBAC markers in controller files:

   Important: Only update RBAC markers in controller files (files containing "Reconcile" function).
   Do not modify webhook files (files in internal/webhook/ or api/*/webhook.go).

   How to find controller files in this project:
   - Search for all Go files containing "func (r *" and "Reconcile("
   - Common locations: internal/controller/, internal/controller/*/, controllers/
   - File pattern: *_controller.go (but verify by checking for Reconcile function)

   For EACH controller file found:
   - Locate ALL +kubebuilder:rbac markers in that file
   - Add namespace=myproject-system parameter to each marker

   Example transformation:

   Before:
   ```go
   // +kubebuilder:rbac:groups=myapp.example.com,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
   // +kubebuilder:rbac:groups=myapp.example.com,resources=mykinds/status,verbs=get;update;patch
   // +kubebuilder:rbac:groups=myapp.example.com,resources=mykinds/finalizers,verbs=update
   // +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
   // +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
   ```

   After:
   ```go
   // +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
   // +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mykinds/status,verbs=get;update;patch
   // +kubebuilder:rbac:groups=myapp.example.com,namespace=myproject-system,resources=mykinds/finalizers,verbs=update
   // +kubebuilder:rbac:groups=core,namespace=myproject-system,resources=events,verbs=create;patch
   // +kubebuilder:rbac:groups=apps,namespace=myproject-system,resources=deployments,verbs=get;list;watch;create;update;patch;delete
   ```

   Important rules:
   - Add namespace= after the groups= parameter
   - Use the exact namespace: myproject-system
   - Update all +kubebuilder:rbac markers in each controller file
   - Do not modify webhook files - webhooks use certificate-based auth, not RBAC
   - Do not add namespace= to metrics-auth-role markers (those stay cluster-scoped)

3. Regenerate RBAC manifests:
   Run: make manifests

   This regenerates config/rbac/role.yaml from the updated markers.
   Verify it shows kind: Role (not ClusterRole).
   Note: The namespace field is NOT in the source YAML - kustomize adds it during build.

4. Verify the migration:
   Run: make generate && make test

   All tests should pass with the namespace-scoped configuration.

5. Review generated files:
   - config/rbac/role.yaml - should be kind: Role (namespace added by kustomize during build)
   - config/manager/manager.yaml - should have WATCH_NAMESPACE env var
   - cmd/main.go - should have getWatchNamespace() and setupCacheNamespaces() functions

Done! The project is now namespace-scoped and will only watch the namespace
specified in WATCH_NAMESPACE (defaults to the pod's namespace via downward API).
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
kubebuilder edit --namespaced=false
```

This command:
- Sets `namespaced: false` in your PROJECT file
- Scaffolds `config/rbac/role.yaml` with `kind: ClusterRole`
- Scaffolds `config/rbac/role_binding.yaml` with `kind: ClusterRoleBinding`

**Manual steps required:**
1. Remove `namespace=` parameter from RBAC markers in all controller files
2. Run `make manifests` to regenerate cluster-scoped RBAC
3. Remove namespace-scoped code from `cmd/main.go`:
   - Remove `getWatchNamespace()` function
   - Remove `setupCacheNamespaces()` function
   - Remove namespace retrieval and cache configuration
   - Remove added imports (`fmt`, `strings`, `cache`) if not used elsewhere
4. Remove `WATCH_NAMESPACE` environment variable from `config/manager/manager.yaml`

## Important Notes

- **Only controllers need RBAC updates**: Only update `+kubebuilder:rbac` markers in controller files (files with `Reconcile` function). Webhook files do NOT use RBAC markers - webhooks use certificate-based authentication with the API server.
- **Webhooks remain cluster-scoped**: `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` are cluster-scoped resources that validate/mutate CRs in all namespaces. This is correct - webhooks enforce schema consistency across the cluster, while controllers (namespace-scoped) only reconcile resources in their watched namespace(s).
- **RBAC markers control scope**: The `namespace=` parameter in controller RBAC markers determines whether controller-gen generates `Role` (namespace-scoped) or `ClusterRole` (cluster-scoped). Without the `namespace=` parameter, controller-gen always generates `ClusterRole`.
- **Controller-gen regenerates role.yaml**: After running `make manifests`, controller-gen will regenerate `config/rbac/role.yaml` based on your controller RBAC markers. The initial `Role` scaffold from `kubebuilder edit --namespaced=true` serves as a template, but controller-gen manages the actual content.
- **Namespace parameter format**: Use `namespace=<your-namespace>` in controller RBAC markers, typically `namespace=<project-name>-system` to match your deployment namespace.
- **Metrics auth role stays cluster-scoped**: The `metrics-auth-role` uses cluster-scoped APIs (TokenReview, SubjectAccessReview) and correctly remains a ClusterRole without namespace parameter.

## See Also

- [Manager Scope](../reference/manager-scope.md) - Detailed explanation of manager scope concepts
- [Project Config](../reference/project-config.md) - PROJECT file configuration reference
