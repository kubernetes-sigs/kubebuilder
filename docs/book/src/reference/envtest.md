# Configuring envtest for integration tests

The [`controller-runtime/pkg/envtest`][envtest] Go library helps write integration tests for your controllers by setting up and starting an instance of etcd and the Kubernetes API server, without kubelet, controller-manager or other components.

## Installation

The `test` make target, also called by the `docker-build` target,
[downloads][setup-envtest] a set of envtest binaries (described above) to run tests with.
Typically nothing needs to be done on your part,
as the download and install script is fully automated,
although it does require `bash` to run.

If you would like to download the tarball containing these binaries,
to use in a disconnected environment for example,
run the following (Kubernetes version 1.19.2 is an example version):

```sh
K8S_VERSION=1.19.2
curl -sSLo envtest-bins.tar.gz "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-${K8S_VERSION}-$(go env GOOS)-$(go env GOARCH).tar.gz"
```

Then install them:

```sh
mkdir /opt/kubebuilder/testbin
tar -C /opt/kubebuilder/testbin --strip-components=1 -zvxf envtest-bins.tar.gz
```

Once these binaries are installed, you can either change the `test` target to:

```makefile
test: manifests generate fmt vet
	go test ./... -coverprofile cover.out
```

Or configure the existing target to skip the download and point to a [custom location](#environment-variables):

```sh
make test SKIP_FETCH_TOOLS=1 KUBEBUILDER_ASSETS=/opt/kubebuilder/testbin
```

## Writing tests

Using `envtest` in integration tests follows the general flow of:

```go
import sigs.k8s.io/controller-runtime/pkg/envtest

//specify testEnv configuration
testEnv = &envtest.Environment{
	CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
}

//start testEnv
cfg, err = testEnv.Start()

//write test logic

//stop testEnv
err = testEnv.Stop()
```

`kubebuilder` does the boilerplate setup and teardown of testEnv for you, in the ginkgo test suite that it generates under the `/controllers` directory.

Logs from the test runs are prefixed with `test-env`.

### Configuring your test control plane

Controller-runtime’s [envtest](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/envtest) framework requires `kubectl`, `kube-apiserver`, and `etcd` binaries be present locally to simulate the API portions of a real cluster. 

For projects built with plugin v3+ (see your PROJECT file's `layout` key), the `make test` command will install these binaries to the `testbin/` directory and use them when running tests that use `envtest`.  

You can use environment variables and/or flags to specify the `kubectl`,`api-server` and `etcd` setup within your integration tests.

### Environment Variables

| Variable name | Type | When to use |
| --- | :--- | :--- |
| `USE_EXISTING_CLUSTER` | boolean | Instead of setting up a local control plane, point to the control plane of an existing cluster. |
| `KUBEBUILDER_ASSETS` | path to directory | Point integration tests to a directory containing all binaries (api-server, etcd and kubectl). |
| `TEST_ASSET_KUBE_APISERVER`, `TEST_ASSET_ETCD`, `TEST_ASSET_KUBECTL` | paths to, respectively, api-server, etcd and kubectl binaries | Similar to `KUBEBUILDER_ASSETS`, but more granular. Point integration tests to use binaries other than the default ones. These environment variables can also be used to ensure specific tests run with expected versions of these binaries. |
| `KUBEBUILDER_CONTROLPLANE_START_TIMEOUT` and `KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT` | durations in format supported by [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration) | Specify timeouts different from the default for the test control plane to (respectively) start and stop; any test run that exceeds them will fail. |
| `KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT` | boolean | Set to `true` to attach the control plane's stdout and stderr to os.Stdout and os.Stderr. This can be useful when debugging test failures, as output will include output from the control plane. |

See that the `test` makefile target will ensure that all is properly setup when you are using it. However, if you would like to run the tests without use the Makefile targets, for example via an IDE, then you can set the environment variables directly in the code of your `suite_test.go`:

```go 
var _ = BeforeSuite(func(done Done) {
	Expect(os.Setenv("TEST_ASSET_KUBE_APISERVER", "../testbin/bin/kube-apiserver")).To(Succeed())
	Expect(os.Setenv("TEST_ASSET_ETCD", "../testbin/bin/etcd")).To(Succeed())
	Expect(os.Setenv("TEST_ASSET_KUBECTL", "../testbin/bin/kubectl")).To(Succeed())

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	testenv = &envtest.Environment{}

	_, err := testenv.Start()
	Expect(err).NotTo(HaveOccurred())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).To(Succeed())

	Expect(os.Unsetenv("TEST_ASSET_KUBE_APISERVER")).To(Succeed())
	Expect(os.Unsetenv("TEST_ASSET_ETCD")).To(Succeed())
	Expect(os.Unsetenv("TEST_ASSET_KUBECTL")).To(Succeed())

})
```  

### Flags
Here's an example of modifying the flags with which to start the API server in your integration tests, compared to the default values in `envtest.DefaultKubeAPIServerFlags`:

```go
customApiServerFlags := []string{
	"--secure-port=6884",
	"--admission-control=MutatingAdmissionWebhook",
}

apiServerFlags := append([]string(nil), envtest.DefaultKubeAPIServerFlags...)
apiServerFlags = append(apiServerFlags, customApiServerFlags...)

testEnv = &envtest.Environment{
	CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	KubeAPIServerFlags: apiServerFlags,
}
```

## Testing considerations

Unless you're using an existing cluster, keep in mind that no built-in controllers are running in the test context. In some ways, the test control plane will behave differently from "real" clusters, and that might have an impact on how you write tests. One common example is garbage collection; because there are no controllers monitoring built-in resources, objects do not get deleted, even if an `OwnerReference` is set up.

To test that the deletion lifecycle works, test the ownership instead of asserting on existence. For example:

```go
expectedOwnerReference := v1.OwnerReference{
	Kind:       "MyCoolCustomResource",
	APIVersion: "my.api.example.com/v1beta1",
	UID:        "d9607e19-f88f-11e6-a518-42010a800195",
	Name:       "userSpecifiedResourceName",
}
Expect(deployment.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
```

[envtest]:https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest
[setup-envtest]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/hack/setup-envtest.sh
