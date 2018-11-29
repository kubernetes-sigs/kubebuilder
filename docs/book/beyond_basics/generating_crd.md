# Generating CRD

Kubebuilder provides a tool named `controller-gen` to generate manifests for CustomResourceDefinitions. The tool resides in the [controller-tools](http://sigs.k8s.io/controller-tools) repository which is vendored in every project. If you examine the `Makefile` in the project, you will see a target named `manifests` as shown below. `manifests` target is also listed as prerequisite for other targets like `run`, `tests`, `deploy` etc to ensure CRD manifests are regenerated when needed.

```sh
# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all
```

When you run `make manifests`, you should see an output like below. CRDs are generated under `configs/crds` directory.
```sh
$ make manifests
go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all
CRD manifests generated under '<curr_dir>/config/crds'
RBAC manifests generated under '<curr_dir>/config/rbac'
```

Though `controller-gen` generates manifests for RBAC as well, but this section describes the generation of CRD manifest. 

`controller-gen` reads kubebuilder annotations of the form `// +kubebuilder:something...` defined as Go comments in the `<your-api-kind>_types.go` file under `pkg/apis/...` to produce the CRD manifests. Sections below describe various supported annotations.

## Validation

CRDs support [validation](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#validation) by definining ([OpenAPI v3 schema](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#schemaObject)) in the validation section. To learn more about the validation feature, refer to the original docs [here](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#validation). One can specify validation for a field by annotating the field with kubebuilder annotation which is of the form`// +kubebuilder:validation:<key=value>`. If you want to specify multiple validations for a field, you can add multiple such annotation as demonstrated in the example below.

Currently, supporting keys are `Maximum`, `Minimum`, `MaxLength`, `MinLength`, `MaxItems`, `MinItems`, `UniqueItems`, `Enum`, `Pattern`, `ExclusiveMaximum`,
 `ExclusiveMinimum`, `MultipleOf`, `Format`.  The `// +kubebuilder:validation:Pattern=.+:.+` annotation specifies the Pattern validation requiring that the `Image` field match the regular expression `.+:.+`

**Example:**

```go
type ToySpec struct {

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

	// +kubebuilder:validation:Enum=Lion,Wolf,Dragon
	Alias string `json:"alias,omitempty"`

	// +kubebuilder:validation:Enum=1,2,3
	Rank    int    `json:"rank"`
}

```

## Additional printer columns

Starting with Kubernetes 1.11, kubectl uses server-side printing. The server
decides which columns are shown by the kubectl get command. You can 
[customize these columns using a CustomResourceDefinition](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#additional-printer-columns).
To add an additional column, add a comment with the following annotation format
just above the struct definition of the Kind.

Format: `// +kubebuilder:printcolumn:name="Name",type="type",JSONPath="json-path",description="desc",priority="priority",format="format"`

Note that `description`, `priority` and `format` are optional. Refer to the
[additonal printer columns docs](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#additional-printer-columns)
to learn more about the values of `name`, `type`, `JsonPath`, `description`, `priority` and `format`.

The following example adds the `Spec`, `Replicas`, and `Age` columns.

```Go
// +kubebuilder:printcolumn:name="Spec",type="integer",JSONPath=".spec.cronSpec",description="status of the kind"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.Replicas",description="The number of jobs launched by the CronJob"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type CronTab struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronTabSpec   `json:"spec,omitempty"`
	Status CronTabStatus `json:"status,omitempty"`
}

```


## Subresource
Custom resources support `/status` and `/scale` subresources as of kubernetes
1.13 release. You can learn more about the subresources [here](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#status-subresource).

### 1. Status
To enable `/status` subresource, annotate the kind with `// +kubebuilder:subresource:status` format.

### 2. Scale
To enable `/scale` subresource, annotate the kind with `// +kubebuilder:subresource:scale:specpath=<jsonpath>,statuspath=<jsonpath>,selectorpath=<jsonpath>` format.

Scale subresource annotation contains three fields: `specpath`, `statuspath` and `selectorpath`.

- `specpath` refers to `specReplicasPath` attribute of Scale object, and value `jsonpath` defines the JSONPath inside of a custom resource that corresponds to `Scale.Spec.Replicas`. This is a required field.
- `statuspath` refers to `statusReplicasPath` attribute of Scale object. and the `jsonpath` value of it defines the JSONPath inside of a custom resource that corresponds to `Scale.Status.Replicas`. This is a required field.
- `selectorpath` refers to `labelSelectorPath` attribute of Scale object, and the value `jsonpath` defines the JSONPath inside of a custom resource that corresponds to `Scale.Status.Selector`. This is an optional field.


**Example:**

```go
type ToySpec struct {
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

In the above example for the type `Toy`, we added `// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas` comment before `Toy` struct definition. `.spec.replicas` refers to the josnpath of Spec struct field (`ToySpec.Replicas`). And jsonpath `.status.healthyReplicas` refers to Status struct field (`ToyStatus.Replicas`).
