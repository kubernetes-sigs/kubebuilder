# Configuring envtest for integration tests

The [`controller-runtime/pkg/envtest`][envtest] Go library helps write integration tests for your controllers by setting up and starting an instance of etcd and the 
Kubernetes API server, without kubelet, controller-manager or other components.

## Installation

Installing the binaries is as a simple as running `make envtest`. `envtest` will download the Kubernetes API server binaries to the `bin/` folder in your project
by default. `make test` is the one-stop shop for downloading the binaries, setting up the test environment, and running the tests. 

The make targets require `bash` to run. 

## Installation in Air Gaped/disconnected environments
If you would like to download the tarball containing the binaries, to use in a disconnected environment you can use 
`setup-envtest` to download the required binaries locally. There are a lot of ways to configure `setup-envtest` to avoid talking to 
the internet you can read about them [here](https://github.com/kubernetes-sigs/controller-runtime/tree/master/tools/setup-envtest#what-if-i-dont-want-to-talk-to-the-internet). 
The examples below will show how to install the Kubernetes API binaries using mostly defaults set by `setup-envtest`. 

### Download the binaries
`make envtest` will download the `setup-envtest` binary to `./bin/`. 
```shell
make envtest
```

Installing the binaries using `setup-envtest` stores the binary in OS specific locations, you can read more about them 
[here](https://github.com/kubernetes-sigs/controller-runtime/tree/master/tools/setup-envtest#where-does-it-put-all-those-binaries)
```sh
./bin/setup-envtest use 1.21.2
```

### Update the test make target
Once these binaries are installed, change the `test` make target to include a `-i` like below. `-i` will only check for locally installed
binaries and not reach out to remote resources. You could also set the `ENVTEST_INSTALLED_ONLY` env variable. 

```makefile
test: manifests generate fmt vet
    KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -i --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out
```

NOTE: The `ENVTEST_K8S_VERSION` needs to match the `setup-envtest` you downloaded above. Otherwise, you will see an error like the below
```sh
no such version (1.24.5) exists on disk for this architecture (darwin/amd64) -- try running `list -i` to see what's on disk
```

## Kubernetes 1.20 and 1.21 binary issues

There have been many reports of the `kube-apiserver` or `etcd` binary [hanging during cleanup][cr-1571]
or misbehaving in other ways. We recommend using the 1.19.2 tools version to circumvent such issues,
which do not seem to arise in 1.22+. This is likely NOT the cause of a `fork/exec: permission denied`
or `fork/exec: not found` error, which is caused by improper tools installation.

[cr-1571]:https://github.com/kubernetes-sigs/controller-runtime/issues/1571

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

Controller-runtime’s [envtest][envtest] framework requires `kubectl`, `kube-apiserver`, and `etcd` binaries be present locally to simulate the API portions of a real cluster.

The `make test` command will install these binaries to the `bin/` directory and use them when running tests that use `envtest`.
Ie,
```shell             
./bin/k8s/
└── 1.25.0-darwin-amd64
    ├── etcd
    ├── kube-apiserver
    └── kubectl

1 directory, 3 files
```

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
	Expect(os.Setenv("TEST_ASSET_KUBE_APISERVER", "../bin/k8s/1.25.0-darwin-amd64/kube-apiserver")).To(Succeed())
	Expect(os.Setenv("TEST_ASSET_ETCD", "../bin/k8s/1.25.0-darwin-amd64/etcd")).To(Succeed())
	Expect(os.Setenv("TEST_ASSET_KUBECTL", "../bin/k8s/1.25.0-darwin-amd64/kubectl")).To(Succeed())
	// OR
	Expect(os.Setenv("KUBEBUILDER_ASSETS", "../bin/k8s/1.25.0-darwin-amd64")).To(Succeed())

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
[setup-envtest]:https://pkg.go.dev/sigs.k8s.io/controller-runtime/tools/setup-envtest
