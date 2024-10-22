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


{{#markerdocs CRD validation}}
