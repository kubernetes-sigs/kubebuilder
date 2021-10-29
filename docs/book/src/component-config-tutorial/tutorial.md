# Tutorial: ComponentConfig

Nearly every project that is built for Kubernetes will eventually need to
support passing in additional configurations into the controller. These could
be to enable better logging, turn on/off specific feature gates, set the sync
period, or a myriad of other controls. Previously this was commonly done using
cli `flags` that your `main.go` would parse to make them accessible within your
program. While this _works_ it's not a future forward design and the Kubernetes
community has been migrating the core components away from this and toward
using versioned config files, referred to as "component configs".

The rest of this tutorial will show you how to configure your kubebuilder
project with the a component config type then moves on to implementing a custom
type so that you can extend this capability.


<aside class="note">

<h1>Following Along vs Jumping Ahead</h1>

Note that most of this tutorial is generated from literate Go files that
form a runnable project, and live in the book source directory:
[docs/book/src/componentconfig-tutorial/testdata/project][tutorial-source].

[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/component-config-tutorial/testdata/project

</aside>

## Resources

* [Versioned Component Configuration File Design](https://docs.google.com/document/d/1FdaEJUEh091qf5B98HM6_8MS764iXrxxigNIdwHYW9c/)

* [Config v1alpha1 Go Docs](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/config/v1alpha1/)
