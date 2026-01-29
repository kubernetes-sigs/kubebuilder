# Understanding Scopes in Kubebuilder

In Kubernetes, **scope** defines the boundaries within which a resource or controller operates.

When building with Kubebuilder, you work with two independent scoping concepts:

1. **[Manager Scope](./manager-scope.md)** - Determines which namespace(s) your manager watches and operates in
2. **[CRD Scope](./crd-scope.md)** - Determines whether your custom resources are namespace-specific or cluster-wide

## What is Scope?

Scope defines the visibility and access boundaries in a Kubernetes cluster:

- **Cluster-scoped**: Operates across the entire cluster with access to all namespaces
- **Namespace-scoped**: Limited to specific namespace(s) for isolation and security

## Manager Scope vs CRD Scope

These concepts are **independent** and configured separately:

- **Manager Scope**: Controls which namespace(s) the manager watches (configured via deployment RBAC and cache)
- **CRD Scope**: Controls whether custom resources are namespace-specific or cluster-wide (configured in CRD manifest)

You can combine them in different ways - for example, a cluster-scoped manager can manage namespace-scoped CRDs (the default pattern).

## Learn More

For detailed information, configuration steps, and code examples:

- **[Manager Scope](./manager-scope.md)** - Manager scope configuration, RBAC, cache setup, and namespace watching
- **[CRD Scope](./crd-scope.md)** - CRD scope configuration, markers, and RBAC considerations
- **[Migrating to Namespace-Scoped Manager](../migration/namespace-scoped.md)** - Step-by-step migration guide for existing projects
