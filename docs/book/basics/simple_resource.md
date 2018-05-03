{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Simple Resource Example

This chapter walks through the definition of a new Resource call *ContainerSet*.  ContainerSet
contains the image and replicas fields, and ensures a Deployment with matching image and replicas
it running in the cluster.

Create the scaffolding for a new resource using the kubebuilder cli:

> $ kubebuilder create resource --group workloads --version v1beta1 --kind ContainerSet

This creates several files, including the Resource schema definition in:

> pkg/apis/workloads/v1beta1/containerset_types.go

{% method %}
## Type Definition

ContainerSet has 4 fields:

- Spec contains the desired cluster state as collectively defined by users, controllers and other sources.
  Examples of controllers that write to the Spec include Autoscalers which write the replicas field.
- Status contains only *observed cluster state* and is written only by controllers
  All information in Status may be derived from other sources.
- TypeMeta contains metadata about the API itself - such as Group, Version, Kind.
- ObjectMeta contains metadata about the specific object instance - such as the name, namespace,
  labels and annotations.  ObjectMeta contains data common to most objects.

#### Reference

- See the [resource code generation tags](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/apis)
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

Note: The `// +kubebuilder:validation:Pattern=.+:.+` annotation declares Pattern validation
requiring that the `Image` field match the regular expression `.+:.+`
{% endpanel %}

{% method %}
## ContainerSetSpec

The ContainerSetSpec contains the container image and replica count, which should be read by
the controller and used to create and manage a new Deployment.  The Spec field contains desired
state defined by either the user or set by controllers (such as autoscalers).

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
	HealthyReplicas `json:"healthyReplicas,omitempty"`
}
```
{% endmethod %}

{% panel style="info", title="When to rerun code generators" %}
While users don't directly modify generated code, users must rerun `kubebuilder generate`
after modifying resources or `// +something` annotations.
{% endpanel %}

{% method %}
## Generated Boilerplate

Kubebuilder generates boilerplate code necessary for using the apimachinery libraries and
implementing wiring.  For simple cases, users should not need to know much more than
that this code exists and is managed by kubebuilder.  This code snippet shows an
example of some of the generated code.

Code generation may be configured by comment annotations of the form `// +something`
on resource and controller structs.  See the [pkg/gen](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/)
reference documentation for more details.

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

