# Watching Secondary Resources `Owned` by the Controller

In Kubernetes controllers, it’s common to manage both **Primary Resources**
and **Secondary Resources**. A **Primary Resource** is the main resource
that the controller is responsible for, while **Secondary Resources**
are created and managed by the controller to support the **Primary Resource**.

In this section, we will explain how to manage **Secondary Resources**
which are `Owned` by the controller. This example shows how to:

- Set the [Owner Reference][cr-owner-ref-doc] between the primary resource (`Busybox`) and the secondary resource (`Deployment`) to ensure proper lifecycle management.
- Configure the controller to `Watch` the secondary resource using `Owns()` in `SetupWithManager()`. See that `Deployment` is owned by the `Busybox` controller because
it will be created and managed by it.

## Setting the Owner Reference

To link the lifecycle of the secondary resource (`Deployment`)
to the primary resource (`Busybox`), we need to set
an [Owner Reference][cr-owner-ref-doc] on the secondary resource.
This ensures that Kubernetes automatically handles cascading deletions:
if the primary resource is deleted, the secondary resource will also be deleted.

Controller-runtime provides the [controllerutil.SetControllerReference][cr-owner-ref-doc] function, which you can use to set this relationship between the resources.

### Setting the Owner Reference

Below, we create the `Deployment` and set the Owner reference between the `Busybox` custom resource and the `Deployment` using `controllerutil.SetControllerReference()`.

```go
// deploymentForBusybox returns a Deployment object for Busybox
func (r *BusyboxReconciler) deploymentForBusybox(busybox *examplecomv1alpha1.Busybox) *appsv1.Deployment {
    replicas := busybox.Spec.Size

    dep := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      busybox.Name,
            Namespace: busybox.Namespace,
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": busybox.Name},
            },
            Template: metav1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": busybox.Name},
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "busybox",
                            Image: "busybox:latest",
                        },
                    },
                },
            },
        },
    }

    // Set the ownerRef for the Deployment, ensuring that the Deployment
    // will be deleted when the Busybox CR is deleted.
    controllerutil.SetControllerReference(busybox, dep, r.Scheme)
    return dep
}
```

### Explanation

By setting the `OwnerReference`, if the `Busybox` resource is deleted, Kubernetes will automatically delete
the `Deployment` as well. This also allows the controller to watch for changes in the `Deployment`
and ensure that the desired state (such as the number of replicas) is maintained.

For example, if someone modifies the `Deployment` to change the replica count to 3,
while the `Busybox` CR defines the desired state as 1 replica,
the controller will reconcile this and ensure the `Deployment`
is scaled back to 1 replica.

**Reconcile Function Example**

```go
// Reconcile handles the main reconciliation loop for Busybox and the Deployment
func (r *BusyboxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // Fetch the Busybox instance
    busybox := &examplecomv1alpha1.Busybox{}
    if err := r.Get(ctx, req.NamespacedName, busybox); err != nil {
        if apierrors.IsNotFound(err) {
            log.Info("Busybox resource not found. Ignoring since it must be deleted")
            return ctrl.Result{}, nil
        }
        log.Error(err, "Failed to get Busybox")
        return ctrl.Result{}, err
    }

    // Check if the Deployment already exists, if not create a new one
    found := &appsv1.Deployment{}
    err := r.Get(ctx, types.NamespacedName{Name: busybox.Name, Namespace: busybox.Namespace}, found)
    if err != nil && apierrors.IsNotFound(err) {
        // Define a new Deployment
        dep := r.deploymentForBusybox(busybox)
        log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
        if err := r.Create(ctx, dep); err != nil {
            log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
            return ctrl.Result{}, err
        }
        // Requeue the request to ensure the Deployment is created
        return ctrl.Result{RequeueAfter: time.Minute}, nil
    } else if err != nil {
        log.Error(err, "Failed to get Deployment")
        return ctrl.Result{}, err
    }

    // Ensure the Deployment size matches the desired state
    size := busybox.Spec.Size
    if *found.Spec.Replicas != size {
        found.Spec.Replicas = &size
        if err := r.Update(ctx, found); err != nil {
            log.Error(err, "Failed to update Deployment size", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
            return ctrl.Result{}, err
        }
        // Requeue the request to ensure the correct state is achieved
        return ctrl.Result{Requeue: true}, nil
    }

    // Update Busybox status to reflect that the Deployment is available
    busybox.Status.AvailableReplicas = found.Status.AvailableReplicas
    if err := r.Status().Update(ctx, busybox); err != nil {
        log.Error(err, "Failed to update Busybox status")
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

## Watching Secondary Resources

To ensure that changes to the secondary resource (such as the `Deployment`) trigger
a reconciliation of the primary resource (`Busybox`), we configure the controller
to watch both resources.

The `Owns()` method allows you to specify secondary resources
that the controller should monitor. This way, the controller will
automatically reconcile the primary resource whenever the secondary
resource changes (e.g., is updated or deleted).

### Example: Configuring `SetupWithManager` to Watch Secondary Resources

```go
// SetupWithManager sets up the controller with the Manager.
// The controller will watch both the Busybox primary resource and the Deployment secondary resource.
func (r *BusyboxReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&examplecomv1alpha1.Busybox{}).  // Watch the primary resource
        Owns(&appsv1.Deployment{}).          // Watch the secondary resource (Deployment)
        Complete(r)
}
```

## Ensuring the Right Permissions

Kubebuilder uses [markers][markers] to define RBAC permissions
required by the controller. In order for the controller to
properly watch and manage both the primary (`Busybox`) and secondary (`Deployment`)
resources, it must have the appropriate permissions granted;
i.e. to `watch`, `get`, `list`, `create`, `update`, and `delete` permissions for those resources.

### Example: RBAC Markers

Before the `Reconcile` method, we need to define the appropriate RBAC markers.
These markers will be used by [controller-gen][controller-gen] to generate the necessary
roles and permissions when you run `make manifests`.

```go
// +kubebuilder:rbac:groups=example.com,resources=busyboxes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
```

- The first marker gives the controller permission to manage the `Busybox` custom resource (the primary resource).
- The second marker grants the controller permission to manage `Deployment` resources (the secondary resource).

Note that we are granting permissions to `watch` the resources.

[owner-ref-k8s-docs]: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
[cr-owner-ref-doc]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller/controllerutil#SetOwnerReference
[controller-gen]: ./../controller-gen.md
[markers]:./../markers/rbac.md
