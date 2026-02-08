# CRD Validation

These markers modify how the CRD validation schema is produced for the
types and fields they modify.  Each corresponds roughly to an OpenAPI/JSON
schema option.

See [Generating CRDs](/reference/generating-crd.md) for examples.

<aside class="note">
<h1>Understanding Marker Grouping in Documentation</h1>

Certain markers may seem duplicated. However, these markers are grouped based on their context of use
— such as fields, types, or arrays. For instance, a marker like `+kubebuilder:validation:Enum` can be applied to
individual fields or array items, and this flexibility is reflected in the documentation.

The grouping ensures clarity by showing how the same marker can be reused for different purposes.

</aside>

<aside class="note">
<h1>Schema & Validation</h1>

Custom resources are validated using the generated OpenAPI v3 schema and must comply with Kubernetes structural schema rules.
Only field types and constraints that can be represented in the CRD schema are enforced by the API server.

Numeric types are constrained by [Kubernetes CRD OpenAPI v3 schema compatibility](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation).
In practice, prefer Go types that map cleanly to the supported OpenAPI formats (for example, `int32` and `int64` for integers).
For values that require decimal-like representation, use `resource.Quantity`.

Use Kubebuilder validation markers to declare additional constraints — such as minimum and maximum values, length limits, patterns, enums, and list or map rules — so they are reflected in the generated CRD and validated at runtime.
</aside>

{{#markerdocs CRD validation}}
