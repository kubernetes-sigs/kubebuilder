{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Running tests

Kubebuilder `kubebuilder create resource` will create scaffolding tests for controllers and resources
along side the controller and resource code.  When run, these tests will start a local control plane
as part of the integration test.  Developers may talk to the local control plane using the provided
config.

{% method %}
## Setup Environment Variables

First export the environment variables so the test harness can locate the control plane binaries.
The control plane binaries are included with kubebuilder.

{% sample lang="shell" %}
```bash
export TEST_ASSET_KUBECTL=/usr/local/kubebuilder/bin/kubectl
export TEST_ASSET_KUBE_APISERVER=/usr/local/kubebuilder/bin/kube-apiserver
export TEST_ASSET_ETCD=/usr/local/kubebuilder/bin/etcd
```
{% endmethod %}

{% method %}
## Run the tests

Next run the tests as normal.

{% sample lang="shell" %}
```bash
go test ./pkg/...
```
{% endmethod %}

