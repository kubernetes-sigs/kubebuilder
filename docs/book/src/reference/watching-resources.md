# Watching Resources

Inside a `Reconcile()` control loop, you are looking to do a collection of operations until it has the desired state on the cluster.
Therefore, it can be necessary to know when a resource that you care about is changed.
In the case that there is an action (create, update, edit, delete, etc.) on a watched resource, `Reconcile()` should be called for the resources watching it.

[Controller Runtime libraries](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/builder) provide many ways for resources to be managed and watched.
This ranges from the easy and obvious use cases, such as watching the resources which were created and managed by the controller, to more unique and advanced use cases.

See each subsection for explanations and examples of the different ways in which your controller can _Watch_ the resources it cares about.

- [Watching Operator Managed Resources](watching-resources/operator-managed.md) -
  These resources are created and managed by the same operator as the resource watching them.
  This section covers both if they are managed by the same controller or separate controllers.
- [Watching Externally Managed Resources](watching-resources/externally-managed.md) -
  These resources could be manually created, or managed by other operators/controllers or the Kubernetes control plane.