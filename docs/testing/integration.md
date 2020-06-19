**Writing and Running Integration Tests**

**This document is for kubebuilder v1 only**

This article explores steps to write and run integration tests for controllers created using Kubebuilder. Kubebuilder provides a template for writing integration tests. You can simply run all integration (and unit) tests within the project by running: `make test`

For example, there is a controller watching *Parent* objects. The *Parent* objects create *Child* objects. Note that the *Child* objects must have their `.ownerReferences` field setting to the `Parent` objects. You can find the template under `pkg/controllers/parent/parent_controller_test.go`:
```
package parent

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	childapis "k8s.io/child/pkg/apis"
	childv1alpha1 "k8s.io/childrepo/pkg/apis/child/v1alpha1"
	parentapis "k8s.io/parent/pkg/apis"
	parentv1alpha1 "k8s.io/parentrepo/pkg/apis/parent/v1alpha1"

	...<other import items>...
)

const timeout = time.Second * 5

var c client.Client
var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "parent", Namespace: "default"}}
var childKey = types.NamespacedName{Name: "child", Namespace: "default"}

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Parent instance to be created.
	parent := &parentv1alpha1.Parent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "parent",
			Namespace: "default",
		},
		Spec: metav1.ParentSpec{
			SomeSpecField:    "SomeSpecValue",
			AnotherSpecField: "AnotherSpecValue",
		},
	}

	// Setup the Manager and Controller. Wrap the Controller Reconcile function
	// so it writes each request to a channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})

	// Setup Scheme for all resources.
	if err = parentapis.AddToScheme(mgr.GetScheme()); err != nil {
		t.Logf("failed to add Parent scheme: %v", err)
	}
	if err = childapis.AddToScheme(mgr.GetScheme()); err != nil {
		t.Logf("failed to add Child scheme: %v", err)
	}

	// Set up and start test manager.
	reconciler, err := newReconciler(mgr)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	recFn, requests := SetupTestReconcile(reconciler)
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())
	defer close(StartTestManager(mgr, g))

	// Create the Parent object and expect the Reconcile and Child to be created.
	c = mgr.GetClient()
	err = c.Create(context.TODO(), parent)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), parent)
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	// Verify Child is created.
	child := &childv1alpha1.Child{}
	g.Eventually(func() error { return c.Get(context.TODO(), childKey, child) }, timeout).
		Should(gomega.Succeed())

	// Manually delete Child since GC isn't enabled in the test control plane.
	g.Expect(c.Delete(context.TODO(), child)).To(gomega.Succeed())
}
```

`SetupTestReconcile` function above brings up an API server and etcd instance. Note that there is no any node creation for integration testing environment. If you want to test your controller on a real node, you should write end-to-end tests.

The manager is started as part of the test itself (`StartTestManager` function).

Both functions are located in `pkg/controllers/parent/parent_controller_suite_test.go` file. The file also contains a `TestMain` function that allows you to specify CRD directory paths for the testing environment.
