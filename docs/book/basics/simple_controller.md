# Controller Example

This chapter walks through a simple Controller implementation.

This example is for the Controller for the ContainerSet API shown in the [Resource Example](simple_resource.md).
It uses the [controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime/pkg) libraries
to implement the Controller and Manager.

Unlike the Hello World example, here we use the underlying Controller libraries directly instead
of the higher-level `application` pattern libraries.  This gives greater control over
the Controller is configured.

> $ kubebuilder create api --group workloads --version v1beta1 --kind ContainerSet

> pkg/controller/containerset/containerset_controller.go

## Setup

{% method %}
#### ContainerSetController

ContainerSetController has a single annotation:

- `// +kubebuilder:rbac` creates RBAC rules in the `config/rbac/rbac_role.yaml` file when `make` is run.
  This will ensure the Kubernetes ServiceAccount running the controller can read / write to the Deployment API.

ContainerSetController has 2 variables:

- `client.Client` is a client for reading / writing Kubernetes APIs.
- `scheme *runtime.Scheme` is a runtime.Scheme used by the library to set OwnerReferences.

#### Adding a Controller to the Manager

Add creates a new Controller that will be started by the Manager.  When adding a Controller it is important to setup
Watch functions to trigger Reconciles.

Watch is a function that takes an event `source.Source` and a `handler.EventHandler`.  The Source provides events
for some type, and the EventHandler responds to events by enqueuing `reconcile.Request`s for objects.
Watch optionally takes a list of Predicates that may be used to filter events.

Sources

- To watch for create / update / delete events for an object use a `source.KindSource` e.g.
`source.KindSource{Type: &v1.Pod}`

Handlers

- To enqueue a Reconcile for the object in the event use a `handler.EnqueueRequestForObject`
- To enqueue a Reconcile for the owner object that created the object in the event use a `handler.EnqueueRequestForOwner`
  with the type of the owner e.g. `&handler.EnqueueRequestForOwner{OwnerType: &appsv1.Deployment{}, IsController: true}`
- To enqueue Reconcile requests for an arbitrary collection of objects in response to the event, use a
  `handler.EnqueueRequestsFromMapFunc`.

Example:

- Create a new `ContainerSetController` struct that will.
  - Invoke Reconcile with the Name and Namespace of a *ContainerSet* for *ContainerSet* create / update / delete events
  - Invoke Reconcile with the Name and Namespace of a *ContainerSet* for *Deployment* create / update / delete events

#### Reference

- See the [controller libraries](https://godoc.org/sigs.k8s.io/controller-runtime/pkg)
godocs for reference documentation on the controller libraries.
- See the [Using Annotations](../beyond_basics/annotations.md) to learn more
about hot use annotations in kubebuilder.

{% sample lang="go" %}
```go
type ContainerSetController struct {
  client.Client
  scheme *runtime.Scheme
}

func Add(mgr manager.Manager) error (
  // Create a new Controller
  c, err := controller.New("containerset-controller", mgr,
    controller.Options{Reconciler: &ContainerSetController{
      Client: mgr.GetClient(),
      scheme: mgr.GetScheme(),
  }})
  if err != nil {
    return err
  }

  // Watch for changes to ContainerSet
  err = c.Watch(
    &source.Kind{Type:&workloadsv1beta1.ContainerSet{}},
      &handler.EnqueueRequestForObject{})
  if err != nil {
    return err
  }

    // Watch for changes to Deployments created by a ContainerSet and trigger a Reconcile for the owner
  err = c.Watch(
    &source.Kind{Type: &appsv1.Deployment{}},
      &handler.EnqueueRequestForOwner{
        IsController: true,
        OwnerType:    &workloadsv1beta1.ContainerSet{},
      })
  if err != nil {
    return err
  }

  return nil
}
```
{% endmethod %}

{% panel style="warning", title="Adding Annotations For Watches And CRUD Operations" %}
It is important`// +kubebuilder:rbac` annotations when adding Watches or CRUD operations
so that when the Controller is deployed it will have the correct permissions.

`make` must be run anytime annotations are changed to regenerated code and configs.
{% endpanel %}


## Implementing Controller Reconcile

{% panel style="success", title="Level vs Edge" %}
The Reconcile function does not differentiate between create, update or deletion events.
Instead it simply reads the state of the cluster at the time it is called.
{% endpanel %}

Reconcile uses a `client.Client` to read and write objects.  The Client is able to
read or write any type of runtime.Object (e.g. Kubernetes object), so users don't need
to generate separate clients for each collection of APIs.

{% method %}

The business logic of the Controller is implemented in the `Reconcile` function.  This function takes the Namespace
 and Name of a ContainerSet, allowing multiple Events to be batched together into a single Reconcile call.

The function shown here creates or updates a Deployment using the replicas and image specified in
ContainerSet.Spec.  Note that it sets an OwnerReference for the Deployment to enable garbage collection
on the Deployment once the ContainerSet is deleted.

1. Read the ContainerSet using the NamespacedName
2. If there is an error or it has been deleted, return
3. Create the new desired DeploymentSpec from the ContainerSetSpec
4. Read the Deployment and compare the Deployment.Spec to the ContainerSet.Spec
5. If the observed Deployment.Spec does not match the desired spec
  - Deployment was not found: create a new Deployment
  - Deployment was found and changes are needed: update the Deployment

{% sample lang="go" %}
```go
var _ reconcile.Reconciler = &ContainerSetController{}

func (r *ReconcileContainerSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
  instance := &workloadsv1beta1.ContainerSet{}
  err := r.Get(context.TODO(), request.NamespacedName, instance)
  if err != nil {
    if errors.IsNotFound(err) {
      // Object not found, return.  Created objects are automatically garbage collected.
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
  }
  return reconcile.Result{}, nil
}
```
{% endmethod %}
