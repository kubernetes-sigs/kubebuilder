# Markers for Config/Code Generation

KubeBuilder makes use of a tool called
[controller-gen](/reference/controller-gen.md) for
generating utility code and Kubernetes YAML.  This code and config
generation is controlled by the presence of special "marker comments" in
Go code.

Markers are single-line comments that start with a plus, followed by
a marker name, optionally followed by some marker specific configuration:

```go
// +kubebuilder:validation:Optional
// +kubebuilder:validation:MaxItems=2
// +kubebuilder:printcolumn:JSONPath=".status.replicas",name=Replicas,type=string
```

See each subsection for information about different types of code and YAML
generation.

## Generating Code & Artifacts in KubeBuilder

KubeBuilder projects have two `make` targets that make use of
controller-gen:

- `make manifests` generates Kubernetes object YAML, like
  [CustomResourceDefinitions](./markers/crd.md),
  [WebhookConfigurations](./markers/webhook.md), and [RBAC
  roles](./markers/rbac.md).

- `make generate` generates code, like [runtime.Object/DeepCopy
  implementations](./markers/object.md).

See [Generating CRDs](./generating-crd.md) for a comprehensive overview.

## Marker Syntax

Exact syntax is described in the [godocs for
controller-tools](https://godoc.org/sigs.k8s.io/controller-tools/pkg/markers).

In general, markers may either be: 

- **Empty** (`+kubebuilder:validation:Optional`): empty markers are like boolean flags on the command line
  -- just specifying them enables some behavior.

- **Anonymous** (`+kubebuilder:validation:MaxItems=2`): anonymous markers take
  a single value as their argument.

- **Multi-option**
  (`+kubebuilder:printcolumn:JSONPath=".status.replicas",name=Replicas,type=string`): multi-option
  markers take one or more named arguments.  The first argument is
  separated from the name by a colon, and latter arguments are
  comma-separated.  Order of arguments doesn't matter.  Some arguments may
  be optional.

Marker arguments may be strings, ints, bools, slices, or maps thereof.
Strings, ints, and bools follow their Go syntax:

```go
// +kubebuilder:validation:ExclusiveMaximum=false
// +kubebuilder:validation:Format="date-time"
// +kubebuilder:validation:Maximum=42
```

For convenience, in simple cases the quotes may be omitted from strings,
although this is not encouraged for anything other than single-word
strings:

```go
// +kubebuilder:validation:Type=string
```

Slices may be specified either by surrounding them with curly braces and
separating with commas:

```go
// +kubebuilder:webhooks:Enum={"crackers, Gromit, we forgot the crackers!","not even wensleydale?"}
```

or, in simple cases, by separating with semicolons:

```go
// +kubebuilder:validation:Enum=Wallace;Gromit;Chicken
```

Maps are specified with string keys and values of any type (effectively
`map[string]interface{}`). A map is surrounded by curly braces (`{}`),
each key and value is separated by a colon (`:`), and each key-value
pair is separated by a comma:

```go
// +kubebuilder:validation:Default={magic: {numero: 42, stringified: forty-two}}
```
