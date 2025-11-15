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

We **do not enable this by default** in scaffolds because it can be too aggressive during upgrades. Instead, we show how to wire it as an opt-in flag.

## What does it solve

Strict validation prevents silent failures when your controller code and CRD schemas get out of sync.

For example, you add a new field `status.newField` to your controller, but the CRD in the cluster hasn't been updated yet. When the controller calls `client.Status().Patch(...)`:

**Without strict validation:**
- API server drops `status.newField` silently
- Controller sees no error
- Field never appears on the object → confusing debugging

**With strict validation:**
- API server returns clear error
- Controller knows CRDs need updating
- Fails fast instead of silent data loss

<aside class="warning">
<h1>Not included in default scaffold</h1>

This feature is **not scaffolded by default** because it requires careful deployment coordination.

**The problem:** Standard deployment tools (`make deploy`, `helm install`) apply CRDs and controller simultaneously with no ordering guarantees. When strict validation is enabled and the controller starts before CRDs finish updating, **all writes fail** until manual intervention.

**The solution:** You need external tooling (separate Helm charts, CI/CD pipeline stages, custom scripts) to ensure CRDs are upgraded and established before the controller starts.

</aside>

## Upgrade scenario example

Consider a common upgrade scenario: you deploy a new controller version that adds `status.newField` to your types, but the CRD in the cluster hasn't been updated yet. When the controller tries to write the new field:

```go
if err := r.Status().Patch(ctx, foo, patch); err != nil {
    // handle error
}
```

**Without strict validation**, the API server accepts the request but silently drops `status.newField`. The controller sees no error, but the field never appears on the object. This makes debugging difficult because there's no indication that something went wrong.

**With strict validation**, the API server rejects the request with a 400 BadRequest error. The controller receives a clear error indicating a CRD–controller mismatch, making it obvious what needs to be fixed.

While this catches bugs fast, it also means any write operation will fail when a new controller runs against old CRDs. The failures continue until the CRDs are updated. That's why strict validation is **off by default**.

## How strict field validation works

`controller-runtime` lets you wrap a client:

```go
strictClient := client.WithFieldValidation(
    baseClient,
    metav1.FieldValidationStrict,
)
```

All write operations (Create, Update, Patch) from `strictClient` send `fieldValidation=strict` to the API server.

The API server:
- Returns an error when the payload has unknown or invalid fields
- Does not perform the write

You can still override per call:

```go
cli.Create(ctx, obj, client.FieldValidation(metav1.FieldValidationWarn))
```


## When to use it

Strict validation is a good fit when:

- You own both the CRDs and the controllers
- Your upgrade process applies CRDs first or ensures they update together
- You want to fail fast when a controller writes fields not in the schema
- You want to catch bugs in your types or conversions early
- You use typed schemas, or explicitly mark dynamic data with `x-kubernetes-preserve-unknown-fields: true`

This works well in development, CI, and production environments where you control the deployment order.


## When NOT to use it

Avoid strict validation in production when:

- Controllers and CRDs upgrade independently (common in Helm, OLM deployments)
- You manage third-party CRDs whose schemas evolve independently
- Your CRDs use unstructured/dynamic data without `x-kubernetes-preserve-unknown-fields`
- You need upgrade tolerance when controller and CRD versions are temporarily mismatched

In these scenarios, strict validation causes BadRequest errors during upgrades. That's why it's:
- **Off by default** in scaffolds
- **Opt-in via flag** for those who need it

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
		"Reject unknown fields instead of dropping them. Useful for dev/CI, and production if CRDs upgrade before controllers.")

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
// This is useful for catching bugs in development, but causes problems in production when you upgrade
// the controller before the CRDs - all writes will fail until CRDs are updated.
//
// Safe for: development, CI, and production only with external tooling to ensure CRDs upgrade first.
// Not safe for: make deploy, helm install, or when you apply everything at once. The scaffolded project
// has no built-in mechanism to ensure CRDs upgrade before the controller - you need external solutions.
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

<aside class="note">
<h1>Important: Use finalMgr everywhere</h1>

After wrapping the manager, use `finalMgr` instead of `mgr` for:
- Controller setup
- Webhook setup
- Health checks
- Starting the manager

This ensures all components use the wrapped client with strict validation.

</aside>

[client-field-validation-docs]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client#WithFieldValidation