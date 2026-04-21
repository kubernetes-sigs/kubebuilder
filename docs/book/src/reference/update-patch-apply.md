# Choosing Between Update, Patch, and Apply

Use `Update` (HTTP `PUT`) for replacement-style changes and `Patch` (HTTP
`PATCH`) for partial changes. In Kubernetes, server-side apply is implemented
as a specialized `PATCH` operation with field ownership tracked by the API
server.

If you want a default that is rarely wrong, prefer `Patch` with optimistic
concurrency.

## Quick Decision Guide

| If your controller needs to... | Usually use | Why |
| --- | --- | --- |
| Replace most of an object or most of its `status` after a fresh read | `Update` | Simple, but it writes the full object and conflicts more often |
| Change only a few fields on an existing object | `Patch` | Safer for shared objects and less likely to overwrite unrelated fields |
| Declaratively manage a known subset of fields on owned child resources | `Apply` | The API server tracks field ownership and merges intent |

## `Update`

Use `Update` when your controller has just read the object and intentionally
plans to write back most of it.

- It sends the full object or subresource.
- It requires handling `409 Conflict` errors and retrying or requeueing.
- `Status().Update(...)` is often reasonable when the controller is the only
  status writer.

## `Patch`

Use `Patch` when your controller changes only part of an object or when other
actors may also update it.

- It is usually the safest general-purpose choice for controllers.
- Prefer it for finalizers, labels, annotations, and other small changes.
- For CRDs, do not rely on strategic merge patch semantics.

## `Apply`

`Apply` means [Server-Side Apply][k8s-ssa]. It is a `PATCH` operation, not a
separate HTTP method.

- Use it when your controller clearly owns the fields it writes.
- It often fits owned child resources such as `Deployment` or `Service`.
- Avoid it when the new value depends on the current live value or field
  ownership is unclear.

## Practical Recommendations

For a typical Kubebuilder project:

- avoid writing the primary Custom Resource `spec` from the controller unless
  the API is explicitly designed for it
- use `Status().Update(...)` when the controller is the only status writer, or
  `Status().Patch(...)` when only part of `status` changes
- prefer `Patch` for shared resources and for changes that affect only part of
  an object

## Further Reading

- [Kubernetes API Concepts][k8s-api-concepts]
- [Server-Side Apply][k8s-ssa]
- [controller-runtime client package][controller-runtime-client]

[k8s-api-concepts]: https://kubernetes.io/docs/reference/using-api/api-concepts/
[k8s-ssa]: https://kubernetes.io/docs/reference/using-api/server-side-apply/
[controller-runtime-client]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client
