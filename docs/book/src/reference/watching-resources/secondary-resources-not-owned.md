# Watching Secondary Resources that are NOT `Owned`

In some scenarios, a controller may need to watch and respond to changes in
resources that it does not `Own`, meaning those resources are created and managed by
another controller.

The following examples demonstrate how a controller can monitor and reconcile resources
that it doesn’t directly manage. This applies to any resource not `Owned` by the controller,
including **Core Types** or **Custom Resources** managed by other controllers or projects
and reconciled in separate processes.

For instance, consider two custom resources—`Busybox` and `BackupBusybox`.
If changes to `Busybox` should trigger reconciliation in the `BackupBusybox` controller, we
can configure the `BackupBusybox` controller to watch for updates in `Busybox`.

### Example: Watching a Non-Owned Busybox Resource to Reconcile BackupBusybox

Consider a controller that manages a custom resource `BackupBusybox`
but also needs to monitor changes to `Busybox` resources across the cluster.
We only want to trigger reconciliation when `Busybox` instances have the Backup
feature enabled.

- **Why Watch Secondary Resources?**
    - The `BackupBusybox` controller is not responsible for creating or owning `Busybox`
    resources, but changes in these resources (such as updates or deletions) directly affect the primary
    resource (`BackupBusybox`).
    - By watching `Busybox` instances with a specific label, the controller ensures that the necessary
    actions (e.g., backups) are triggered only for the relevant resources.

### Configuration Example

Here’s how to configure the `BackupBusyboxReconciler` to watch changes in the
`Busybox` resource and trigger reconciliation for `BackupBusybox`:

```go
// SetupWithManager sets up the controller with the Manager.
// The controller will watch both the BackupBusybox primary resource and the Busybox resource.
func (r *BackupBusyboxReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&examplecomv1alpha1.BackupBusybox{}).  // Watch the primary resource (BackupBusybox)
        Watches(
            &source.Kind{Type: &examplecomv1alpha1.Busybox{}},  // Watch the Busybox CR
            handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
                // Trigger reconciliation for the BackupBusybox in the same namespace
                return []reconcile.Request{
                    {
                        NamespacedName: types.NamespacedName{
                            Name:      "backupbusybox",  // Reconcile the associated BackupBusybox resource
                            Namespace: obj.GetNamespace(),  // Use the namespace of the changed Busybox
                        },
                    },
                }
            }),
        ).  // Trigger reconciliation when the Busybox resource changes
        Complete(r)
}
```

Here’s how we can configure the controller to filter and watch
for changes to only those `Busybox` resources that have the specific label:

```go
// SetupWithManager sets up the controller with the Manager.
// The controller will watch both the BackupBusybox primary resource and the Busybox resource, filtering by a label.
func (r *BackupBusyboxReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&examplecomv1alpha1.BackupBusybox{}).  // Watch the primary resource (BackupBusybox)
        Watches(
            &source.Kind{Type: &examplecomv1alpha1.Busybox{}},  // Watch the Busybox CR
            handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
                // Check if the Busybox resource has the label 'backup-needed: "true"'
                if val, ok := obj.GetLabels()["backup-enable"]; ok && val == "true" {
                    // If the label is present and set to "true", trigger reconciliation for BackupBusybox
                    return []reconcile.Request{
                        {
                            NamespacedName: types.NamespacedName{
                                Name:      "backupbusybox",  // Reconcile the associated BackupBusybox resource
                                Namespace: obj.GetNamespace(),  // Use the namespace of the changed Busybox
                            },
                        },
                    }
                }
                // If the label is not present or doesn't match, don't trigger reconciliation
                return []reconcile.Request{}
            }),
        ).  // Trigger reconciliation when the labeled Busybox resource changes
        Complete(r)
}
```
