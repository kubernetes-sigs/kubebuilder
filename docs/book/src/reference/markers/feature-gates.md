# Feature Gates

Feature gates allow you to enable or disable experimental features in your Kubebuilder controllers. This is similar to how Kubernetes core uses feature gates to manage experimental functionality.

## Quick Start

### Marking Fields

```go
type MyResourceSpec struct {
    // Standard field
    Name string `json:"name"`

    // Experimental field
    // +feature-gate experimental-bar
    // +optional
    Bar *string `json:"bar,omitempty"`
}
```

### Running with Feature Gates

```bash
# Enable feature gates
./manager --feature-gates=experimental-bar=true

# Multiple gates
./manager --feature-gates=experimental-bar=true,advanced-features=false

# All gates enabled (useful for testing)
./manager --feature-gates=experimental-bar=true,advanced-features=true
```

## Overview

Feature gates provide a mechanism to:
- Control the availability of experimental features
- Enable gradual rollout of new functionality
- Maintain backward compatibility during API evolution
- Test experimental functionality safely in production environments

## Usage

### Marking Fields with Feature Gates

Use the `+feature-gate` marker to mark experimental fields in your API types:

```go
type MyResourceSpec struct {
    // Standard field - always available
    Name string `json:"name"`

    // Experimental field that requires the "experimental-bar" feature gate
    // +feature-gate experimental-bar
    // +optional
    Bar *string `json:"bar,omitempty"`

    // Another experimental field with different gate
    // +feature-gate advanced-features
    // This field enables advanced processing capabilities
    // +optional
    AdvancedConfig *AdvancedConfig `json:"advancedConfig,omitempty"`
}
```

### Feature Gate Validation

Feature gates are validated for proper format:
- **Valid**: `experimental-bar=true,advanced-features=false`
- **Valid**: `experimental-bar=true` (short form)
- **Invalid**: `experimental-bar=maybe` (only 'true' and 'false' are allowed)
- **Invalid**: `ExperimentalBar=true` (should use kebab-case)

### Automated Discovery

Kubebuilder automatically discovers feature gates from your API types during scaffolding:

1. **During `kubebuilder create api`**: Scans for existing `+feature-gate` markers
2. **Generates `internal/featuregates/featuregates.go`**: Contains all discovered gates
3. **Updates `cmd/main.go`**: Adds `--feature-gates` flag support

```go
// Generated in internal/featuregates/featuregates.go
var availableFeatureGates = map[string]bool{
    "experimental-bar":   false, // Default disabled
    "advanced-features": false, // Default disabled
}
```

### Controller Integration

Access feature gate state in your controller logic:

```go
func (r *MyResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // Get your resource
    var myResource myapiv1.MyResource
    if err := r.Get(ctx, req.NamespacedName, &myResource); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Check if experimental feature is enabled
    if featureGates.IsEnabled("experimental-bar") && myResource.Spec.Bar != nil {
        log.Info("Using experimental bar feature", "value", *myResource.Spec.Bar)
        // Implement experimental functionality
        return r.handleExperimentalBar(ctx, &myResource)
    }

    // Standard reconciliation logic
    return r.handleStandard(ctx, &myResource)
}
```

## Best Practices

### Naming Conventions

Use descriptive, lowercase names with hyphens:
- ✅ `experimental-bar`
- ✅ `advanced-features`
- ✅ `timezone-support`
- ❌ `ExperimentalBar` (should be kebab-case)
- ❌ `advanced_features` (use hyphens, not underscores)

### Documentation

Always document feature-gated fields:

```go
// Bar is an experimental field that provides enhanced functionality.
// It requires the "experimental-bar" feature gate to be enabled.
// When disabled, this field is ignored during reconciliation.
// +feature-gate experimental-bar
// +optional
Bar *string `json:"bar,omitempty"`
```

### Gradual Rollout Strategy

1. **Alpha**: Feature behind feature gate (disabled by default)
   ```go
   // +feature-gate experimental-feature
   ExperimentalField *string `json:"experimentalField,omitempty"`
   ```

2. **Beta**: Feature available but can be disabled
   ```go
   // +feature-gate beta-feature  
   BetaField *string `json:"betaField,omitempty"`
   ```
   
3. **Stable**: Remove feature gate, feature always available
   ```go
   // No feature gate marker - always available
   StableField string `json:"stableField"`
   ```

### Testing

Test your controller with different feature gate configurations:

```bash
# Test with all features disabled (default)
make test

# Test with specific features enabled  
FEATURE_GATES="experimental-bar=true" make test

# Test with all features enabled
FEATURE_GATES="experimental-bar=true,advanced-features=true" make test
```

## Deployment Configurations

### Development Environment

```yaml
# config/manager/manager.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --leader-elect
        - --feature-gates=experimental-bar=true,advanced-features=true
```

### Production Environment

```yaml
# Production - only stable features
apiVersion: apps/v1  
kind: Deployment
metadata:
  name: controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --leader-elect
        - --feature-gates=experimental-bar=false,advanced-features=false
```

### Canary Deployment

```yaml
# Canary deployment with experimental features
apiVersion: apps/v1
kind: Deployment  
metadata:
  name: controller-manager-canary
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: manager
        args:
        - --leader-elect
        - --feature-gates=experimental-bar=true
```

## Troubleshooting

### Common Issues

1. **Feature gate not discovered**
   - Ensure the `+feature-gate` marker is on the line immediately before the field
   - Re-run `kubebuilder create api` to regenerate feature gate files
   - Check that the marker follows the correct format: `// +feature-gate gate-name`

2. **Invalid feature gate name**
   - Use only lowercase letters, numbers, and hyphens
   - Examples: `experimental-bar`, `alpha-feature-v2`

3. **Validation errors**
   - Check that all specified gates are discovered in your API types
   - Verify syntax: `gate1=true,gate2=false` (no spaces around `=`)

4. **Controller not recognizing feature gates**
   - Ensure `cmd/main.go` includes the generated feature gate initialization
   - Verify that `internal/featuregates/featuregates.go` is properly generated

### Debugging

Enable debug logging to see feature gate discovery and validation:

```bash
# See feature gate initialization
./manager --feature-gates=experimental-bar=true --zap-log-level=debug

# Check available gates
./manager --help | grep feature-gates
```

### Verification

Verify your feature gates are working:

```bash
# List all available feature gates
grep -r "+feature-gate" api/

# Check generated feature gate file
cat internal/featuregates/featuregates.go

# Verify controller initialization
grep -A 5 -B 5 "feature-gates" cmd/main.go
```

## Implementation Status

### ✅ Production Ready

- Feature gate discovery from API type markers
- CLI flag `--feature-gates` for runtime control  
- Automatic scaffolding integration
- Validation and error handling
- Controller runtime integration

### 🚧 Future Enhancement

- **CRD schema modification**: Requires [controller-tools support](https://github.com/kubernetes-sigs/controller-tools/issues/1238)
- **Helm chart integration**: Dynamic feature gate configuration in Helm values
- **Operator lifecycle management**: Feature gate coordination across multiple controllers

When controller-tools gains feature gate support, you'll be able to use:

```go
// +kubebuilder:feature-gate=experimental-bar
// +optional
Bar *string `json:"bar,omitempty"`
```

This will automatically exclude the field from CRD schemas when the feature gate is disabled, providing true schema-level gating.

## Examples

### Complete Working Example

Here's a complete example showing feature gates in action:

```go
// api/v1/webapp_types.go
type WebAppSpec struct {
    // Core functionality - always available
    Image    string `json:"image"`
    Replicas int32  `json:"replicas"`

    // Experimental auto-scaling
    // +feature-gate auto-scaling
    // +optional
    AutoScaling *AutoScalingConfig `json:"autoScaling,omitempty"`

    // Beta feature - advanced routing
    // +feature-gate advanced-routing
    // +optional
    AdvancedRouting *RoutingConfig `json:"advancedRouting,omitempty"`
}

type AutoScalingConfig struct {
    MinReplicas int32 `json:"minReplicas"`
    MaxReplicas int32 `json:"maxReplicas"`
    TargetCPU   int32 `json:"targetCPU"`
}

type RoutingConfig struct {
    Rules []RoutingRule `json:"rules"`
}
```

```go
// controllers/webapp_controller.go
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    var webapp webappv1.WebApp
    if err := r.Get(ctx, req.NamespacedName, &webapp); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Handle auto-scaling if enabled
    if featureGates.IsEnabled("auto-scaling") && webapp.Spec.AutoScaling != nil {
        log.Info("Configuring auto-scaling", "min", webapp.Spec.AutoScaling.MinReplicas)
        if err := r.reconcileAutoScaling(ctx, &webapp); err != nil {
            return ctrl.Result{}, err
        }
    }

    // Handle advanced routing if enabled  
    if featureGates.IsEnabled("advanced-routing") && webapp.Spec.AdvancedRouting != nil {
        log.Info("Configuring advanced routing")
        if err := r.reconcileAdvancedRouting(ctx, &webapp); err != nil {
            return ctrl.Result{}, err
        }
    }

    // Core reconciliation logic
    return r.reconcileCore(ctx, &webapp)
}
```

This example demonstrates how feature gates enable experimental functionality while maintaining backward compatibility.