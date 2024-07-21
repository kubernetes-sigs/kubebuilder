# CRD Scope


## Overview

CRDs come with a built-in scope field that plays a crucial role in determining the visibility and accessibility of the resulting Custom Resources (CRs). This field essentially dictates whether your CRs are cluster-wide or restricted to specific namespaces.

## Reasons for Choosing Different Scopes:

- **Namespace-scoped CRDs**: These are ideal when you want to limit access to CRs within certain namespaces. This is useful for scenarios like managing resources specific to a particular team or application. Additionally, you can have different versions of CRs available in different namespaces, allowing for gradual rollouts or experimentation.

- **Cluster-scoped CRDs**: If you need all namespaces to have access and interact with your CRs in a uniform manner, opt for a cluster-scoped CRD. This is beneficial for shared resources or central configuration management across the entire cluster.

## Setting the Scope

CRD manifests are usually generated using the `operator-sdk create api` command. These manifests reside in the `config/crd/bases` directory. Within a CRD's manifest, the `spec.scope` field controls its API scope. This field accepts two valid values:

- **Cluster**: This makes the CR accessible and manageable from all namespaces within the cluster.

- **Namespaced**: This restricts CR access and management to the specific namespace where the CR is created.

For projects employing the Operator SDK in Go, the `operator-sdk create api` command has a `--namespaced flag`. This flag determines the value of `spec.scope` and modifies the corresponding `types.go` file for the resource. In other operator types, the scope can be directly set by editing the `spec.scope` field in the CRD's YAML manifest file.


## Set create api –namespaced flag

When creating a new API, the `--namespaced` flag controls whether the resulting CRD will be cluster or namespace scoped. By default, `--namespaced` is set to true which sets the scope to Namespaced. An example command to create a cluster-scoped API would be:

```shell
$ operator-sdk create api --group cache --version v1alpha1 --kind Memcached --resource=true --controller=true --namespaced=false
```

## Set Scope Marker in types.go

You can also manually set the scope in the Go types.go file by adding or changing the kubebuilder scope marker to your resource. This file is usually located in `api/<version>/<kind>_types.go` or `apis/<group>/<version>/<kind>_types.go` if you are using the multigroup layout. Once this marker is set, the CRD files will be generated with the approriate scope. Here is an example API type with the marker set to cluster scope:

```go
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// Memcached is the Schema for the memcacheds API
type Memcached struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedSpec   `json:"spec,omitempty"`
	Status MemcachedStatus `json:"status,omitempty"`
}
```
To set the scope to namespaced, the marker would be set to `//+kubebuilder:resource:scope=Namespaced` instead.

## Set scope in CRD YAML file

The scope can be manually set directly in the CRD’s Kind YAML file, normally located in `config/crd/bases/<group>.<domain>_<kind>.yaml`. An example YAML file for a namespace-scoped CRD is shown below:

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.2.5
  creationTimestamp: null
  name: memcacheds.cache.example.com
spec:
  group: cache.example.com
  names:
    kind: Memcached
    listKind: MemcachedList
    plural: memcacheds
    singular: memcached
  scope: Namespaced
  subresources:
    status: {}
... 
```



