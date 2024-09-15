# Watching Resources

When extending the Kubernetes API, we aim to ensure that our solutions behave consistently with Kubernetes itself.
For example, consider a `Deployment` resource, which is managed by a controller. This controller is responsible
for responding to changes in the cluster—such as when a `Deployment` is created, updated, or deleted—by triggering
reconciliation to ensure the resource’s state matches the desired state.

Similarly, when developing our controllers, we want to watch for relevant changes in resources that are crucial
to our solution. These changes—whether creations, updates, or deletions—should trigger the reconciliation
loop to take appropriate actions and maintain consistency across the cluster.

The [controller-runtime][controller-runtime] library provides several ways to watch and manage resources.

## Primary Resources

The **Primary Resource** is the resource that your controller is responsible
for managing. For example, if you create a custom resource definition (CRD) for `MyApp`,
the corresponding controller is responsible for managing instances of `MyApp`.

In this case, `MyApp` is the **Primary Resource** for that controller, and your controller’s
reconciliation loop focuses on ensuring the desired state of these primary resources is maintained.

When you create a new API using Kubebuilder, the following default code is scaffolded,
ensuring that the controller watches all relevant events—such as creations, updates, and
deletions—for (`For()`) the new API.

This setup guarantees that the reconciliation loop is triggered whenever an instance
of the API is created, updated, or deleted:

```go
// Watches the primary resource (e.g., MyApp) for create, update, delete events
if err := ctrl.NewControllerManagedBy(mgr).
   For(&<YourAPISpec>{}). <-- See there that the Controller is For this API
   Complete(r); err != nil {
   return err
}
```

## Secondary Resources

Your controller will likely also need to manage **Secondary Resources**,
which are the resources required on the cluster to support the **Primary Resource**.

Changes to these **Secondary Resources** can directly impact the **Primary Resource**,
so the controller must watch and reconcile these resources accordingly.

### Which are Owned by the Controller

These **Secondary Resources**, such as `Services`, `ConfigMaps`, or `Deployments`,
when `Owned` by the controllers, are created and managed by the specific controller
and are tied to the **Primary Resource** via [OwnerReferences][owner-ref-k8s-docs].

For example, if we have a controller to manage our CR(s) of the Kind `MyApp`
on the cluster, which represents our application solution, all resources required
to ensure that `MyApp` is up and running with the desired number of instances
will be **Secondary Resources**. The code responsible for creating, deleting,
and updating these resources will be part of the `MyApp` Controller.
We would add the appropriate [OwnerReferences][owner-ref-k8s-docs]
using the [controllerutil.SetControllerReference][cr-owner-ref-doc]
function to indicate that these resources are owned by the same controller
responsible for managing `MyApp` instances, which will be reconciled by the `MyAppReconciler`.

Additionally, if the **Primary Resource** is deleted, Kubernetes' garbage collection mechanism
ensures that all associated **Secondary Resources** are automatically deleted in a
cascading manner.

### Which are NOT `Owned` by the Controller

Note that **Secondary Resources** can either be APIs/CRDs defined in your project or in other projects that are
relevant to the **Primary Resources**, but which the specific controller is not responsible for creating or managing.

For example, if we have a CRD that represents a backup solution (i.e. `MyBackup`) for our `MyApp`,
it might need to watch changes in the `MyApp` resource to trigger reconciliation in `MyBackup`
to ensure the desired state. Similarly, `MyApp`'s behavior might also be impacted by
CRDs/APIs defined in other projects.

In both scenarios, these resources are treated as **Secondary Resources**, even if they are not `Owned`
(i.e., not created or managed) by the `MyAppController`.

In Kubebuilder, resources that are not defined in the project itself and are not
a **Core Type** (those not defined in the Kubernetes API) are called **External Types**.

An **External Type** refers to a resource that is not defined in your
project but one that you need to watch and respond to.
For example, if **Operator A** manages a `MyApp` CRD for application deployment,
and **Operator B** handles backups, **Operator B** can watch the `MyApp` CRD as an external type
to trigger backup operations based on changes in `MyApp`.

In this scenario, **Operator B** could define a `BackupConfig` CRD that relies on the state of `MyApp`.
By treating `MyApp` as a **Secondary Resource**, **Operator B** can watch and reconcile changes in **Operator A**'s `MyApp`,
ensuring that backup processes are initiated whenever `MyApp` is updated or scaled.

## General Concept of Watching Resources

Whether a resource is defined within your project or comes from an external project, the concept of **Primary**
and **Secondary Resources** remains the same:
- The **Primary Resource** is the resource the controller is primarily responsible for managing.
- **Secondary Resources** are those that are required to ensure the primary resource works as desired.

Therefore, regardless of whether the resource was defined by your project or by another project,
your controller can watch, reconcile, and manage changes to these resources as needed.

## Why does watching the secondary resources matter?

When building a Kubernetes controller, it’s crucial to not only focus
on **Primary Resources** but also to monitor **Secondary Resources**.
Failing to track these resources can lead to inconsistencies in your
controller's behavior and the overall cluster state.

Secondary resources may not be directly managed by your controller,
but changes to these resources can still significantly
impact the primary resource and your controller's functionality.
Here are the key reasons why it's important to watch them:

- **Ensuring Consistency**:
    - Secondary resources (e.g., child objects or external dependencies) may diverge from their desired state.
    For instance, a secondary resource may be modified or deleted, causing the system to fall out of sync.
    - Watching secondary resources ensures that any changes are detected immediately, allowing the controller to
    reconcile and restore the desired state.

- **Avoiding Random Self-Healing**:
    - Without watching secondary resources, the controller may "heal" itself only upon restart or when specific events
    are triggered. This can cause unpredictable or delayed reactions to issues.
    - Monitoring secondary resources ensures that inconsistencies are addressed promptly, rather than waiting for a
    controller restart or external event to trigger reconciliation.

- **Effective Lifecycle Management**:
    - Secondary resources might not be owned by the controller directly, but their state still impacts the behavior
    of primary resources. Without watching these, you risk leaving orphaned or outdated resources.
    - Watching non-owned secondary resources lets the controller respond to lifecycle events (create, update, delete)
    that might affect the primary resource, ensuring consistent behavior across the system.

## Why not use `RequeueAfter X` for all scenarios instead of watching resources?

Kubernetes controllers are fundamentally **event-driven**. When creating a controller,
the **Reconciliation Loop** is typically triggered by **events** such as `create`, `update`, or
`delete` actions on resources. This event-driven approach is more efficient and responsive
compared to constantly requeuing or polling resources using `RequeueAfter`. This ensures that
the system only takes action when necessary, maintaining both performance and efficiency.

In many cases, **watching resources** is the preferred approach for ensuring Kubernetes resources
remain in the desired state. It is more efficient, responsive, and aligns with Kubernetes' event-driven architecture.
However, there are scenarios where `RequeueAfter` is appropriate and necessary, particularly for managing external
systems that do not emit events or for handling resources that take time to converge, such as long-running processes.
Relying solely on `RequeueAfter` for all scenarios can lead to unnecessary overhead and
delayed reactions. Therefore, it is essential to prioritize **event-driven reconciliation** by configuring
your controller to **watch resources** whenever possible, and reserving `RequeueAfter` for situations
where periodic checks are required.

### When `RequeueAfter X` is Useful

While `RequeueAfter` is not the primary method for triggering reconciliations, there are specific cases where it is
necessary, such as:

- **Observing External Systems**: When working with external resources that do not generate events
(e.g., external databases or third-party services), `RequeueAfter` allows the
controller to periodically check the status of these resources.
- **Time-Based Operations**: Some tasks, such as rotating secrets or
renewing certificates, must happen at specific intervals. `RequeueAfter` ensures these operations
are performed on schedule, even when no other changes occur.
- **Handling Errors or Delays**: When managing resources that encounter errors or require time to self-heal,
`RequeueAfter` ensures the controller waits for a specified duration before checking the resource’s status again,
avoiding constant reconciliation attempts.

## Usage of Predicates

For more complex use cases, [Predicates][cr-predicates] can be used to fine-tune
when your controller should trigger reconciliation. Predicates allow you to filter
events based on specific conditions, such as changes to particular fields, labels, or annotations,
ensuring that your controller only responds to relevant events and operates efficiently.

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[owner-ref-k8s-docs]: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
[cr-predicates]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/predicate
[secondary-resources-doc]: watching-resources/secondary-owned-resources
[predicates-with-external-type-doc]: watching-resources/predicates-with-watch
[cr-owner-ref-doc]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller/controllerutil#SetOwnerReference
