{% panel style="danger", title="STAGING" %}
Staging Environment - Not Official Documentation!

This book contains APIs, libraries and tools that are proposals only and have not been ratified!
{% endpanel %}

# Simple Resource Example

This chapter walks through a simple resource declaration.

> $ kubebuilder create resource --group workloads --version v1beta1 --kind ContainerSet
> pkg/apis/workloads/v1beta1/containerset_types.go

{% method %}
## Type Definition

This is a simple API resource definition example for an API that wraps and simplifies the Deployment API.

ContainerSet has a Spec, Status, TypeMeta and ObjectMeta fields.

- TypeMeta contains metadata about the API itself - such as Group, Version, Kind.
- ObjectMeta contains metadata about the specific object instance - such as the name, namespace, labels and annotations.

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
used to configure code generators to create boilerplate code based on the go struct definition.  For
now we will ignore these.  To learn more on configuring code generation see the *Code Generation* chapter.
{% endpanel %}

{% method %}
## ContainerSetSpec

The ContainerSetSpec contains the container image and replica count, which should be read by
the controller and used to create and manage a new Deployment.

{% sample lang="go" %}
```go
// ContainerSetSpec defines the desired state of ContainerSet
type ContainerSetSpec struct {
    // replics is the number of replicas to maintain
    Replicas int32 `json:"replicas,omitempty"`

    // image is the container image to run
    Image string `json:"image,omitempty"`
}
```
{% endmethod %}

{% method %}
## ContainerSetStatus

The ContainerSetStatus contains the number of healthy replicas, which should be updated
by the controller each time a new Pod becomes healthy.

In order for the controller to keep this field current, it will need to listen for Pod events and map them
back to the owning ContainerSet resource.

{% sample lang="go" %}
```go
// ContainerSetStatus defines the observed state of ContainerSet
type ContainerSetStatus struct {
	HealthyReplicas `json:"healthyReplicas,omitempty"`
}
```
{% endmethod %}

{% panel title="Rerunning Code Generators" %}
While users don't directly modify generated code, users must rerun `kubebuilder generate`
after adding or modifying resources to update the generated code with the changes.
{% endpanel %}

{% method %}
## Generated Boilerplate

Kubebuilder generates boilerplate code required for registering the type with the apimachinery libraries,
implementing required interfaces such as *deep copying* objects, and providing CRD representations of
the resources.

Code generation may be configured by comment annotations of the form `// +something`.

> $ kubebuilder generate

{% sample lang="go" %}
*This code snippet has been truncated for display purposes.*

> pkg/apis/workloads/v1beta1/zz_generated.kubebuilder.go

```go
// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: "workloads.k8s.io", Version: "v1beta1"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)
...
```
{% endmethod %}

