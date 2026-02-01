# CRD Scope

This document explains CustomResourceDefinition (CRD) scope in Kubernetes: how CRDs can be defined as namespace-scoped or cluster-scoped resources.

<aside class="note">
<h1>CRD Scope vs Manager Scope</h1>

CRD scope is independent from manager scope. See [Understanding Scopes](./scopes.md) for an explanation of how these two concepts differ.
</aside>

## Overview

CRD scope determines the visibility and availability of custom resources:

| Scope | Description | Example Resources |
|-------|-------------|-------------------|
| **Namespace-scoped** (default) | Resources exist within a specific namespace | Deployments, Services, ConfigMaps, Pods |
| **Cluster-scoped** | Resources are global across the entire cluster | Nodes, ClusterRoles, Namespaces, PersistentVolumes |

## Namespace-Scoped CRDs (Default)

By default, Kubebuilder creates namespace-scoped CRDs:

```bash
kubebuilder create api --group cache --version v1alpha1 --kind Memcached
```

Generated CRD manifest:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: memcacheds.cache.example.com
spec:
  scope: Namespaced  # Default
  group: cache.example.com
  names:
    kind: Memcached
    plural: memcacheds
  versions:
  - name: v1alpha1
    # ...
```

Custom resources are created in specific namespaces:

```bash
kubectl apply -f memcached.yaml -n my-namespace
kubectl get memcacheds -n my-namespace
```

**When to use:**
- Resources tied to specific applications, teams, or tenants
- Multi-tenant environments where isolation is required
- Most application-level resources

**Considerations:**
- Testing new CRD versions requires proper versioning and conversion strategies
- Conversion webhooks must account for namespace scope
- Facilitates controlled rollout within specific namespaces

## Cluster-Scoped CRDs

Cluster-scoped CRDs create resources that are global across the entire cluster.

### Creating Cluster-Scoped CRDs

When creating the API, use the `--namespaced=false` flag:

```bash
kubebuilder create api --group infrastructure --version v1 --kind Database --namespaced=false
```

Generated CRD manifest:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: databases.infrastructure.example.com
spec:
  scope: Cluster  # Cluster-scoped
  group: infrastructure.example.com
  names:
    kind: Database
    plural: databases
  versions:
  - name: v1
    # ...
```

Custom resources are cluster-wide (no namespace):

```bash
kubectl apply -f database.yaml
kubectl get databases  # No namespace needed
```

**When to use:**
- Resources that are global to the cluster (infrastructure, configuration)
- Resources that need to be accessible from all namespaces
- Resources that manage cluster-level concerns

**Examples:**
- Infrastructure configurations (cloud provider settings, cluster DNS)
- Global policies or quotas
- Cross-namespace resource aggregation

## Changing CRD Scope

### For Existing APIs

After creating an API, you can change its scope using the `+kubebuilder:resource:scope` marker:

**For cluster-scoped:**

```go
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// Database is the Schema for the databases API
type Database struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   DatabaseSpec   `json:"spec,omitempty"`
    Status DatabaseStatus `json:"status,omitempty"`
}
```

**For namespace-scoped:**

```go
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced

// Memcached is the Schema for the memcacheds API
type Memcached struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   MemcachedSpec   `json:"spec,omitempty"`
    Status MemcachedStatus `json:"status,omitempty"`
}
```

After updating markers, regenerate manifests:

```bash
make manifests
```

<aside class="warning">
<h1>Scope Changes Are Breaking</h1>

Changing CRD scope from Namespaced to Cluster (or vice versa) is a **breaking change**:
- Existing custom resources will become invalid
- Users must migrate their resources manually
- Consider creating a new CRD with a different version instead

Only change scope during initial development before any production usage.
</aside>

## RBAC for CRD Scope

### Namespace-Scoped CRDs

Controllers watching namespace-scoped CRDs use namespace-scoped RBAC:

```go
//+kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update
```

Generated RBAC (cluster-scoped manager):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
- apiGroups: ["cache.example.com"]
  resources: ["memcacheds"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

Generated RBAC (namespace-scoped manager):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: manager-role
  namespace: manager-namespace
rules:
- apiGroups: ["cache.example.com"]
  resources: ["memcacheds"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

### Cluster-Scoped CRDs

Controllers watching cluster-scoped CRDs **must** use cluster-wide RBAC:

```go
//+kubebuilder:rbac:groups=infrastructure.example.com,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.example.com,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.example.com,resources=databases/finalizers,verbs=update
```

Generated RBAC (always ClusterRole for cluster-scoped CRDs):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
- apiGroups: ["infrastructure.example.com"]
  resources: ["databases"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

<aside class="note">
<h1>Important</h1>

Even if your manager is namespace-scoped (watches only one namespace), if it manages cluster-scoped CRDs, it still needs `ClusterRole` permissions for those resources.

Manager scope and CRD scope are independent:
- **Manager scope**: Controlled by cache configuration (which namespaces to watch)
- **CRD scope**: Controlled by the CRD's `scope` field (resource visibility)
</aside>

## Version Conversion and Webhooks

For namespace-scoped CRDs with multiple versions, conversion webhooks must account for namespace scope:

```go
//+kubebuilder:webhook:path=/convert,mutating=false,failurePolicy=fail,groups=cache.example.com,resources=memcacheds,verbs=create;update,versions=v1;v1beta1,name=cmemcached.kb.io,sideEffects=None,admissionReviewVersions=v1
```

The webhook must handle conversion for resources in any namespace. See the [multi-version tutorial](https://book.kubebuilder.io/multiversion-tutorial/tutorial) for details.

## Testing

### Testing Namespace-Scoped CRDs

```bash
# Create resource in namespace
kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml -n test-namespace

# Verify it exists in that namespace only
kubectl get memcacheds -n test-namespace
kubectl get memcacheds -n other-namespace  # Should not find it
```

### Testing Cluster-Scoped CRDs

```bash
# Create cluster-scoped resource (no namespace)
kubectl apply -f config/samples/infrastructure_v1_database.yaml

# Verify it's cluster-wide
kubectl get databases  # No namespace needed
```

## See Also

- [Manager Scope](./manager-scope.md) - Configuring manager watching scope
- [Generating CRDs](./generating-crd.md) - CRD generation and markers
- [Multi-Version Tutorial](https://book.kubebuilder.io/multiversion-tutorial/tutorial) - CRD versioning and conversion
- [Kubernetes CRD Documentation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/)
