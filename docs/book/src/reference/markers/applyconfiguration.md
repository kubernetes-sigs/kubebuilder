# Apply Configuration

These markers control when ApplyConfiguration types are generated for the kinds
in a package. These types provide a type-safe way to build patches for
[Server-Side Apply][k8s-ssa-docs].

See the [Server-Side Apply](../server-side-apply.md) reference for how the `--ssa`
flag enables this generation and for examples of using the generated types.

{{#markerdocs apply}}

[k8s-ssa-docs]: https://kubernetes.io/docs/reference/using-api/server-side-apply/
