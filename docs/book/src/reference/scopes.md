# Understanding and Setting Scopes for Managers (Operators) and CRDs

This section covers the configuration of the operational and resource scopes
within a Kubebuilder project. Managers("Operators") in Kubernetes can be scoped to either
specific namespaces or the entire cluster, influencing how resources are watched and managed.

Additionally, CustomResourceDefinitions (CRDs) can be defined to be either
namespace-scoped or cluster-scoped, affecting their availability
across the cluster.

## Configuring Manager Scope

Managers can operate under different scopes depending on
the resources they need to handle:

### (Default) Watching All Namespaces

By default, if no namespace is specified, the manager will observe all namespaces.
This is configured as follows:

```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
...
})
```

### Watching a Single Namespace

To constrain the manager to monitor resources within a specific namespace, set the Namespace option:

```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
...
   Cache: cache.Options{
      DefaultNamespaces: map[string]cache.Config{"operator-namespace": cache.Config{}},
   },
})
```

### Watching Multiple Namespaces

A manager can also be configured to watch a specified set of namespaces using [Cache Config][CacheConfig]:

```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
...
Cache: cache.Options{
    DefaultNamespaces: map[string]cache.Config{
        "operator-namespace1": cache.Config{},
        "operator-namespace2": cache.Config{},
        },
    },
})
```

## Configuring CRD Scope

The scope of CRDs determines their visibility either within specific namespaces or across the entire cluster.

### Namespace-scoped CRDs

Namespace-scoped CRDs are suitable when resources need to be isolated to specific namespaces.
This setting helps manage resources related to particular teams or applications.
However, it is important to note that due to the unique definition of CRDs (Custom Resource Definitions) in Kubernetes, testing a new version of a CRD is not straightforward. Proper versioning and conversion strategies need to be implemented (example in our [kubebuilder tutorial][kubebuilder-multiversion-tutorial]), and coordination is required to manage which manager instance handles the conversion (see the official [kubernetes documentation][k8s-crd-conversion] about this).
Additionally, the namespace scope must be taken into account for mutating and validating webhook configurations to ensure they are correctly applied within the intended scope. This facilitates a more controlled and phased rollout strategy.

### Cluster-scoped CRDs

For resources that need to be accessible and manageable across the entire cluster,
such as shared configurations or global resources, cluster-scoped CRDs are used.

#### Configuring CRDs Scopes

**When the API is created**

The scope of a CRD is defined when generating its manifest.
Kubebuilder facilitates this through its API creation command.

By default, APIs are created with CRD scope as namespaced. However,
for cluster-wide you use `--namespaced=false`, i.e.:

```shell
kubebuilder create api --group cache --version v1alpha1 --kind Memcached --resource=true --controller=true --namespaced=false
```

This command generates the CRD with the Cluster scope,
meaning it will be accessible and manageable across all
namespaces in the cluster.

**By updating existing APIs**

After you create an API you are still able to change the scope.
For example, to configure a CRD to be cluster-wide,
add the `+kubebuilder:resource:scope=Cluster` marker
above the API type definition in your Go file.
Here is an example:

```go
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=mc

...
```

After setting the desired scope with markers,
run `make manifests` to generate the files.
This command invokes [`controller-gen`][controller-tools] to generate the CRD manifests
according to the markers specified in your Go files.

The generated manifests will then correctly reflect
the scope as either Cluster or Namespaced without
needing manual adjustment in the YAML files.

[controller-tools]: https://sigs.k8s.io/controller-tools
[CacheConfig]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/cache#Config
[kubebuilder-multiversion-tutorial]: https://book.kubebuilder.io/multiversion-tutorial/tutorial
[k8s-crd-conversion]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion