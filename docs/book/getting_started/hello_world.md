{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Hello World

{% panel style="warning", title="Note on project structure" %}
Kubernete APIs require boilerplate code that is not shown here and is managed by kubebuilder.

Project structure may be created by running `kubebuilder init` and then creating a
new API with `kubebuilder create resource`. More on this topic in
[Project Creation and Structure](../basics/project_creation_and_structure.md) 
{% endpanel %}

This chapter shows an abridged Kubebuilder project for a simple API.

Kubernetes APIs have 3 components.  These components live in separate go packages:

* The API schema definition, or *Resource*, as a go struct.  This implicitly defines endpoints.
* The API implementation, or *Controller*, as a go function.
* The executable, or controller-manager, as a go main.

{% method %}
## Pancake API Resource Definition {#hello-world-api}

This is a Resource definition.  It is a go struct containing the API schema that
implicitly defines CRUD endpoints for the Resource.

For a more information on Resources see [What is a Resource](../basics/what_is_a_resource.md).

While it is not shown here, most Resources will split their fields in into a Spec and a Status field.

For a more complete example see [Simple Resource Example](../basics/simple_resource.md) 

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

This is a Controller implementation.  It contains a Reconcile function which takes an object
key as an argument and reconciles the observed state of the cluster with the desired state of the cluster.

Reconcile should be trigger by watch events for the Pancake Resource type, but may also be triggered
by watch events for related resource types, such as for any objects created by Reconcile. 

When Reconcile is triggered by watch events for other resource types, each event is
mapped to the key of a Pancake object.  This will trigger a full reconcile of
the Pancake object, which will in turn read related cluster state, including the object the
original event was for.

For a more information on Controllers see [What is a Controller](../basics/what_is_a_controller.md).

The code shown here has been abridged; for a more complete example see
[Simple Controller Example](../basics/simple_controller.md)

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

