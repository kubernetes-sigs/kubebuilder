{% panel style="danger", title="Staging" %}
Staging documentation under review.
{% endpanel %}

# Hello World

{% panel style="warning", title="Note on project structure" %}
Kubernete APIs require boilerplate code that is not shown here and is managed by kubebuilder.

Project structure may be created by running `kubebuilder init` and then creating a
new API with `kubebuilder create resource`. More on this topic in *Project Creation and Structure*
{% endpanel %}

This chapter shows an abridged Kubebuilder project for a simple API.

Kubernetes APIs have 3 components.  These components live in separate go packages:

* The API schema definition, or *Resource*, as a go struct.  This implicitly defines endpoints.
* The API implementation, or *Controller*, as a go function.
* The executable, or controller-manager, as a go main.

{% method %}
## Pancake API Resource Definition {#hello-world-api}

This is a Resource definition.  It is a go struct containing the API schema and it
implicitly defines CRUD endpoints the Resource.

While it is not shown here, most Resources will split their fields in into Spec and Status fields.

{% sample lang="go" %}
```go
type Pancake struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Message string `json:"message"`
}
```
{% endmethod %}

{% method %}
## Pancake Controller {#hello-world-controller}

This is a Controller implementation.  It is a Reconcile function which takes the key of an object
and reconciles the observed state of the cluster with the desired state of the cluster.

Reconcile should be trigger by watch events for the Pancake Resource type, but may also be triggered
by watch events for related resource types, such as for objects created by Reconcile. 

When Reconcile is triggered by watch events for other resource types, each event is
mapped to the key of a Pancake object.  This will trigger a full reconcile of
the Pancake object, which will in turn read related objects, including the object the
original event was for.

The code shown here has been abridged; for a more complete example see *Simple Controller*.

{% sample lang="go" %}
```go
// Note: This code lives under
// pkg/controller/pancake/controller.go

func (bc *PancakeController) Reconcile(k types.ReconcileKey) error {
    p, err := bc.pancakeclient.
    	Pancakes(k.Namespace).
    	Get(k.Name, v1.GetOptions{})
    if err != nil {
        return err
    }
    fmt.Println(p.Spec.Message)
    return nil
}

func ProvideController(arguments args.InjectArgs) (
	*controller.GenericController, error) {
    ...
    
    if err := gc.Watch(&breakfastv1alpha1.Pancake{}); err != nil {
        return gc, err
    }
    ...
}
```
{% endmethod %}

