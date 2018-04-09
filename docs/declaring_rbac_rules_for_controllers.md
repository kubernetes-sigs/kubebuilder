# Declaring RBAC rules for controllers

This document describes how to declare rbac rules for your controllers
in the code.

Admin RBAC rules are automatically generated for all APIs implemented
within the framework.  These rules are installed by the generated
installer along with a Namespace and ServiceAccount for the controller.

> Generated RBAC rules live in the `pkg/apis/zz_generated.kubebuilder.go` file
  under `func (MetaData) GetRules() []rbacv1.PolicyRule`

Since your controller will likely be interacting with additional resources
(e.g. core resources), it is possible to declare additional RBAC rules
for the controller ServiceAccount to be installed.

To define additional rbac rules, add a `//kubebuilder:+rbac` comment to the controller struct
under `pkg/controller/<name>/controller.go`

```go
// +kubebuilder:rbac:groups=apps;extensions,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch;create;update;delete
// +kubebuilder:controller:group=foo,version=v1alpha1,kind=Bar,resource=bars
type BarControllerImpl struct {
    builders.DefaultControllerFns

    // lister indexes properties about Bar
    lister listers.BarLister
}
```
