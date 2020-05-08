# Kubebuilder v2 vs v3

<h1>Work In Progress: V3 Work Incoming</h1>

This document cover all breaking changes when migrating from v2 to v3.

The details of all changes (breaking or otherwise) can be found in
[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime/releases),
[controller-tools](https://github.com/kubernetes-sigs/controller-tools/releases)
and [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder/releases)
release notes.

## Kubebuilder

- Kubebuilder v3 introduces the plugins architecture. You can find the design
doc [Extensible CLI and Scaffolding Plugins](https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md).

## Migration Steps

### Fix controller alias typo

The alias for the `controllers` generate was in the singular and was update to plural to fix the typo issue. So, 
in the `main.go` update the controller alias for the plural and the places where it has been used. Example:

Replace:

```go
crewcontroller "sigs.k8s.io/kubebuilder/testdata/project-v3-multigroup/controllers/crew"
...
if err = (&crewcontroller.CaptainReconciler{
```

With: 

```go
crewcontrollers "sigs.k8s.io/kubebuilder/testdata/project-v3-multigroup/controllers/crew"
...
if err = (&crewcontrollers.CaptainReconciler{
```
