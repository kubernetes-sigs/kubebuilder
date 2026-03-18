# Choosing Between Update, Patch, and Apply

New controller authors often ask for one rule that is always correct. There is no such rule.
The right choice depends on what fields your controller owns, how much of the object it wants
to manage, and how likely it is that other actors also write to that object.

If you want a default that is rarely wrong, prefer `Patch` with optimistic concurrency.

## Quick Decision Guide

| If your controller needs to... | Usually use | Why |
| --- | --- | --- |
| Replace most of an object or most of its `status` after a fresh read | `Update` | Simple, but it writes the full object and conflicts more often |
| Change only a few fields on an existing object | `Patch` | Safer for shared objects and less likely to overwrite unrelated fields |
| Declaratively manage a known subset of fields, often on owned child resources | `Apply` | Good create-or-update flow with field ownership tracked by the API server |

For controller authors, this is a useful starting point:

- `Update` is the blunt tool.
- `Patch` is the safest general-purpose tool.
- `Apply` is best when your controller clearly owns specific fields and can send its full intent every time.

## `Update`

`Update` uses HTTP `PUT`. You send back the full object for that resource or subresource.

Use `Update` when:

- your controller has just read the object and is intentionally writing back most of it
- your controller is effectively the main writer for that object or subresource
- you are comfortable handling `409 Conflict` errors and retrying

Be careful with `Update` because:

- it requires a current `resourceVersion`
- it rewrites the whole object, not just the fields you changed
- it can accidentally drop fields your client does not know about
- it conflicts more often when other actors update unrelated fields

For many controllers, `Status().Update(...)` is still reasonable because the controller is often
the only writer of that `status` subresource. Even then, you should expect conflicts and retry or
requeue when needed.

## `Patch`

`Patch` changes only part of an object. For controller code, this is often the best default because
controllers usually change a small number of fields while other actors may be updating the same object.

Use `Patch` when:

- you only want to change a few fields
- the object may also be changed by users, admission webhooks, or other controllers
- you want to reduce the chance of overwriting unrelated changes

For controller-runtime clients, `MergeFrom` is a common choice. To make it safer, add optimistic
concurrency so the patch includes `metadata.resourceVersion`:

```go
base := deployment.DeepCopy()

deployment.Spec.Replicas = &size

if err := r.Patch(
    ctx,
    deployment,
    client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{}),
); err != nil {
    return ctrl.Result{}, err
}
```

This keeps the lost-update protection of `Update` while avoiding a full-object write.

Use `Patch` especially for:

- finalizers
- labels and annotations
- small changes to `spec`
- small changes to `status`
- shared secondary resources

### A Note About Patch Types

Kubernetes supports multiple patch types. For controllers, the most relevant ones are:

- JSON merge patch, which controller-runtime uses for `client.MergeFrom(...)`
- Server-Side Apply, covered below

Avoid relying on Strategic Merge Patch for custom resources served via CRDs. Kubernetes does not
support `application/strategic-merge-patch+json`
([StrategicMergeFrom()](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client#StrategicMergeFrom))
for APIs defined using `CustomResourceDefinition`.

Also note that merge patch replaces lists rather than merging them item by item. If you patch a list,
be explicit about the full list value you want.

For CRDs, markers such as [`+listType` and
`+listMapKey`](./markers/crd-processing.md) describe
schema-level list semantics, which are relevant for [Server-Side Apply merge
behavior](https://kubernetes.io/docs/reference/using-api/server-side-apply/#merge-strategy), not for
JSON merge patch used by `client.MergeFrom(...)`.

If you want `Create`, `Update`, and non-apply `Patch` requests to fail fast when they contain
unknown fields, see [Strict Field Validation](./strict-field-validation.md). In controller-runtime,
`Apply` requests are already strict.

## `Apply`

`Apply` means [Server-Side Apply][k8s-ssa]. Instead of sending a read-modify-write update, your controller
sends its declarative intent and lets the API server merge that intent with the live object.
In this section, `Apply` refers to the Kubernetes API operation over HTTP, not the `kubectl apply`
command-line workflow.

Use `Apply` when:

- your controller owns the fields it is writing
- your controller can send the complete desired value for those owned fields on every reconcile
- you want create-or-update behavior without a separate `Get`

`Apply` is often a good fit for child objects such as:

- `Deployment`
- `Service`
- `ConfigMap`
- `Job`

especially when your controller created those resources and clearly owns their managed fields.

`Apply` is usually a poor fit when:

- the new value depends on the current live value
- you need read-modify-write behavior
- field ownership is unclear
- users or other controllers are expected to edit the same fields

When using SSA in a controller:

- use a stable field manager name
- send the full intent for the fields your controller owns
- force conflicts only for objects your controller truly owns and manages

`Apply` and [strict field validation](./strict-field-validation.md) are still separate concepts, but
controller-runtime does not expose a separate `fieldValidation` setting for `Apply`. `Apply`
requests are already strict and fail if they contain unknown or duplicate fields.

<aside class="note">
<h1>Unknown Fields and Version Skew</h1>

Strict field validation is separate from choosing `Update`, `Patch`, or `Apply`.
It controls whether the API server drops unknown fields, warns, or returns an error.
This is especially useful for catching skew between controller code and installed CRDs.
For controller-runtime, `Apply` already fails on unknown or duplicate fields, while `Create`,
`Update`, and non-apply `Patch` can be configured to ignore, warn, or fail.
For built-in resources, be more careful: strict validation can reduce compatibility across
Kubernetes versions if your controller writes fields that do not exist on every target cluster.
See [Strict Field Validation](./strict-field-validation.md).

</aside>

## Practical Recommendations for New Controller Authors

For a typical Kubebuilder project:

- For the primary Custom Resource `spec`: usually do not write it from the controller at all. Users own desired state.
- For the primary Custom Resource `status`: `Status().Update(...)` is fine when your controller is the only status writer. Use `Status().Patch(...)` if you only change part of `status` or expect concurrent writers.
- For finalizers on the primary resource: prefer `Patch`.
- For owned child resources: prefer `Apply` if you want declarative ownership, or `Patch` if the update depends on the current live object.
- For shared resources: prefer `Patch` with optimistic concurrency.

If you are unsure, start with `Patch` plus optimistic concurrency. It is not always the shortest code,
but it is a strong default for avoiding accidental overwrites.

## Further Reading

For the underlying Kubernetes semantics, see:

- [Kubernetes API Concepts][k8s-api-concepts]
- [Server-Side Apply][k8s-ssa]
- [Strict Field Validation](./strict-field-validation.md)
- [controller-runtime client package][controller-runtime-client]

[k8s-api-concepts]: https://kubernetes.io/docs/reference/using-api/api-concepts/
[k8s-ssa]: https://kubernetes.io/docs/reference/using-api/server-side-apply/
[controller-runtime-client]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client
