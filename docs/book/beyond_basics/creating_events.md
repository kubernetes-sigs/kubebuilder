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
var _ reconcile.Reconciler = &ContainerSetController{}

func (r *ReconcileContainerSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
  instance := &workloadsv1beta1.ContainerSet{}
  err := r.Get(context.TODO(), request.NamespacedName, instance)
  if err != nil {
    if errors.IsNotFound(err) {
      // Object not found, return. Created objects are automatically garbage collected.
      // For additional cleanup logic use finalizers.
      return reconcile.Result{}, nil
    }
    // Error reading the object - requeue the request.
    return reconcile.Result{}, err
  }

  // TODO(user): Change this to be the object type created by your controller
  // Define the desired Deployment object
  deploy := &appsv1.Deployment{
    ObjectMeta: metav1.ObjectMeta{
      Name:      instance.Name + "-deployment",
      Namespace: instance.Namespace,
    },
    Spec: appsv1.DeploymentSpec{
      Selector: &metav1.LabelSelector{
        MatchLabels: map[string]string{"deployment": instance.Name + "-deployment"},
      },
      Replicas: &instance.Spec.Replicas,
      Template: corev1.PodTemplateSpec{
        ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"deployment": instance.Name + "-deployment"}},
        Spec: corev1.PodSpec{
          Containers: []corev1.Container{
            {
                Name:  instance.Name,
                Image: instance.Spec.Image,
            },
          },
        },
      },
    },
  }
  
  if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
    return reconcile.Result{}, err
  }

  // TODO(user): Change this for the object type created by your controller
  // Check if the Deployment already exists
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

  // TODO(user): Change this for the object type created by your controller
  // Update the found object and write the result back if there are any changes
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
