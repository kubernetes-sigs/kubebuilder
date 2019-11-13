## Using envtest in integration tests
[`controller-runtime`](http://sigs.k8s.io/controller-runtime) offers `envtest` ([godoc](https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/envtest)), a package that helps write integration tests for your controllers by setting up and starting an instance of etcd and the Kubernetes API server, without kubelet, controller-manager or other components.

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
You can use environment variables and/or flags to specify the `api-server` and `etcd` setup within your integration tests.

#### Environment Variables

| Variable name | Type | When to use |
| --- | :--- | :--- |
| `USE_EXISTING_CLUSTER` | boolean | Instead of setting up a local control plane, point to the control plane of an existing cluster. |
| `KUBEBUILDER_ASSETS` | path to directory | Point integration tests to a directory containing all binaries (api-server, etcd and kubectl). |
| `TEST_ASSET_KUBE_APISERVER`, `TEST_ASSET_ETCD`, `TEST_ASSET_KUBECTL` | paths to, respectively, api-server, etcd and kubectl binaries | Similar to `KUBEBUILDER_ASSETS`, but more granular. Point integration tests to use binaries other than the default ones. These environment variables can also be used to ensure specific tests run with expected versions of these binaries. |
| `KUBEBUILDER_CONTROLPLANE_START_TIMEOUT` and `KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT` | durations in format supported by [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration) | Specify timeouts different from the default for the test control plane to (respectively) start and stop; any test run that exceeds them will fail. |
| `KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT` | boolean | Set to `true` to attach the control plane's stdout and stderr to os.Stdout and os.Stderr. This can be useful when debugging test failures, as output will include output from the control plane. |


#### Flags
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

### Testing considerations

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
