# Generating CRDs

KubeBuilder uses a tool called [`controller-gen`][controller-tools] to
generate utility code and Kubernetes object YAML, like
CustomResourceDefinitions.

To do this, it makes use of special "marker comments" (comments that start
with `// +`) to indicate additional information about fields, types, and
packages.  In the case of CRDs, these are generally pulled from your
`_types.go` files.  For more information on markers, see the [marker
reference docs][marker-ref].

KubeBuilder provides a `make` target to run controller-gen and generate
CRDs: `make manifests`.

When you run `make manifests`, you should see CRDs generated under the
`config/crd/bases` directory.  `make manifests` can generate a number of
other artifacts as well -- see the [marker reference docs][marker-ref] for
more details.

## Validation

CRDs support [declarative validation][kube-validation] using an [OpenAPI
v3 schema][openapi-schema] in the `validation` section.

In general, [validation markers](./markers/crd-validation.md) may be
attached to fields or to types. If you're defining complex validation, if
you need to re-use validation, or if you need to validate slice elements,
it's often best to define a new type to describe your validation.

For example:

```go
type ToySpec struct {
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:MaxItems=500
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=true
	Knights []string `json:"knights,omitempty"`

	Alias   Alias   `json:"alias,omitempty"`
	Rank    Rank    `json:"rank"`
}

// +kubebuilder:validation:Enum=Lion;Wolf;Dragon
type Alias string

// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=3
// +kubebuilder:validation:ExclusiveMaximum=false
type Rank int32

```

## Additional Printer Columns

Starting with Kubernetes 1.11, `kubectl get` can ask the server what
columns to display.  For CRDs, this can be used to provide useful,
type-specific information with `kubectl get`, similar to the information
provided for built-in types.

The information that gets displayed can be controlled with the
[additionalPrinterColumns field][kube-additional-printer-columns] on your
CRD, which is controlled by the
[`+kubebuilder:printcolumn`][crd-markers] marker on the Go type for
your CRD.

For instance, in the following example, we add fields to display
information about the knights, rank, and alias fields from the validation
example:

```go
// +kubebuilder:printcolumn:name="Alias",type=string,JSONPath=`.spec.alias`
// +kubebuilder:printcolumn:name="Rank",type=integer,JSONPath=`.spec.rank`
// +kubebuilder:printcolumn:name="Bravely Run Away",type=boolean,JSONPath=`.spec.knights[?(@ == "Sir Robin")]`,description="when danger rears its ugly head, he bravely turned his tail and fled",priority=10
type Toy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToySpec   `json:"spec,omitempty"`
	Status ToyStatus `json:"status,omitempty"`
}

```

## Subresources

CRDs can choose to implement the `/status` and `/scale`
[subresources][kube-subresources] as of Kubernetes 1.13.

It's generally reccomended that you make use of the `/status` subresource
on all resources that have a status field.

Both subresources have a corresponding [marker][crd-markers].

### Status

The status subresource is enabled via `+kubebuilder:subresource:status`.
When enabled, updates at the main resource will not change status.
Similarly, updates to the status subresource cannot change anything but
the status field.

For example:

```go
// +kubebuilder:subresource:status
type Toy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToySpec   `json:"spec,omitempty"`
	Status ToyStatus `json:"status,omitempty"`
}
```

### Scale

The scale subresource is enabled via `+kubebuilder:subresource:scale`.
When enabled, users will be able to use `kubectl scale` with your
resource.  If the `selectorpath` argument pointed to the string form of
a label selector, the HorizontalPodAutoscaler will be able to autoscale
your resource.

For example:

```go
type CustomSetSpec struct {
	Replicas *int32 `json:"replicas"`
}

type CustomSetStatus struct {
	Replicas int32 `json:"replicas"`
    Selector string `json:"selector"` // this must be the string form of the selector
}


// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
type CustomSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToySpec   `json:"spec,omitempty"`
	Status ToyStatus `json:"status,omitempty"`
}
```

## Multiple Versions

As of Kubernetes 1.13, you can have multiple versions of your Kind defined
in your CRD, and use a webhook to convert between them.

For more details on this process, see the [multiversion
tutorial](/multiversion-tutorial/tutorial.md).

By default, KubeBuilder disables generating different validation for
different versions of the Kind in your CRD, to be compatible with older
Kubernetes versions.

You'll need to enable this by switching the line in your makefile that
says `CRD_OPTIONS ?= "crd:trivialVersions=true` to `CRD_OPTIONS ?= crd`

Then, you can use the `+kubebuilder:storageversion` [marker][crd-markers]
to indicate the [GVK](/cronjob-tutorial/gvks.md "Group-Version-Kind") that
should be used to store data by the API server.

## Under the hood

KubeBuilder scaffolds out make rules to run `controller-gen`.  The rules
will automatically install controller-gen if it's not on your path using
`go get` with Go modules.

You can also run `controller-gen` directly, if you want to see what it's
doing.

Each controller-gen "generator" is controlled by an option to
controller-gen, using the same syntax as markers.  For instance, to
generate CRDs with "trivial versions" (no version conversion webhooks), we
call `controller-gen crd:trivialVersions=true paths=./api/...`.

controller-gen also supports different output "rules" to control how
and where output goes.  Notice the `manifests` make rule (condensed
slightly to only generate CRDs):

```makefile
# Generate manifests for CRDs
manifests: controller-gen
	$(CONTROLLER_GEN) crd:trivialVersions=true paths="./..." output:crd:artifacts:config=config/crd/bases
```

It uses the `output:crd:artifacts` output rule to indicate that
CRD-related config (non-code) artifacts should end up in
`config/crd/bases` instead of `config/crd`.

To see all the options for `controller-gen`, run

```shell
$ controller-gen -h
```

or, for more details:

```shell
$ controller-gen -hhh
```

[marker-ref]: ./markers.md "Markers for Config/Code Generation"

[kube-validation]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#validation "Custom Resource Definitions: Validation"

[openapi-schema]: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#schemaObject "OpenAPI v3"

[kube-additional-printer-colums]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#additional-printer-columns "Custom Resource Definitions: Additional Printer Columns"

[kube-subresources]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#status-subresource "Custom Resource Definitions: Status Subresource"

[crd-markers]: ./markers/crd.md "CRD Generation"

[controller-tools]: https://sigs.k8s.io/controller-tools "Controller Tools"
