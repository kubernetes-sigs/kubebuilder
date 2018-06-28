# Simple Controller Example

This chapter walks through a simple Controller implementation.

This example is for the Controller for the ContainerSet API shown in *Simple Resource Example*.
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

- `// +kubebuilder:rbac` creates RBAC rules in the `config/rbac/group_kind.yaml` files when `make` is run.
  This will ensure the Kubernetes ServiceAccount running the controller can read / write to the Deployment API

ContainerSetController has 2 variables:

- `client.Client` is a client for reading / writing Kubernetes APIs.
- `scheme *runtime.Scheme` is a runtime.Scheme used by the library to set OwnerReferences.

#### Add

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
- To enqueue a Reconcile for the object that created the object in the event use a `handler.EnqueueRequestForOwner`
  with the type of the owner e.g. `&handler.EnqueueRequestForOwner{OwnerType: &appsv1.Deployment{}, IsController: true}`
- To enqueue Reconcile requests for an arbitrary collection of objects in response to the event, use a
  `handler.EnqueueRequestsFromMapFunc`.

Example:

- Create a new `ContainerSetController` struct that will.
  - Invoke Reconcile with the Name and Namespace of a *ContainerSet* for *ContainerSet* create / update / delete events
  - Invoke Reconcile with the Name and Namespace of a *ContainerSet* for *Deployment* create / update / delete events

#### Reference

- See the [controller libraries](https://godoc.org/sigs.k8s.io/controller-runtime/pkg) godocs for reference
documentation on the controller libraries.
- See the [controller code generation tags](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller)
godocs for reference documentation on controller annotations.


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
	err = c.Watch(&source.Kind{Type: &crewv1.FirstMate{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

    // Watch for changes to Deployments created by a ContainerSet and trigger a Reconcile for the owner
	err = c.Watch(
		&source.Kind{Type: &appsv1.Deployment{}},
	    &handler.EnqueueRequestForOwner{
		    IsController: true,
		    OwnerType:    &crewv1.FirstMate{},
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


## Reconcile

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

func (r *ContainerSetController) Reconcile(request reconcile.Request) (
	reconcile.Result, error) {
    // Read the ContainerSet
	cs := &v1alpha1.ContainerSet{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cs)
    
    // Handle deleted or error case
	if err != nil {
		if errors.IsNotFound(err) {
			// Not found.  Don't worry about cleaning up Deployments,
			// GC will handle it.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

    // Calculate the expected Deployment Spec
	spec := getDeployment(request)

    // Read the Deployment
	dep := &v1.Deployment{}
	err := r.client.Get(context.TODO(), request.NamespacedName, dep)

    // If not found, create it
    if errors.IsNotFound(err) {
        dep = &appsv1.Deployment{Spec: spec}
		dep.Name = request.Name
		dep.Namespace = request.Namespace
    	if err := controllerutil.SetControllerReference(cs, deploy, r.scheme); err != nil {
	    	return reconcile.Result{}, err
	    }
		if err := r.Create(context.TODO(), dep); err != nil {
	    	return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

    // If found, update it
    image := dep.Spec.Template.Spec.Containers[0].Image
    replicas := *dep.Spec.Replicas
    if replicas == cs.Spec.Replicas && image == cs.Spec.Image {
        return reconcile.Result{}, nil
    }
    dep.Spec.Replicas = &cs.Spec.Replicas
    dep.Spec.Template.Spec.Containers[0].Image = cs.Spec.Image
    if err := r.Update(context.TODO(), dep); err != nil {
        return reconcile.Result{}, err
    }
    
    return reconcile.Result{}, nil
}

func getDeployment(request reconcile.Request) *v1.Deployment {
	return &appsv1.DeploymentSpec{
        Selector: &metav1.LabelSelector{
            MatchLabels: map[string]string{
                "container-set": request.Name},
        },
        Replicas: &cs.Spec.Replicas,
        Template: corev1.PodTemplateSpec{
            ObjectMeta: metav1.ObjectMeta{
                Labels: map[string]string{
                    "container-set": request.Name},
            },
            Spec: corev1.PodSpec{
                Containers: []corev1.Container{
                    {
                        Name: request.Name,
                        Image: cs.Spec.Image,
                    },
                },
            },
        },
    }
}
```
{% endmethod %}
