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

<aside class="note">
<h1>Examples</h1>

You can use the plugin [DeployImage](../plugins/available/deploy-image-plugin-v1-alpha.md) to check examples. This plugin allows users to scaffold API/Controllers to deploy and manage an Operand (image) on the cluster following the guidelines and best practices. It abstracts the complexities of achieving this goal while allowing users to customize the generated code.

Therefore, you can check that a test using ENV TEST will be generated for the controller which has the purpose to ensure that the Deployment is created successfully. You can see an example of its code implementation under the `testdata` directory with the [DeployImage](../plugins/available/deploy-image-plugin-v1-alpha.md) samples [here](https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/project-v4-with-plugins).

</aside>
