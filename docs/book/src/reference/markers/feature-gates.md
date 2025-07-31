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
    Bar *string `json:"bar,omitempty"`
}
```

### Running with Feature Gates

```bash
# Enable feature gates
./manager --feature-gates=experimental-bar

# Multiple gates
./manager --feature-gates=experimental-bar,advanced-features

# Mixed states
./manager --feature-gates=experimental-bar=true,advanced-features=false
```

## Overview

Feature gates provide a mechanism to:
- Control the availability of experimental features
- Enable gradual rollout of new functionality
- Maintain backward compatibility during API evolution

## Usage

### Marking Fields with Feature Gates

Use the `+feature-gate` marker to mark experimental fields in your API types:

```go
type MyResourceSpec struct {
    // Standard field
    Name string `json:"name"`

    // Experimental field that requires the "experimental-bar" feature gate
    // +feature-gate experimental-bar
    Bar *string `json:"bar,omitempty"`
}
```

### Running with Feature Gates

Enable feature gates when running your controller:

```bash
# Enable a single feature gate
./manager --feature-gates=experimental-bar

# Enable multiple feature gates
./manager --feature-gates=experimental-bar,advanced-features

# Mixed enabled/disabled states
./manager --feature-gates=experimental-bar=true,advanced-features=false
```

### Feature Gate Formats

The `--feature-gates` flag accepts:
- `feature1` - Enables feature1 (defaults to true)
- `feature1=true` - Explicitly enables feature1
- `feature1=false` - Explicitly disables feature1
- `feature1,feature2` - Enables both features
- `feature1=true,feature2=false` - Mixed states

## Best Practices

### Naming Conventions

Use descriptive, lowercase names with hyphens:
- ✅ `experimental-bar`
- ✅ `advanced-features`
- ❌ `ExperimentalBar`
- ❌ `advanced_features`

### Documentation

Always document feature-gated fields:

```go
// Bar is an experimental field that requires the "experimental-bar" feature gate
// +feature-gate experimental-bar
// +optional
Bar *string `json:"bar,omitempty"`
```

### Gradual Rollout Strategy

1. **Alpha**: Feature behind feature gate
2. **Beta**: Feature enabled by default, gate for opt-out
3. **Stable**: Remove feature gate, feature always available

## Future Integration

When [controller-tools supports feature gates](https://github.com/kubernetes-sigs/controller-tools/issues/1238), you'll be able to use:

```go
// +kubebuilder:feature-gate=experimental-bar
// +optional
Bar *string `json:"bar,omitempty"`
```

This will automatically exclude the field from CRD schemas when the feature gate is disabled.

## Examples

### Basic Example

```go
type CronJobSpec struct {
    // Standard fields
    Schedule string `json:"schedule"`
    
    // Experimental timezone support
    // +feature-gate timezone-support
    Timezone *string `json:"timezone,omitempty"`
}
```

### Controller Logic

```go
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Check if timezone feature is enabled
    if featureGates.IsEnabled("timezone-support") {
        // Use timezone-aware scheduling
        return r.reconcileWithTimezone(ctx, req)
    }
    
    // Fall back to standard scheduling
    return r.reconcileStandard(ctx, req)
}
```

## Troubleshooting

### Common Issues

1. **Feature gate not discovered**: Ensure the `+feature-gate` marker is on the line before the field
2. **Invalid feature gate name**: Use lowercase with hyphens only
3. **Validation errors**: Check that all specified gates are available

### Debugging

Enable debug logging to see feature gate discovery:

```bash
./manager --feature-gates=experimental-bar --zap-log-level=debug
```

## Implementation Status

### ✅ Production Ready

- Feature gate discovery and validation
- Controller integration with `--feature-gates` flag
- Scaffolding integration for new projects

### 🔄 Future Enhancement

- CRD schema modification (requires [controller-tools support](https://github.com/kubernetes-sigs/controller-tools/issues/1238))

When [controller-tools supports feature gates](https://github.com/kubernetes-sigs/controller-tools/issues/1238), you'll be able to use:

```go
// +kubebuilder:feature-gate=experimental-bar
// +optional
Bar *string `json:"bar,omitempty"`
```

This will automatically exclude the field from CRD schemas when the feature gate is disabled. 