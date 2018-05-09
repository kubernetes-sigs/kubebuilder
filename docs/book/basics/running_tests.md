{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Running tests

Kubebuilder will create scaffolding tests for controllers and resources.  When run, these tests will start
a local control plane as part of the integration test.  Developers may talk to the local control plane
using the provided config.

#### Resource Tests

The resource tests are created under `pkg/apis/<group>/<version>/<kind>_types_test.go`.  When a resource
is created with `kubebuilder create resource`, a test file will be created to store and read back the object.

Update the test to include validation you add to your resource.

For more on Resources see [What Is A Resource](../basics/what_is_a_resource.md) 


#### Controller Tests

The controller tests are created under `pkg/controller/<kind>/controller_test.go`.  When a resource
is created with `kubebuilder create resource`, a test file will be created to start the controller
and reconcile objects.  The default test will create a new object and verify that the controller
Reconcile function is called.

Update the test to verify the business logic of your controller.

For more on Controllers see [What Is A Controller](../basics/what_is_a_controller.md) 

{% method %}
## Run the tests

Run the tests using `go test`.

{% sample lang="shell" %}
```bash
go test ./pkg/...
```
{% endmethod %}


{% method %}
## Optional: Change Control Plane Test Binaries

To override the test binaries used to start the control plane, set the `TEST_ASSET_` environment variables.
This can be useful for performing testing against multiple Kubernetes cluster versions.

If these environment variables are unset, kubebuiler will default to the binaries packaged with kubebuilder.

{% sample lang="shell" %}
```bash
export TEST_ASSET_KUBECTL=/usr/local/kubebuilder/bin/kubectl
export TEST_ASSET_KUBE_APISERVER=/usr/local/kubebuilder/bin/kube-apiserver
export TEST_ASSET_ETCD=/usr/local/kubebuilder/bin/etcd
```
{% endmethod %}


