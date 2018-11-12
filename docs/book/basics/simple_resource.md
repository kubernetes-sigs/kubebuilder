# Resource Example

This chapter walks through the definition of a new Resource call *ContainerSet*.  ContainerSet
contains the image and replicas fields, and ensures a Deployment with matching image and replicas
it running in the cluster.

Create the scaffolding for a new resource using the kubebuilder cli:

> $ kubebuilder create api --group workloads --version v1beta1 --kind ContainerSet

This creates several files, including the Resource schema definition in:

> pkg/apis/workloads/v1beta1/containerset_types.go

{% method %}
## Type Definition

ContainerSet has 4 fields:

- Spec contains the desired cluster state specified by the object.  While much of the Spec is
  defined by users, unspecified parts may be filled in with defaults or by Controllers such as autoscalers.
- Status contains only *observed cluster state* and is only written by controllers
  Status is not the source of truth for any information, but instead aggregates and publishes observed state.
- TypeMeta contains metadata about the API itself - such as Group, Version, Kind.
- ObjectMeta contains metadata about the specific object instance - such as the name, namespace,
  labels and annotations.  ObjectMeta contains data common to most objects.

#### Reference

- See the [resource code generation tags](https://godoc.org/sigs.k8s.io/kubebuilder/pkg/gen/apis)
godocs for reference documentation on resource annotations.

{% sample lang="go" %}
```go
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ContainerSet creates a new Deployment running multiple replicas of a single container with the given
// image.
// +k8s:openapi-gen=true
// +resource:path=containersets
type ContainerSet struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    // spec contains the desired behavior of the ContainerSet
    Spec   ContainerSetSpec   `json:"spec,omitempty"`

    // status contains the last observed state of the ContainerSet
    Status ContainerSetStatus `json:"status,omitempty"`
}
```
{% endmethod %}

{% panel style="info", title="Comment annotation directives" %}
The definition contains several comment annotations of the form `// +something`.  These are
used to configure code generators to run against this code.  The code generators will 
generate boilerplate functions and types to complete the Resource definition.
To learn more on configuring code generation see the *Code Generation* chapter.

To learn more about how to use annotations in kubebuilder, refer to [Using Annotation](../beyond_basics/annotations.md)
{% endpanel %}

{% method %}
## ContainerSetSpec

The ContainerSetSpec contains the container image and replica count, which should be read by
the controller and used to create and manage a new Deployment.  The Spec field contains desired
state defined by the user or, if unspecified, field defaults or Controllers set values.
An example of an unspecified field that could be owned by a Controller would be the `replicas`
field, which may be set by autoscalers.

{% sample lang="go" %}
```go
// ContainerSetSpec defines the desired state of ContainerSet
type ContainerSetSpec struct {
  // replics is the number of replicas to maintain
  Replicas int32 `json:"replicas,omitempty"`

  // image is the container image to run.  Image must have a tag.
  // +kubebuilder:validation:Pattern=.+:.+
  Image string `json:"image,omitempty"`
}
```
{% endmethod %}

{% method %}
## ContainerSetStatus

The ContainerSetStatus contains the number of healthy replicas, and should be set by the controller
each time the ContainerSet is reconciled.

This field is propagated from the DeploymentStatus, and so the controller must watch for Deployment
events to update the field.

{% sample lang="go" %}
```go
// ContainerSetStatus defines the observed state of ContainerSet
type ContainerSetStatus struct {
  HealthyReplicas int32 `json:"healthyReplicas,omitempty"`
}
```
{% endmethod %}

{% panel style="info", title="Running Code Generators" %}
While users don't directly modify generated code, the code must be regenerated after resources are
modified by adding or removing fields.  This is automatically done when running `make`.

Code generation may be configured for resources using annotations of the form `// +something`.
See the [pkg/gen](https://godoc.org/sigs.k8s.io/kubebuilder/pkg/gen/) reference documentation.
{% endpanel %}

{% method %}
## Scaffolded Boilerplate

Kubebuilder scaffolds boilerplate code to register resources with the runtime.Scheme used to
map go structs to GroupVersionKinds.

- SchemeGroupVersion is the GroupVersion for the APIs in this package
- SchemeBuilder should have every API in the package type added to it

{% sample lang="go" %}

```go
var (	
  // SchemeGroupVersion is group version used to register these objects
  SchemeGroupVersion = schema.GroupVersion{Group: "workloads.k8s.io", Version: "v1beta1"}

  // SchemeBuilder is used to add go types to the GroupVersionKind scheme
  SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion})
```

```go
func init() {
  // Register the types with the SchemeBuilder
  SchemeBuilder.Register(&v1.ContainerSet{}, &v1.ContainerSetList{})
}
```
{% endmethod %}

