# Using Predicates to Refine Watches

When working with controllers, it's often beneficial to use **Predicates** to
filter events and control when the reconciliation loop should be triggered.

[Predicates][predicates-doc] allow you to define conditions based on events (such as create, update, or delete)
and resource fields (such as labels, annotations, or status fields). By using **[Predicates][predicates-doc]**,
you can refine your controller’s behavior to respond only to specific changes in the resources
it watches.

This can be especially useful when you want to refine which
changes in resources should trigger a reconciliation. By using predicates,
you avoid unnecessary reconciliations and can ensure that the
controller only reacts to relevant changes.

## When to Use Predicates

**Predicates are useful when:**

- You want to ignore certain changes, such as updates that don't impact the fields your controller is concerned with.
- You want to trigger reconciliation only for resources with specific labels or annotations.
- You want to watch external resources and react only to specific changes.

## Example: Using Predicates to Filter Update Events

Let’s say that we only want our **`BackupBusybox`** controller to reconcile
when certain fields of the **`Busybox`** resource change, for example, when
the `spec.size` field changes, but we want to ignore all other changes (such as status updates).

### Defining a Predicate

In the following example, we define a predicate that only
allows reconciliation when there’s a meaningful update
to the **`Busybox`** resource:

```go
import (
    "sigs.k8s.io/controller-runtime/pkg/predicate"
    "sigs.k8s.io/controller-runtime/pkg/event"
)

// Predicate to trigger reconciliation only on size changes in the Busybox spec
updatePred := predicate.Funcs{
    // Only allow updates when the spec.size of the Busybox resource changes
    UpdateFunc: func(e event.UpdateEvent) bool {
        oldObj := e.ObjectOld.(*examplecomv1alpha1.Busybox)
        newObj := e.ObjectNew.(*examplecomv1alpha1.Busybox)

        // Trigger reconciliation only if the spec.size field has changed
        return oldObj.Spec.Size != newObj.Spec.Size
    },

    // Allow create events
    CreateFunc: func(e event.CreateEvent) bool {
        return true
    },

    // Allow delete events
    DeleteFunc: func(e event.DeleteEvent) bool {
        return true
    },

    // Allow generic events (e.g., external triggers)
    GenericFunc: func(e event.GenericEvent) bool {
        return true
    },
}
```

### Explanation

In this example:
- The **`UpdateFunc`** returns `true` only if the **`spec.size`** field has changed between the old and new objects, meaning that all other changes in the `spec`, like annotations or other fields, will be ignored.
- **`CreateFunc`**, **`DeleteFunc`**, and **`GenericFunc`** return `true`, meaning that create, delete, and generic events are still processed, allowing reconciliation to happen for these event types.

This ensures that the controller reconciles only when the specific field **`spec.size`** is modified, while ignoring any other modifications in the `spec` that are irrelevant to your logic.

### Example: Using Predicates in `Watches`

Now, we apply this predicate in the **`Watches()`** method of
the **`BackupBusyboxReconciler`** to trigger reconciliation only for relevant events:

```go
// SetupWithManager sets up the controller with the Manager.
// The controller will watch both the BackupBusybox primary resource and the Busybox resource, using predicates.
func (r *BackupBusyboxReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&examplecomv1alpha1.BackupBusybox{}).  // Watch the primary resource (BackupBusybox)
        Watches(
            &source.Kind{Type: &examplecomv1alpha1.Busybox{}},  // Watch the Busybox CR
            handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
                return []reconcile.Request{
                    {
                        NamespacedName: types.NamespacedName{
                            Name:      "backupbusybox",  // Reconcile the associated BackupBusybox resource
                            Namespace: obj.GetNamespace(),  // Use the namespace of the changed Busybox
                        },
                    },
                }
            }),
            builder.WithPredicates(updatePred),  // Apply the predicate
        ).  // Trigger reconciliation when the Busybox resource changes (if it meets predicate conditions)
        Complete(r)
}
```

### Explanation

- **[`builder.WithPredicates(updatePred)`][predicates-doc]**: This method applies the predicate, ensuring that reconciliation only occurs
when the **`spec.size`** field in **`Busybox`** changes.
- **Other Events**: The controller will still trigger reconciliation on `Create`, `Delete`, and `Generic` events.

[predicates-doc]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/source#WithPredicates