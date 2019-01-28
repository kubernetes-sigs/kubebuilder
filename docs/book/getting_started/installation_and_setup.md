# Installation and Setup

Kubebuilder requires multiple binaries to be installed and cannot be installed with `go get`.

{% method %}

## Installing a stable release

Install kubebuilder by downloading the latest stable release from the
[github repo](https://github.com/kubernetes-sigs/kubebuilder/releases).

{% sample lang="mac" %}
```bash
version=1.0.8 # latest stable version
arch=amd64

# download the release
curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${version}/kubebuilder_${version}_darwin_${arch}.tar.gz"

# extract the archive
tar -zxvf kubebuilder_${version}_darwin_${arch}.tar.gz
mv kubebuilder_${version}_darwin_${arch} kubebuilder && sudo mv kubebuilder /usr/local/

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```

{% sample lang="linux" %}
```bash
version=1.0.8 # latest stable version
arch=amd64

# download the release
curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${version}/kubebuilder_${version}_linux_${arch}.tar.gz"

# extract the archive
tar -zxvf kubebuilder_${version}_linux_${arch}.tar.gz
mv kubebuilder_${version}_linux_${arch} kubebuilder && sudo mv kubebuilder /usr/local/

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```

{% endmethod %}

{% method %}

## Installing latest release from master

You can install the latest kubebuilder release built from the master. Note that
this release is not well tested, so you might encounter some bugs.

{% sample lang="mac" %}
```bash
arch=amd64

# download the release
curl -L -O https://storage.googleapis.com/kubebuilder-release/kubebuilder_master_darwin_${arch}.tar.gz

# extract the archive
tar -zxvf kubebuilder_master_darwin_${arch}.tar.gz
mv kubebuilder_master_darwin_${arch} kubebuilder && sudo mv kubebuilder /usr/local/

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```
{% sample lang="linux" %}
```bash
arch=amd64

# download the release
curl -L -O https://storage.googleapis.com/kubebuilder-release/kubebuilder_master_linux_${arch}.tar.gz

# extract the archive
tar -zxvf kubebuilder_master_linux_${arch}.tar.gz
mv kubebuilder_master_linux_${arch} kubebuilder && sudo mv kubebuilder /usr/local/

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```
{% endmethod %}
