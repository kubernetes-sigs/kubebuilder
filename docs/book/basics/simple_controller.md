{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Simple Controller Example

This chapter walks through a simple Controller implementation.

This is a simple example of the Controller for the ContainerSet API shown in *Simple Resource Example*.

> $ kubebuilder create resource --group workloads --version v1beta1 --kind ContainerSet

> pkg/controller/containerset/controller.go

## Setup

{% method %}

Code generation requires the following to be defined in controller.go:
 
- a `ProvideController` function returning an initialized `controller.GenericController`.  This
  will be called from the controller-manager at runtime.
- a go struct annotated with `// +kubebuilder:controller`.  This wires the controller in the `inject` package.

#### ContainerSetController

ContainerSetController has 3 annotations:

- `// +kubebuilder:controller` registers the controller in the generated `pkg/inject` code ensuring it will be
  started by the controller-manager main at runtime.
- `// +kubebuilder:rbac` creates RBAC rules in the `hack/install.yaml` file created by `kubebuilder create config`.
  This will ensure the Kubernetes ServiceAccount running the controller can read / write to the Deployment API
- `// +kubebuilder:informers` starts the informer for listening to Deployment events

ContainerSetController has 2 variables:

- `InjectArgs` contains the clients provided to *ProvideController*
- `containersetrecorder` contains a recorder to publish information displayed by `kubectl describe`.

#### ProvideController

ProviderController configures a new controller to watch for changes and call Reconcile.

ProvideController will be called from `pkg/inject/zz_generated.kubebuilder.go` after running `kubebuider generate`.

- Create a new `ContainerSetController` struct
- Create a new `GenericController` with the `ContainerSetController` Reconcile function
- Watch for events on *ContainerSet*s and call Reconcile for the key
- Watch for events on *Deployment*s and call Reconcile for the key of the Owning ContainerSet
- Return the `GenericController` from the function

**Note:** when watching the Deployment, a Predicate is used to filter events where the
ResourceVersion of the Deployment have not changed.  This is an optimization to filter
out Deployment events that don't require a reconcile.

#### Reference

- See the [controller libraries](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller) godocs
for reference documentation on watches.
- See the [controller code generation tags](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller)
godocs for reference documentation on controller annotations.


{% sample lang="go" %}
```go
// +kubebuilder:controller:group=workloads,version=v1alpha1,kind=ContainerSet,resource=containersets
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:informers:group=apps,version=v1,kind=Deployment
type ContainerSetController struct {
	args.InjectArgs
    containersetrecorder record.EventRecorder
}

func ProvideController(arguments args.InjectArgs) (
	    *controller.GenericController, error) {
    bc := &ContainerSetController{
		InjectArgs: arguments,
        containersetrecorder: arguments.CreateRecorder(
        	"ContainerSetController"),
    }

    gc := &controller.GenericController{
        Name: "ContainerSetController",
        Reconcile: bc.Reconcile,
        InformerRegistry: arguments.ControllerManager,
    }

    // Watch ContainerSet
    if err := gc.Watch(&workloadsv1alpha1.ContainerSet{});
        err != nil {
        return gc, err
    }

    // Watch Deployments
    containerSetLookup := func(k types.ReconcileKey) (
    	interface{}, error) {
        d, err := bc.Clientset.
        	WorkloadsV1alpha1().
        	ContainerSets(k.Namespace).
        	Get(k.Name, metav1.GetOptions{})
        return d, err
    }
    if err := gc.WatchControllerOf(
    	&appsv1.Deployment{}, 
    	eventhandlers.Path{containerSetLookup},
        predicates.ResourceVersionChanged); err != nil {
            return gc, err
    }
    return gc, nil
}
```
{% endmethod %}

{% panel style="warning", title="Adding Annotations For Watches And CRUD Operations" %}
It is critical to add the `// +kubebuilder:informers` and `// +kubebuilder:rbac` annotations when
adding watches or CRUD operations to your controller through either `GenericController.Watch*`
or CRUD (e.g. `.Update`) operations.

After updating the annotations, `kubebuilder generate` must be rerun to regenerated code, and
`kubebuilder create config` must be run to regenerated installation yaml with the rbac rules.
{% endpanel %}


## Reconcile

{% panel style="success", title="Level vs Edge" %}
The Reconcile function does not differentiate between create, update or deletion events.
Instead it simply reads the desired state defined in ContainerSet.Spec and compares it
to the observed state.
{% endpanel %}

{% method %}

The business logic of the controller is implemented in the `Reconcile` function.  This function takes the *key* of a
ContainerSet, allowing multiple Events to be batched together into a single Reconcile call.

The function shown here creates or updates a Deployment using the replicas and image specified in
ContainerSet.Spec.  Note that it creates an OwnerReference for the Deployment to enable garbage collection
once the ContainerSet is deleted.

1. Read the ContainerSet using the ReconcileKey
2. If there is an error or it has been deleted, return
3. Create the new desired DeploymentSpec from the ContainerSetSpec
4. Read the Deployment and compare the Deployment.Spec to the ContainerSet.Spec
5. If the observed Deployment.Spec does not match the desired spec
  - Deployment was not found: create a new Deployment
  - Deployment was found and changes are needed: update the Deployment

{% sample lang="go" %}
```go
func (bc *ContainerSetController) Reconcile(
	k types.ReconcileKey) error {
    
    // Read the ContainerSet state
    cs, err := bc.Clientset.
    	WorkloadsV1alpha1().
    	ContainerSets(k.Namespace).
    	Get(k.Name, metav1.GetOptions{})
    if err != nil{
        if errors.IsNotFound(err) {
            return nil
        }
        return err
    }

    // Create the canonical DeploymentSpec
	spec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"container-set": k.Name},
		},
		Replicas: &cs.Spec.Replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"container-set": k.Name},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: k.Name,
						Image: cs.Spec.Image,
					},
				},
			},
		},
	}

	// Read the DeploymentState
    dep, err := bc.KubernetesClientSet.
    	AppsV1().
    	Deployments(k.Namespace).
    	Get(k.Name, metav1.GetOptions{})
    if errors.IsNotFound(err) {
    	// Create the Deployment
        dep = &appsv1.Deployment{
        	Spec: spec,
		}
		// Set OwnerReferences so the Deployment is GCed
		dep.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(cs, schema.GroupVersionKind{
				Group:   "workloads.k8s.io",
				Version: "v1alpha1",
				Kind:    "ContainerSet",
			}),
		}
		dep.Name = k.Name
		dep.Namespace = k.Namespace
		_, err = bc.KubernetesClientSet.AppsV1().
			Deployments(k.Namespace).Create(dep)
	} else {
		// Update the Deployment iff its observed Spec does
		// not matched the desired Spec
		image := dep.Spec.Template.Spec.Containers[0].Image
		replicas := *dep.Spec.Replicas
		if replicas == cs.Spec.Replicas &&
			image == cs.Spec.Image {
			return nil
		}
		dep.Name = k.Name
		dep.Namespace = k.Namespace
		dep.Spec = spec
		_, err = bc.KubernetesClientSet.AppsV1().
			Deployments(k.Namespace).Update(dep)
	}
    if err != nil {
        return err
    }
    return nil
}
```
{% endmethod %}
