# Annotation

The definition contains several comment annotations of the form `// +something`.  These are used to configure code generators to run against this code.  The code generators will generate boilerplate functions and types to complete the Resource definition.
To learn more on configuring code generation see the *Code Generation* chapter.

## Validation

Format: `// +kubebuilder:validation:<key=value>`
The validation annotation supports CRD validation ([OpenAPI v3 schema](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#schemaObject)).
The last `key-value` part should be a sigle key-value pair. If you have multiple validation key-value pairs, should put them into separte annotation comments.
Currently, supporting keys are `Maximum`, `Minimum`, `MaxLength`, `MinLength`, `MaxItems`, `MinItems`, `UniqueItems`, `Enum`, `Pattern`, `ExclusiveMaximum`,
 `ExclusiveMinimum`, `MultipleOf`, `Format`
The `// +kubebuilder:validation:Pattern=.+:.+` annotation declares Pattern validation requiring that the `Image` field match the regular expression `.+:.+`

**Example:**

```go
// ToySpec defines the desired state of Toy
type ToySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:ExclusiveMinimum=true
	Power  float32 `json:"power,omitempty"`
	Bricks int32   `json:"bricks,omitempty"`
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:MaxItems=500
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=false
	Knights []string `json:"knights,omitempty"`
	Winner  bool     `json:"winner,omitempty"`
	// +kubebuilder:validation:Enum=Lion,Wolf,Dragon
	Alias string `json:"alias,omitempty"`
	// +kubebuilder:validation:Enum=1,2,3
	Rank    int    `json:"rank"`
	Comment []byte `json:"comment,omitempty"`

	Template v1.PodTemplateSpec       `json:"template"`
    Claim    v1.PersistentVolumeClaim `json:"claim,omitempty"`
    
    Replicas *int32 `json:"replicas"`
}

```


## Subresource


### 1. Status

Format: `// +kubebuilder:subresource:status`,


### 2. Scale
   
Format: `// +kubebuilder:subresource:scale:specpath=<jsonpath>,statuspath=<jsonpath>,selectorpath=<jsonpath>`


Scale subresource annotation contains three fields: `specpath`, `statuspath` and `selectorpath`.
1) `specpath` refers to `specReplicasPath` attribute of Scale object, and value `jsonpath` defines the JSONPath inside of a custom resource that corresponds to `Scale.Spec.Replicas`. This filed is required and not empty
2) `statuspath` refers to `statusReplicasPath` attribute of Scale object. and the `jsonpath` value of it defines the JSONPath inside of a custom resource that corresponds to `Scale.Status.Replicas`. This filed is required and not empty
3) `selectorpath`refers to `labelSelectorPath` attribute of Scale object, and the value `jsonpath` defines the JSONPath inside of a custom resource that corresponds to Scale.Status.Selector. This filed is optional, which can be omitted or with empty value


**Example:**

```go

// ToySpec defines the desired state of Toy
type ToySpec struct {

    // Other Fields of ToySpec

	Replicas *int32 `json:"replicas"` // Add this field in Toy Spec, so the jsonpath to this field is `.spec.replicas`
}


// ToyStatus defines the observed state of Toy
type ToyStatus struct {

	Replicas int32 `json:"replicas"` // Add this field in Toy Status, so the jsonpath to this field is `.status.replicas`
}


// Toy is the Schema for the toys API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
type Toy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToySpec   `json:"spec,omitempty"`
	Status ToyStatus `json:"status,omitempty"`
}

```

In order to enable scale subresource in type definition file, you have to apply the scale subresource right before the kind struct definition, with correct jsonpath values according to the spec and status. And then make sure the jsonpaths are already defined in the Spec and Status struct. Finally, update the `<kind>_types_test.go` files according to the types Spec and Status changes.
In the example of `Toy`, we can add `// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas` right a comment line before `Toy` struct definition. `.spec.replicas` referes the josnpath to Spec struct field (`ToySpec.Replicas`). And jsonpath `.status.healthyReplicas` refers to Status struct field (`ToyStatus.Replicas`). We don't have specific lableSelector deifined, so skip the `selectorpath` parts.