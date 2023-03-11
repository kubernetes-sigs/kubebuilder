# Writing controller tests

Testing Kubernetes controllers is a big subject, and the boilerplate testing
files generated for you by kubebuilder are fairly minimal.

To walk you through integration testing patterns for Kubebuilder-generated controllers, we will revisit the CronJob we built in our first tutorial and write a simple test for it.

The basic approach is that, in your generated `suite_test.go` file, you will use envtest to create a local Kubernetes API server, instantiate and run your controllers, and then write additional `*_test.go` files to test it using [Ginkgo](http://onsi.github.io/ginkgo).

If you want to tinker with how your envtest cluster is configured, see section [Configuring envtest for integration tests](../reference/envtest.md) as well as the [`envtest docs`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest?tab=doc).

## Test Environment Setup

{{#literatego ../cronjob-tutorial/testdata/project/internal/controller/suite_test.go}}

## Testing your Controller's Behavior

{{#literatego ../cronjob-tutorial/testdata/project/internal/controller/cronjob_controller_test.go}}

This Status update example above demonstrates a general testing strategy for a custom Kind with downstream objects. By this point, you hopefully have learned the following methods for testing your controller behavior:

* Setting up your controller to run on an envtest cluster
* Writing stubs for creating test objects
* Isolating changes to an object to test specific controller behavior

## Advanced Examples

There are more involved examples of using envtest to rigorously test controller behavior. Examples include:

* Azure Databricks Operator: see their fully fleshed-out
  [`suite_test.go`](https://github.com/microsoft/azure-databricks-operator/blob/0f722a710fea06b86ecdccd9455336ca712bf775/controllers/suite_test.go)
  as well as any `*_test.go` file in that directory [like this
  one](https://github.com/microsoft/azure-databricks-operator/blob/0f722a710fea06b86ecdccd9455336ca712bf775/controllers/secretscope_controller_test.go).
