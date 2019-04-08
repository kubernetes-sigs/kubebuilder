# Controller Watch Functions

This chapter describes how to use the controller package functions to configure Controllers to watch
Resources.

[Link to reference documentation](https://godoc.org/sigs.k8s.io/controller-runtime)

{% method %}
## Watching the Controller Resource

Controllers may watch Resources and trigger Reconcile calls with the key of the
object from the watch event.

This example configures a controller to watch for Pod events, and call Reconcile with
the Pod key.

If Pod *default/foo* is created, updated or deleted, then Reconcile will be called with
*namespace: default, name: foo*

{% sample lang="go" %}
```go
// Annotation for generating RBAC role to Watch Pods
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list
```

```go
// Watch for Pod events, and enqueue a reconcile.Request to trigger a Reconcile
err := c.Watch(
	&source.Kind{Type: &v1.Pod{}},
	&handler.EnqueueRequestForObject{})
if err != nil {
	return err
}
```

```go
// You can also watch unstructured objects
u := &unstructured.Unstructured{}
u.SetGroupVersionKind(schema.GroupVersionKind{
	Kind:    "Pod",
	Group:   "",
	Version: "v1",
})

err = c.Watch(&source.Kind{Type: u}, &handler.EnqueueRequestForObject{})
```
{% endmethod %}


{% method %}
## Watching Created Resources

Controllers may watch Resources of types they create and trigger Reconcile calls with the key of
the Owner of the object.

This example configures a Controller to watch for Pod events, and call Reconcile with
the Owner ReplicaSet key.  This is done by looking up the object referred to by the Owner reference
from the watch event object.

- Define a function to lookup the Owner from the key
- Call `WatchControllerOf` with the Owned object and the function to lookup the owner

If Pod *default/foo-pod* was created by ReplicaSet *default/foo-rs*, and the Pod is
(re)created, updated or deleted, then Reconcile will be called with *namespace: default, name: foo-rs*

**Note:** This requires adding the following annotations to your Controller struct to ensure the
correct RBAC rules are in place and informers have been started.

```go
// Annotation to generate RBAC roles to watch and update Pods
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list;create;update;delete
```

{% sample lang="go" %}
```go
// Watch for Pod events, and enqueue a reconcile.Request for the ReplicaSet in the OwnerReferences
err := c.Watch(
	&source.Kind{Type: &corev1.Pod{}},
	&handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:	&appsv1.ReplicaSet{}})
if err != nil {
	return err
}
```
{% endmethod %}

{% method %}
## Watching Arbitrary Resources

Controllers may watch arbitrary Resources and map them to a key of the Resource managed by the
controller.  Controllers may even map an event to multiple keys, triggering Reconciles for
each key.

Example: To respond to cluster scaling events (e.g. the deletion or addition of Nodes),
a Controller would watch Nodes and map the watch events to keys of objects managed by
the controller.

This simple example configures a Controller to watch for Pod events, and then reconciles objects with
names derived from the Pod's name.

If Pod *default/foo* is created, updated or deleted, then Reconcile will be called for
*namespace: default, name: foo-parent-1* and for *namespace: default, name: foo-parent-2*.

**Note:** This requires adding the following annotations to your Controller struct to ensure the
correct RBAC rules are in place and informers have been started.

```go
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list
```

{% sample lang="go" %}
```go
// Define a mapping from the object in the event to one or more
// objects to Reconcile
mapFn := handler.ToRequestsFunc(
	func(a handler.MapObject) []reconcile.Request {
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				Name:	  a.Meta.GetName() + "-1",
				Namespace: a.Meta.GetNamespace(),
			}},
			{NamespacedName: types.NamespacedName{
				Name:	  a.Meta.GetName() + "-2",
				Namespace: a.Meta.GetNamespace(),
			}},
		}
	})


// 'UpdateFunc' and 'CreateFunc' used to judge if a event about the object is
// what we want. If that is true, the event will be processed by the reconciler.
p := predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		// The object doesn't contain label "foo", so the event will be
		// ignored.
		if _, ok := e.MetaOld.GetLabels()["foo"]; !ok {
			return false
		}
		return e.ObjectOld != e.ObjectNew
	},
	CreateFunc: func(e event.CreateEvent) bool {
		if _, ok := e.Meta.GetLabels()["foo"]; !ok {
			return false
		}
		return true
	},
}

// Watch Deployments and trigger Reconciles for objects
// mapped from the Deployment in the event
err := c.Watch(
	&source.Kind{Type: &appsv1.Deployment{}},
	&handler.EnqueueRequestsFromMapFunc{
		ToRequests: mapFn,
	},
	// Comment it if default predicate fun is used.
	p)
if err != nil {
	return err
}
```
{% endmethod %}


{% method %}
## Watching Channels

Controllers may trigger Reconcile for events written to Channels.  This is useful if the Controller
needs to trigger a Reconcile in response to something other than a create / update / delete event
to a Kubernetes object.  Note: in most situations this case is better handled by updating a Kubernetes
object with the external state that would trigger the Reconcile.

{% sample lang="go" %}
```go
events := make(chan event.GenericEvent)
err := ctrl.Watch(
	&source.Channel{Source: events},
	&handler.EnqueueRequestForObject{},
)
if err != nil {
	return err
}
```
{% endmethod %}
