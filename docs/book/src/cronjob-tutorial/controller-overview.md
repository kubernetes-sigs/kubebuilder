# What's in a controller?

Controllers are the core of Kubernetes, and of any operator.

It's a controller's job to ensure that, for any given object, the actual
state of the world (both the cluster state, and potentially external state
like running containers for Kubelet or loadbalancers for a cloud provider)
matches the desired state in the object.  Each controller focuses on one
*root* Kind, but may interact with other Kinds.

This process is called *reconciling*.

In controller-runtime, the logic that implements the reconciling for
a specific kind is called a [*Reconciler*](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile?tab=doc).  A reconciler
takes the name of an object, and returns whether or not to try
again (e.g. in case of errors or periodic controllers, like the
HorizontalPodAutoscaler).

{{#literatego ./testdata/emptycontroller.go}}

Now that you've seen the basic structure of a reconciler, fill out
the logic for `CronJob`s.
