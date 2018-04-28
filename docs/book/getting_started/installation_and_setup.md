{% panel style="danger", title="STAGING" %}
Staging Environment - Not Official Documentation!

This book contains APIs, libraries and tools that are proposals only and have not been ratified!
{% endpanel %}


# Installation and Setup

{% panel style="warning", title="Important" %}
Kubebuilder packages together multiple binaries, such as code-generators, and cannot be installed with `go get`.

Kubebuilder may be built from source by locally running *Google Container Builder*.  See the
[CONTRIBUTING](https://github.com/kubernetes-sigs/kubebuilder/blob/master/CONTRIBUTING.md) docs for more information.
{% endpanel %}

{% method %}

Kubebuilder should be installed by downloading the latest stable release from the kubebuilder
[github repo](https://github.com/kubernetes-sigs/kubebuilder/releases).

Kubebuilder can then be run to setup a project from a project directory underneath the `GOPATH` by
running `kubebuilder init --domain yourdomain.com`.

{% sample lang="mac" %}
```bash
# download the release
curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/1beta1.5/kubebuilder_1beta1.5_darwin_amd64.tar.gz

# extract the archive
tar -zxvf kubebuilder_1beta1.5_darwin_amd64.tar.gz
sudo mv kubebuilder_1beta1.5_darwin_amd64 /usr/local/kubebuilder

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```

{% sample lang="linux" %}
```bash
# download the release
wget https://github.com/kubernetes-sigs/kubebuilder/releases/download/v1beta1.5/kubebuilder_1beta1.5_darwin_amd64.tar.gz

# extract the archive
tar -zxvf kubebuilder_v1beta1.5_linux_amd64.tar.gz
sudo mv kubebuilder_v1beta1.5_linux_amd64 /usr/local/kubebuilder

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```

{% endmethod %}

