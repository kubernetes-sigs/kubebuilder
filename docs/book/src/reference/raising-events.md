# Creating Events

It is often useful to publish *Event* objects from the controller Reconcile function as they allow users or any automated processes to see what is going on with a particular object and respond to them.

 Recent Events for an object can be viewed by running `$ kubectl describe <resource kind> <resource name>`. Also, they can be checked by running `$ kubectl get events`.

<aside class="warning">
<h1>Events should be raised in certain circumstances only</h1>

Be aware that it is **not** recommended to emit Events for all operations. If authors raise too many events, it brings bad UX experiences for those consuming the solutions on the cluster, and they may find it difficult to filter an actionable event from the cluster. For more information, please take a look at the [Kubernetes APIs convention][Events].

</aside>

## Writing Events

Anatomy of an Event:

```go
Event(object runtime.Object, eventtype, reason, message string)
```

- `object` is the object this event is about.
- `eventtype` is this event type, and is either *Normal* or *Warning*. ([More info][Event-Example])
- `reason` is the reason this event is generated. It should be short and unique with `UpperCamelCase` format. The value could appear in *switch* statements by automation. ([More info][Reason-Example])
- `message` is intended to be consumed by humans. ([More info][Message-Example])



<aside class="note">
<h1>Example Usage</h1>

Following is an example of a code implementation that raises an Event.

```go
	// The following implementation will raise an event
	r.Recorder.Eventf(cr, "Warning", "Deleting",
		"Custom Resource %s is being deleted from the namespace %s",
		cr.Name, cr.Namespace)
```

</aside>

### How to be able to raise Events?

Following are the steps with examples to help you raise events in your controller's reconciliations.
Events are published from a Controller using an [EventRecorder][Events]`type CorrelatorOptions struct`,
which can be created for a Controller by calling `GetRecorder(name string)` on a Manager. See that we will change the implementation scaffolded in `cmd/main.go`:

```go
	if err = (&controller.MyKindReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		// Note that we added the following line:
		Recorder: mgr.GetEventRecorderFor("mykind-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MyKind")
		os.Exit(1)
	}
```

### Allowing usage of EventRecorder on the Controller

To raise an event, you must have access to `record.EventRecorder` in the Controller.  Therefore, firstly let's update the controller implementation:
```go
import (
	...
	"k8s.io/client-go/tools/record"
	...
)
// MyKindReconciler reconciles a MyKind object
type MyKindReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	// See that we added the following code to allow us to pass the record.EventRecorder
	Recorder record.EventRecorder
}
```
### Passing the EventRecorder to the Controller

Events are published from a Controller using an [EventRecorder]`type CorrelatorOptions struct`,
which can be created for a Controller by calling `GetRecorder(name string)` on a Manager. See that we will change the implementation scaffolded in `cmd/main.go`:

```go
	if err = (&controller.MyKindReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		// Note that we added the following line:
		Recorder: mgr.GetEventRecorderFor("mykind-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MyKind")
		os.Exit(1)
	}
```

### Granting the required permissions

You must also grant the RBAC rules permissions to allow your project to create Events. Therefore, ensure that you add the [RBAC][rbac-markers] into your controller:

```go
...
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
...
func (r *MyKindReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
```

And then, run `$ make manifests` to update the rules under `config/rbac/role.yaml`.

[Events]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#events
[Event-Example]: https://github.com/kubernetes/api/blob/6c11c9e4685cc62e4ddc8d4aaa824c46150c9148/core/v1/types.go#L6019-L6024
[Reason-Example]: https://github.com/kubernetes/api/blob/6c11c9e4685cc62e4ddc8d4aaa824c46150c9148/core/v1/types.go#L6048
[Message-Example]: https://github.com/kubernetes/api/blob/6c11c9e4685cc62e4ddc8d4aaa824c46150c9148/core/v1/types.go#L6053
[rbac-markers]: ./markers/rbac.md
