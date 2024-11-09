# Configuring envtest for integration tests

The [`controller-runtime/pkg/envtest`][envtest] Go library helps write integration tests for your controllers by setting up and starting an instance of etcd and the
Kubernetes API server, without kubelet, controller-manager or other components.

## Installation

Installing the binaries is as a simple as running `make envtest`. `envtest` will download the Kubernetes API server binaries to the `bin/` folder in your project
by default. `make test` is the one-stop shop for downloading the binaries, setting up the test environment, and running the tests.


You can refer to the Makefile of the Kubebuilder scaffold and observe that the envtest setup is consistently aligned across all controller-runtime releases. Starting from `release-0.19`, it is configured to automatically download the artefact from the correct location, **ensuring that kubebuilder users are not impacted.**

```shell
ENVTEST_K8S_VERSION = 1.31.0
ENVTEST_VERSION ?= release-0.19
...
.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))
```

## Installation in Air Gapped/disconnected environments
If you would like to download the tarball containing the binaries, to use in a disconnected environment you can use
[`setup-envtest`][setup-envtest] to download the required binaries locally. There are a lot of ways to configure `setup-envtest` to avoid talking to
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
./bin/setup-envtest use 1.31.0
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

<aside class="note">
<h1>Examples</h1>

You can use the plugin [DeployImage](../plugins/available/deploy-image-plugin-v1-alpha.md) to check examples. This plugin allows users to scaffold API/Controllers to deploy and manage an Operand (image) on the cluster following the guidelines and best practices. It abstracts the complexities of achieving this goal while allowing users to customize the generated code.

Therefore, you can check that a test using ENV TEST will be generated for the controller which has the purpose to ensure that the Deployment is created successfully. You can see an example of its code implementation under the `testdata` directory with the [DeployImage](../plugins/available/deploy-image-plugin-v1-alpha.md) samples [here](https://github.com/kubernetes-sigs/kubebuilder/blob/v3.7.0/testdata/project-v4-with-plugins/controllers/busybox_controller_test.go).

</aside>

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
```

You can use environment variables and/or flags to specify the `kubectl`,`api-server` and `etcd` setup within your integration tests.

### Environment Variables

| Variable name | Type | When to use |
| --- | :--- | :---                                                                                                                                                                                                                                                    |
| `USE_EXISTING_CLUSTER` | boolean | Instead of setting up a local control plane, point to the control plane of an existing cluster. |
| `KUBEBUILDER_ASSETS` | path to directory | Point integration tests to a directory containing all binaries (api-server, etcd and kubectl).                                                                                                                                                          |
| `TEST_ASSET_KUBE_APISERVER`, `TEST_ASSET_ETCD`, `TEST_ASSET_KUBECTL` | paths to, respectively, api-server, etcd and kubectl binaries | Similar to `KUBEBUILDER_ASSETS`, but more granular. Point integration tests to use binaries other than the default ones. These environment variables can also be used to ensure specific tests run with expected versions of these binaries.            |
| `KUBEBUILDER_CONTROLPLANE_START_TIMEOUT` and `KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT` | durations in format supported by [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration) | Specify timeouts different from the default for the test control plane to (respectively) start and stop; any test run that exceeds them will fail.                                                                                                      |
| `KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT` | boolean | Set to `true` to attach the control plane's stdout and stderr to os.Stdout and os.Stderr. This can be useful when debugging test failures, as output will include output from the control plane.                                                        |

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

<aside class="note">
<h1>ENV TEST Config Options</h1>

You can look at the controller-runtime docs to know more about its configuration options, see [here](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest#Environment). On top of that, if you are
looking to use ENV TEST to test your webhooks then you might want to give a look at its install [options](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest#WebhookInstallOptions).

</aside>

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

<aside class="warning">

<h2>Namespace usage limitation</h2>

EnvTest does not support namespace deletion. Deleting a namespace will seem to succeed, but the namespace will just be put in a Terminating state, and never actually be reclaimed. Trying to recreate the namespace will fail. This will cause your reconciler to continue reconciling any objects left behind, unless they are deleted.

To overcome this limitation you can create a new namespace for each test. Even so, when one test completes (e.g. in "namespace-1") and another test starts (e.g. in "namespace-2"), the controller will still be reconciling any active objects from "namespace-1". This can be avoided by ensuring that all tests clean up after themselves as part of the test teardown.  If teardown of a namespace is difficult, it may be possible to wire the reconciler in such a way that it ignores reconcile requests that come from namespaces other than the one being tested:

```go
type MyCoolReconciler struct {
	client.Client
	...
	Namespace     string  // restrict namespaces to reconcile
}
func (r *MyCoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("myreconciler", req.NamespacedName)
	// Ignore requests for other namespaces, if specified
	if r.Namespace != "" && req.Namespace != r.Namespace {
		return ctrl.Result{}, nil
	}
```
Whenever your tests create a new namespace, it can modify the value of reconciler.Namespace. The reconciler will effectively ignore the previous namespace.
For further information see the issue raised in the controller-runtime [controller-runtime/issues/880](https://github.com/kubernetes-sigs/controller-runtime/issues/880) to add this support.
</aside>

## Cert-Manager and Prometheus options

Projects scaffolded with Kubebuilder can enable the [`metrics`][metrics] and the [`cert-manager`][cert-manager] options. Note that when we are using the ENV TEST we are looking to test the controllers and their reconciliation. It is considered an integrated test because the ENV TEST API will do the test against a cluster and because of this the binaries are downloaded and used to configure its pre-requirements, however, its purpose is mainly to `unit` test the controllers.

Therefore, to test a reconciliation in common cases you do not need to care about these options. However, if you would like to do tests with the Prometheus and the Cert-manager installed you can add the required steps to install them before running the tests.
Following an example.

```go
    // Add the operations to install the Prometheus operator and the cert-manager
    // before the tests.
    BeforeEach(func() {
        By("installing prometheus operator")
        Expect(utils.InstallPrometheusOperator()).To(Succeed())

        By("installing the cert-manager")
        Expect(utils.InstallCertManager()).To(Succeed())
    })

    // You can also remove them after the tests::
    AfterEach(func() {
        By("uninstalling the Prometheus manager bundle")
        utils.UninstallPrometheusOperManager()

        By("uninstalling the cert-manager bundle")
        utils.UninstallCertManager()
    })
```

Check the following example of how you can implement the above operations:

```go
const (
	prometheusOperatorVersion = "0.51"
	prometheusOperatorURL     = "https://raw.githubusercontent.com/prometheus-operator/" + "prometheus-operator/release-%s/bundle.yaml"
	certmanagerVersion = "v1.5.3"
	certmanagerURLTmpl = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	_, err := Run(cmd)
	return err
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	//was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
		)

	_, err := Run(cmd)
	return err
}

// LoadImageToKindClusterWithName loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}

	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}
```
However, see that tests for the metrics and cert-manager might fit better well as e2e tests and not under the tests done using ENV TEST for the controllers. You might want to give a look at the [sample example][sdk-e2e-sample-example] implemented into [Operator-SDK][sdk] repository to know how you can write your e2e tests to ensure the basic workflows of your project.
Also, see that you can run the tests against a cluster where you have some configurations in place they can use the option to test using an existing cluster:

```go
testEnv = &envtest.Environment{
	UseExistingCluster: true,
}
```

<aside class="note">
<h1>Setup ENV TEST tool</h1>
To know more about the tooling used to configure ENVTEST which is used in the setup-envtest target in the Makefile
of the projects build with Kubebuilder see the [README][readme]
of its tooling.
</aside>

[metrics]: https://book.kubebuilder.io/reference/metrics.html
[envtest]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest
[setup-envtest]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/tools/setup-envtest
[cert-manager]: https://book.kubebuilder.io/cronjob-tutorial/cert-manager.html
[sdk-e2e-sample-example]: https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/memcached-operator/test/e2e
[sdk]: https://github.com/operator-framework/operator-sdk
[readme]: https://github.com/kubernetes-sigs/controller-runtime/blob/main/tools/setup-envtest/README.md
