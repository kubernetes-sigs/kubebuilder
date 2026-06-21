# (Alpha) Server-Side Apply

The `--ssa` flag scaffolds APIs with [Server-Side Apply][server-side-apply] support, enabling safer field management when multiple actors modify the same resources.

<aside class="note warning" role="note">
<p class="note-title">Alpha feature</p>

The `--ssa` flag is an alpha feature and may change in future releases.

</aside>

This adds:
- API markers (`+genclient`, `+kubebuilder:ac:generate=true`, `+kubebuilder:resource:path=<plural>`) for ApplyConfiguration generation
- Makefile integration to generate type-safe ApplyConfiguration types alongside CRD and RBAC manifests

## When to use it

Use Server-Side Apply when:
- Multiple controllers or users manage the same resource
- Users customize CRs with labels, annotations, or spec fields your controller shouldn't overwrite
- You want declarative field ownership tracking
- Other operators will manage instances of your CRs (they can import your generated ApplyConfiguration types)

<aside class="note" role="note">
<p class="note-title">Note</p>

For controllers that are the sole owner of a resource and manage the entire object, traditional `Update()` is simpler and sufficient. Use Server-Side Apply when field ownership matters.

</aside>

## How it works

Traditional `Update()` overwrites the entire object. Server-Side Apply with the client's `Apply` method only manages the fields you specify.

After running `make manifests`, ApplyConfiguration types are created at:
```
api/<version>/applyconfiguration/api/<version>/          (single group)
api/<group>/<version>/applyconfiguration/<group>/<version>/  (multi-group)
```

Import them as (multi-group projects use
`example.com/myproject/api/<group>/v1/applyconfiguration/<group>/v1`):
```go
appsv1apply "example.com/myproject/api/v1/applyconfiguration/api/v1"
```

## Mixing SSA and non-SSA APIs

The `+kubebuilder:ac:generate=true` marker is package-level: it enables ApplyConfiguration generation for all kinds in the same group/version. Kinds scaffolded without `--ssa` include the `+kubebuilder:ac:generate=false` marker, so ApplyConfiguration types are only generated for the kinds that opted in. When you create an API with `--ssa`, kinds previously scaffolded without this marker in that group/version receive it automatically.

To change which kinds are generated:
- Set `+kubebuilder:ac:generate=true` above a kind to include it
- Set `+kubebuilder:ac:generate=false` above a kind to exclude it

<aside class="note" role="note">
<p class="note-title">Note</p>

If your project has customizations and kubebuilder cannot update the Makefile or a `*_types.go` file, it logs a warning with the manual change required. Scaffolding is not interrupted.

APIs added manually (not tracked in the PROJECT file) are not updated automatically. Review them and add `+kubebuilder:ac:generate=false` above any kind that should not use Server-Side Apply.

</aside>

## Usage

Create an API with the `--ssa` flag:

```shell
kubebuilder create api --group apps --version v1 --kind Application --ssa
```

Implement Server-Side Apply in your controller:

```go
import (
    "context"

    metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"

    appsv1apply "example.com/myproject/api/v1/applyconfiguration/api/v1"
)

func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Build desired state - specify only fields you manage
    app := appsv1apply.Application(req.Name, req.Namespace).
        WithStatus(appsv1apply.ApplicationStatus().
            WithConditions(metav1apply.Condition().
                WithType("Available").
                WithStatus("True").
                WithReason("Reconciled")))

    // Apply status, forcing ownership of the fields this controller manages
    if err := r.SubResource("status").Apply(ctx, app,
        client.FieldOwner("application-controller"), client.ForceOwnership); err != nil {
        return ctrl.Result{}, err
    }
    return ctrl.Result{}, nil
}
```

Then run:
```shell
make generate manifests
```

## Best practices

- Always specify `client.FieldOwner("controller-name")` to identify your controller
- Controllers should pass `client.ForceOwnership` for the fields they manage; Kubernetes
  [recommends][server-side-apply] that controllers always force conflicts
- Only specify fields your controller manages using the builder pattern

## Additional resources

- [Kubernetes Server-Side Apply Documentation][server-side-apply]
- [Apply Configuration markers](./markers/applyconfiguration.md)
- [controller-gen CLI Reference](./controller-gen.md)

[server-side-apply]: https://kubernetes.io/docs/reference/using-api/server-side-apply/
