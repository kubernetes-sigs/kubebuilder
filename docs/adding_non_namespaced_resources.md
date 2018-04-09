# Adding non-namespaced resources

This document covers how to create a non-namespaced resource using
`kubebuilder`.

## Prerequisites

- [adding resources](adding_resources.md)

## Creating a non-namespaced resource with kubebuilder

Use the `--non-namespaced=true` flag when creating a resource:

`kubebuilder create resource --non-namespaced=true --group <group> --version <version> --kind <Kind>`

## Non-namespaced resources

Non-namespaced resources have the following differences from namespaced resources:

- `nonNamespaced` comment directive above the type
  - `// +nonNamespaced=true` comment under `// +genclient=true`
- The generated CRD will have `Scope: "Cluster"` instead of
  `Scope: "Namespaced"`
- Do not provide namespace when creating the client from a clientset

Example:

```go
// +genclient=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +nonNamespaced=true

// +kubebuilder:resource:path=foos
// +k8s:openapi-gen=true
// Foo defines some thing
type Foo struct {
...
}

...
```
