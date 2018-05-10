{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Installation and Setup

{% panel style="warning", title="Important" %}
Kubebuilder requires multiple binaries to be installed and cannot be installed with `go get`.

Kubebuilder may be built from source by running *Google Container Builder* locally.  See the
[CONTRIBUTING](https://github.com/kubernetes-sigs/kubebuilder/blob/master/CONTRIBUTING.md) docs for guidance.
{% endpanel %}

{% method %}

Install kubebuilder by downloading the latest stable release from the
[github repo](https://github.com/kubernetes-sigs/kubebuilder/releases).

Kubebuilder can then setup a project in the user's `GOPATH`.

`kubebuilder init --domain desiredapisdomain.com`.

{% sample lang="mac" %}
```bash
# download the release
curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v0.1.8/kubebuilder_0.1.8_darwin_amd64.tar.gz

# extract the archive
tar -zxvf kubebuilder_0.1.8_darwin_amd64.tar.gz
sudo mv kubebuilder_0.1.8_darwin_amd64 /usr/local/kubebuilder

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```

{% sample lang="linux" %}
```bash
# download the release
wget https://github.com/kubernetes-sigs/kubebuilder/releases/download/v0.1.8/kubebuilder_0.1.8_linux_amd64.tar.gz

# extract the archive
tar -zxvf kubebuilder_0.1.8_linux_amd64.tar.gz
sudo mv kubebuilder_0.1.8_linux_amd64 /usr/local/kubebuilder

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```

{% endmethod %}

