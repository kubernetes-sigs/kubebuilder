/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cronjob

const suiteTestIntro = `
// +kubebuilder:docs-gen:collapse=Apache License

/*
When we created the CronJob API with` + " `" + `kubebuilder create api` + "`" + ` in a [previous chapter](/cronjob-tutorial/new-api.md), Kubebuilder already did some test work for you.
Kubebuilder scaffolded a` + " `" + `internal/controller/suite_test.go` + "`" + ` file that does the bare bones of setting up a test environment.

First, it will contain the necessary imports.
*/
`

const suiteTestEnv = `
// +kubebuilder:docs-gen:collapse=Imports

/*
Now, let's go through the code generated.
*/

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client // You'll be using this client in your tests.
)
`

const suiteTestAddSchema = `
	/*
		The CronJob Kind is added to the runtime scheme used by the test environment.
		This ensures that the CronJob API is registered with the scheme, allowing the test controller to recognize and interact with CronJob resources.
	*/
	err = batchv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	/*
		After the schemas, you will see the following marker.
		This marker is what allows new schemas to be added here automatically when a new API is added to the project.
	*/

	// +kubebuilder:scaffold:scheme

	/*
		The envtest environment is configured to load Custom Resource Definitions (CRDs) from the specified directory.
		This setup enables the test environment to recognize and interact with the custom resources defined by these CRDs.
	*/`

const suiteTestClient = `
	/*
		A client is created for our test CRUD operations.
	*/`

const suiteTestDescription = `
	/*
		One thing that this autogenerated file is missing, however, is a way to start your controller.
		The code above will set up a client for interacting with your custom Kind,
		but will not be able to test your controller behavior.
		If you want to test your custom controller logic, you’ll need to add some familiar-looking manager logic
		to your BeforeSuite() function, so you can register your custom controller to run on this test cluster.

		You may notice that the code below runs your controller with nearly identical logic to your CronJob project’s main.go!
		The only difference is that the manager is started in a separate goroutine so it does not block the cleanup of envtest
		when you’re done running your tests.

		Note that we set up both a "live" k8s client and a separate client from the manager. This is because when making
		assertions in tests, you generally want to assert against the live state of the API server. If you use the client
		from the manager (` + "`" + `k8sManager.GetClient` + "`" + `), you'd end up asserting against the contents of the cache instead, which is
		slower and can introduce flakiness into your tests. We could use the manager's ` + "`" + `APIReader` + "`" + ` to accomplish the same
		thing, but that would leave us with two clients in our test assertions and setup (one for reading, one for writing),
		and it'd be easy to make mistakes.

		Note that we keep the reconciler running against the manager's cache client, though -- we want our controller to
		behave as it would in production, and we use features of the cache (like indices) in our controller that aren't
		available when talking directly to the API server.
	*/
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&CronJobReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
`

const suiteTestCleanup = `
/*
Kubebuilder also generates boilerplate functions for cleaning up envtest and running your test files in your controllers/ directory.
You won't need to touch these.
*/

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

/*
Now that you have your controller running on a test cluster and a client ready to perform operations on your CronJob, we can start writing integration tests!
*/
`
