# Writing controller tests

Testing Kubernetes controller is a big subject, and the boilerplate testing
files generated for you by kubebuilder are fairly minimal.

[Writing and Running Integration Tests](/reference/testing/envtest.md) documents steps to consider when writing integration steps for your controllers, and available options for configuring your test control plane using [`envtest`](https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/envtest).

Until more documentation has been written, your best bet to get started is to look at some
existing examples, such as:

* Azure Databricks Operator: see their fully fleshed-out
  [`suite_test.go`](https://github.com/microsoft/azure-databricks-operator/blob/0f722a710fea06b86ecdccd9455336ca712bf775/controllers/suite_test.go)
  as well as any `*_test.go` file in that directory [like this
  one](https://github.com/microsoft/azure-databricks-operator/blob/0f722a710fea06b86ecdccd9455336ca712bf775/controllers/secretscope_controller_test.go).

The basic approach is that, in your generated `suite_test.go` file, you will
create a local Kubernetes API server, instantiate and run your controllers, and
then write additional `*_test.go` files to test it using
[Ginko](http://onsi.github.io/ginkgo).
