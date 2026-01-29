# Strict Field Validation

## Overview

By default, when your controller writes an object that contains fields not defined in the CRD schema, the API server:

- Accepts the request
- Drops the unknown fields
- May only log a warning

This can hide bugs and version skew between:
- The controller code (Go types) and
- The CRD schema installed in the cluster

`controller-runtime` exposes [client.WithFieldValidation][client-field-validation-docs] to turn on strict server-side field validation for all client writes. When enabled, the API server returns a hard error instead of silently dropping unknown fields.

CRDs should be installed before controllers. However, during upgrades this can be best effort rather than guaranteed. Deployment tools may apply resources without strict ordering, and even tools with ordering features don't guarantee the CRD update succeeds. For controllers using external CRDs from third-party operators, ordering may not be controllable at all.

When controllers and CRDs get out of sync, controller writes may fail with unknown field errors, status updates may not persist, and conversion webhooks can fail if the CRD schema is outdated. Controllers may crash-loop until CRDs are updated.

## What does it solve

Strict validation prevents silent failures when your controller code and CRD schemas get out of sync.

For example, you add a new field `status.newField` to your controller, but the CRD in the cluster hasn't been updated yet. When the controller calls `client.Status().Patch(...)`:

**Without strict validation:**
- API server drops `status.newField` silently
- Controller sees no error
- Field never appears on the object - confusing debugging

**With strict validation:**
- API server returns clear error
- Controller knows CRDs need updating
- Fails fast instead of silent data loss

## When to use it

Strict validation is a good fit when:

- You own both the CRDs and the controllers
- Your upgrade process applies CRDs first or ensures they update together
- You want to fail fast when a controller writes fields not in the schema
- You want to catch bugs in your types or conversions early
- You use typed schemas, or explicitly mark dynamic data with `x-kubernetes-preserve-unknown-fields: true`

## When NOT to use it

Avoid strict validation in production when:

- Controllers and CRDs upgrade independently (i.e., common in Helm)
- You manage third-party CRDs whose schemas evolve independently
- Your CRDs use unstructured/dynamic data without `x-kubernetes-preserve-unknown-fields`
- You need upgrade tolerance when controller and CRD versions are temporarily mismatched

## How Kubebuilder scaffold handles CRD ordering

### Makefile Targets (`make install` + `make deploy`)

```bash
make install  # Installs CRDs into the cluster
make deploy   # Deploys the controller
```

Two separate commands. CRDs are installed first and established before the controller starts. Order is guaranteed.

### YAML bundle distribution (make build-installer)

```bash
kubectl apply -f dist/install.yaml
```

The bundle positions CRDs early in the file, after Namespace and before Deployment. This works for new installations.

During upgrades, if the CRD already exists and the update fails or is slow, the Deployment may still update. The new controller may start before the CRD update completes.

### Helm chart distribution

By using the [Helm plugin](../plugins/available/helm-v2-alpha.md), you can distribute your solution as a Helm chart package. Users can install or upgrade it with:

```bash
helm install my-operator ./dist/chart
```

Kubebuilder places CRDs in `templates/crd/` to ensure they upgrade with the controller. Helm has a built-in resource order that helps during installation.

During upgrades, if the CRD update fails or is slow to propagate, Helm may still update the Deployment.

Moreover, users can skip CRD updates with `helm upgrade --set crd.enable=false`.

<aside class="note">

Helm recommends a `crds/` directory, but CRDs there **never upgrade**. Kubebuilder uses `templates/crd/` to keep CRDs in sync with controllers but the order is **best effort**. See [Why CRDs are added under templates](../plugins/available/helm-v2-alpha.md#why-crds-are-added-under-templates) for details.

</aside>

<aside class="note">
<h1>Upgrade and Lifecycle Considerations</h1>

During upgrades, CRD updates may not complete before the controller starts. This can happen when deployment tools apply resources without explicit ordering mechanisms.

CRD updates may hit errors or take time to propagate through the API server. If the controller Deployment updates at the same time, the new controller pod may start before the CRD update finishes. With strict validation enabled, controller writes will fail until the CRD is ready.

Ordering tools like ArgoCD sync waves or FluxCD dependsOn can help by controlling when resources are applied. However, these control timing, not whether the CRD update succeeds. If not properly configured or if the CRD apply encounters issues, the controller may still deploy.

For controllers that use CRDs from third-party operators (cert-manager, prometheus-operator, etc.), those CRDs have independent lifecycles. The third-party operator may upgrade its CRDs separately from your controller, which can lead to version mismatches.

</aside>

## Wiring an opt-in flag in cmd/main.go

This feature is **not scaffolded by default**. Follow these steps to add it manually.

### Step 1: Add the strictManager wrapper

In `cmd/main.go`, add this type definition after the `init()` function:

```go
// strictManager wraps the manager to reject unknown fields instead of silently dropping them.
// When the controller writes a field that doesn't exist in the CRD, the write fails immediately.
// This helps catch typos and version mismatches between your code and cluster CRDs.
type strictManager struct {
	ctrl.Manager
	strictClient client.Client
}

func (m *strictManager) GetClient() client.Client {
	return m.strictClient
}
```

### Step 2: Add required imports

Add these imports to `cmd/main.go`:

```go
import (
	// ... your existing imports ...
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)
```

### Step 3: Add the command-line flag

In the `main()` function, where other flags are defined, add:

```go
func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var strictFieldValidation bool  // Add this

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "...")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "...")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "...")

	// Add this flag
	flag.BoolVar(&strictFieldValidation, "strict-field-validation", false,
		"Reject unknown fields instead of dropping them.")

	// ... rest of your code ...
}
```

### Step 4: Wrap the manager conditionally

After creating the manager with `ctrl.NewManager()`, add this wrapper logic:

```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
	Scheme:                 scheme,
	// ... your other options ...
})
if err != nil {
	setupLog.Error(err, "unable to start manager")
	os.Exit(1)
}

// When enabled, the controller rejects writes with unknown fields instead of silently dropping them.
var finalMgr ctrl.Manager = mgr
if strictFieldValidation {
	finalMgr = &strictManager{
		Manager: mgr,
		strictClient: client.WithFieldValidation(
			mgr.GetClient(),
			metav1.FieldValidationStrict,
		),
	}
}

// Use finalMgr for all subsequent setup
if err := (&controller.MyReconciler{
	Client: finalMgr.GetClient(),
	Scheme: finalMgr.GetScheme(),
}).SetupWithManager(finalMgr); err != nil {
	setupLog.Error(err, "unable to create controller", "controller", "My")
	os.Exit(1)
}

// Continue using finalMgr for health checks, starting manager, etc.
if err := finalMgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
	setupLog.Error(err, "unable to set up health check")
	os.Exit(1)
}

if err := finalMgr.Start(ctrl.SetupSignalHandler()); err != nil {
	setupLog.Error(err, "problem running manager")
	os.Exit(1)
}
```

[client-field-validation-docs]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client#WithFieldValidation
