# Creating Events

It is often useful to publish *Event* objects from the controller Reconcile function. 

Events allow users to see what is going on with a particular object, and allow automated processes to see and respond to them.

{% panel style="success", title="Getting Events" %}
Recent Events for an object can be viewed by running `kubectl describe`
{% endpanel %}

{% method %}

Events are published from a Controller using an [EventRecorder](https://github.com/kubernetes/client-go/blob/master/tools/record/event.go#L56),
which can be created for a Controller by calling `GetRecorder(name string)` on a Manager.

`Name` should be identifiable and descriptive as it will appear in the `From` column of `kubectl describe` command.

{% sample lang="go" %}
```go
// Annotation for generating RBAC role for writing Events
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
```

```go
// ReconcileContainerSet reconciles a ContainerSet object
type ReconcileContainerSet struct {
  client.Client
  scheme *runtime.Scheme
  recorder record.EventRecorder
}
```
```go
// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
  return &ReconcileContainerSet{
    Client:   mgr.GetClient(),
    scheme:   mgr.GetScheme(),
    recorder: mgr.GetRecorder("containerset-controller"),
  }
}
```
{% endmethod %}

{% method %}

## Writing Events

Anatomy of an Event:

```go
Event(object runtime.Object, eventtype, reason, message string)
```

- `object` is the object this event is about.
- `eventtype` is the type of this event, and is either *Normal* or *Warning*.
- `reason` is the reason this event is generated. It should be short and unique with `UpperCamelCase` format. The value could appear in *switch* statements by automation.
- `message` is intended to be consumed by humans.


Building on the example introduced in [Controller Example](../basics/simple_controller.md), we can add Events to our reconcile logic using `recorder` as our `EventRecorder`

{% sample lang="go" %}
```go
  //Reconcile logic up here...

  // Create the resource
  found := &appsv1.Deployment{}
  err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
  if err != nil && errors.IsNotFound(err) {
    log.Printf("Creating Deployment %s/%s\n", deploy.Namespace, deploy.Name)
    err = r.Create(context.TODO(), deploy)
    if err != nil {
      return reconcile.Result{}, err
    }
    
    // Write an event to the ContainerSet instance with the namespace and name of the 
    // created deployment
    r.recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created deployment %s/%s", deploy.Namespace, deploy.Name))
  
  } else if err != nil {
    return reconcile.Result{}, err
  }

  // Preform update
  if !reflect.DeepEqual(deploy.Spec, found.Spec) {
    found.Spec = deploy.Spec
    log.Printf("Updating Deployment %s/%s\n", deploy.Namespace, deploy.Name)
    err = r.Update(context.TODO(), found)
    if err != nil {
      return reconcile.Result{}, err
    }

    // Write an event to the ContainerSet instance with the namespace and name of the 
    // updated deployment
    r.recorder.Event(instance, "Normal", "Updated", fmt.Sprintf("Updated deployment %s/%s", deploy.Namespace, deploy.Name))
  
  }
  return reconcile.Result{}, nil
}
```

{% endmethod %}
