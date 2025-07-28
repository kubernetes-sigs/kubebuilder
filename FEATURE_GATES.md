# Feature Gates for Kubebuilder

This document describes the feature gates functionality implemented in Kubebuilder, which allows developers to mark fields in their API types as belonging to specific feature gates.

## Overview

Feature gates provide a way to enable or disable experimental features in your CRDs, similar to how Kubernetes core types handle feature gates. This allows for:

- Gradual rollout of new features
- A/B testing of experimental functionality
- Safe deprecation of features
- Better control over API stability

## Usage

### 1. Marking Fields with Feature Gates

In your API types, you can mark fields with feature gate annotations:

```go
// MyResourceSpec defines the desired state of MyResource
type MyResourceSpec struct {
    // Standard field
    Name string `json:"name"`

    // Experimental field that requires the "experimental-feature" gate
    // +feature-gate experimental-feature
    ExperimentalField string `json:"experimentalField,omitempty"`

    // Another experimental field
    // +feature-gate another-feature
    AnotherField int `json:"anotherField,omitempty"`
}
```

### 2. Running with Feature Gates

When you run your controller, you can enable or disable feature gates:

```bash
# Enable all feature gates
./manager --feature-gates=experimental-feature,another-feature

# Enable some gates and disable others
./manager --feature-gates=experimental-feature=true,another-feature=false

# Disable a specific gate
./manager --feature-gates=experimental-feature=false
```

### 3. Feature Gate Discovery

Kubebuilder automatically discovers feature gates from your API types and generates validation code. The available feature gates are listed in the help text:

```bash
./manager --help
```

You'll see output like:
```
--feature-gates string   A set of key=value pairs that describe feature gates for alpha/experimental features. Options are: experimental-feature, another-feature
```

## Implementation Details

### Feature Gate Parsing

The feature gate string follows this format:
- `feature1` - enables feature1 (default)
- `feature1=true` - explicitly enables feature1
- `feature1=false` - explicitly disables feature1
- `feature1,feature2=false,feature3` - mixed enabled/disabled gates

### Marker Parsing

Kubebuilder scans your API types for markers in the format:
```
// +feature-gate gate-name
```

These markers are found in:
- Struct field comments
- Function comments
- Type comments

### Validation

The controller validates that:
1. All specified feature gates are valid (exist in the codebase)
2. Feature gate values are properly formatted
3. No duplicate or conflicting gate specifications

## Examples

### Basic Example

```go
// api/v1/myresource_types.go
type MyResourceSpec struct {
    // Standard field
    Name string `json:"name"`

    // Experimental field
    // +feature-gate alpha-feature
    AlphaField string `json:"alphaField,omitempty"`
}
```

### Advanced Example

```go
// api/v1/complexresource_types.go
type ComplexResourceSpec struct {
    // Standard fields
    Name string `json:"name"`
    Replicas int32 `json:"replicas"`

    // Beta feature
    // +feature-gate beta-feature
    BetaField string `json:"betaField,omitempty"`

    // Alpha feature
    // +feature-gate alpha-feature
    AlphaField string `json:"alphaField,omitempty"`

    // Experimental feature
    // +feature-gate experimental-feature
    ExperimentalField string `json:"experimentalField,omitempty"`
}
```

Running with different configurations:

```bash
# Enable all features
./manager --feature-gates=alpha-feature,beta-feature,experimental-feature

# Enable only beta features
./manager --feature-gates=beta-feature

# Disable experimental features
./manager --feature-gates=alpha-feature,beta-feature,experimental-feature=false
```

## Best Practices

1. **Use descriptive gate names**: Choose names that clearly indicate what the feature does
2. **Document your gates**: Add comments explaining what each feature gate enables
3. **Gradual rollout**: Start with alpha features, then beta, then stable
4. **Consistent naming**: Use kebab-case for feature gate names
5. **Validation**: Always validate feature gate inputs in your controllers

## Migration Guide

### From No Feature Gates

1. Add feature gate markers to your experimental fields
2. Rebuild your project: `make manifests`
3. Update your deployment to include the `--feature-gates` flag
4. Test with different gate combinations

### To Stable Features

When a feature is ready for stable release:
1. Remove the feature gate marker
2. Update documentation
3. Consider the field stable and always available

## Troubleshooting

### Common Issues

1. **Unknown feature gate**: Make sure the gate name matches exactly in your code
2. **Invalid format**: Check that your feature gate string follows the correct format
3. **Missing validation**: Ensure your controller validates feature gates before use

### Debugging

Enable verbose logging to see feature gate processing:

```bash
./manager --feature-gates=your-gate --v=2
```

## Future Enhancements

Planned improvements include:
- CRD schema modification based on enabled gates
- Automatic documentation generation
- Integration with controller-runtime feature gates
- Webhook validation based on feature gates 