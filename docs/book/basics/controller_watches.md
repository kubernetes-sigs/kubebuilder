{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Controller Watch Functions

This chapter describes how to use the controller package functions to configure Controllers to watch
Resources.

[Link to reference documentation](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller)

{% method %}
## Watching Controller Resource

Controllers may watch Resources and trigger Reconcile calls with the key of the
object from the watch event. 

This example configures a controller to watch for Pod events, and call Reconcile with
the Pod key.

If Pod *default/foo* is created, updated or deleted, then Reconcile will be called with
*namespace: default, name: foo*

{% sample lang="go" %}
```go
if err := c.Watch(&v1.Pod{}); err != nil {
    log.Fatalf("%v", err)
}
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
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list
// +kubebuilder:informers:group=core,version=v1,kind=Pod
```

{% sample lang="go" %}
```go
fn := func(k types.ReconcileKey) (interface{}, error) {
    return informerFactory.
    	Apps().V1().
    	ReplicaSets().
    	Lister().
    	ReplicaSets(k.Namespace).Get(k.Name)
}
if err := c.WatchControllerOf(
	&corev1.Pod{}, eventhandlers.Path{fn}
); err != nil {
    log.Fatalf("%v", err)
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
// +kubebuilder:informers:group=core,version=v1,kind=Pod
```

{% sample lang="go" %}
```go
if err := c.WatchTransformationKeysOf(&corev1.Pod{},
    func(i interface{}) []types.ReconcileKey {
        p, ok := i.(*corev1.Pod)
        if !ok {
            return []types.ReconcileKey{}
        }

        // Find multiple parents based off the name
        n := strings.Split(p.Name, "-")[0]
        return []types.ReconcileKey{
            {p.Namespace, n + "-mapto-1"},
            {p.Namespace, n + "-mapto-2"},
        }
    },
); err != nil {
    log.Fatalf("%v", err)
	
}
```
{% endmethod %}


{% method %}
## Watching Channels

Controllers may watch channels for events to trigger Reconciles.  This is useful if the Controller
manages some external state that it is either polled or calls back via a WebHook.

This simple example configures a Controller to read `namespace/name` keys from a channel and
trigger Reconciles.

If podkeys has *default/foo* inserted, then Reconcile will be called for *namespace: default, name: foo*.

{% sample lang="go" %}
```go
podkeys := make(chan string)
if err := c.WatchChannel(podkeys); err != nil {
    log.Fatalf("%v", err)
}
```
{% endmethod %}
