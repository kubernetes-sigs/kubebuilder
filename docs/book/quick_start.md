# Quick Start

This Quick Start guide will cover:

- Create a project
- Create an API
- Run locally
- Run in-cluster
- Build documentation

## Installation
{% method %}

- Install [dep](https://github.com/golang/dep)
- Install [kustomize](https://github.com/kubernetes-sigs/kustomize)
- Install [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)

{% sample lang="mac" %}
```bash
version=1.0.0 # latest stable version
arch=amd64

# download the release
curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v$version/kubebuilder_$version_darwin_$arch.tar.gz

# extract the archive
tar -zxvf kubebuilder_$version_darwin_$arch.tar.gz
sudo mv kubebuilder_$version_darwin_$arch /usr/local/kubebuilder

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```

{% sample lang="linux" %}
```bash
version=1.0.0 # latest stable version
arch=amd64

# download the release
curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v$version/kubebuilder_$version_linux_$arch.tar.gz

# extract the archive
tar -zxvf kubebuilder_$version_linux_$arch.tar.gz
sudo mv kubebuilder_$version_linux_$arch /usr/local/kubebuilder

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin
```
{% endmethod %}

## Create a new API

{% method %}

#### Project Creation

Initialize the project directory.

```bash
kubebuilder init --domain k8s.io --license apache2 --owners "The Kubernetes Authors"
```
{% endmethod %}

{% method %}

#### API Creation

Create a new API called *Sloop*.  The will create files for you to edit under `pkg/apis/<group>/<version>` and under
`pkg/controller/<kind>`.

**Optional:** Edit the schema or reconcile business logic in the `pkg/apis` and `pkg/controller` respectively.
For more on this see [What is a Controller](basics/what_is_a_controller.md)
and [What is a Resource](basics/what_is_a_resource.md)

```bash
kubebuilder create api --group ships --version v1beta1 --kind Sloop
```
{% endmethod %}

{% method %}

#### Locally Running An API

**Optional:** Create a new [minikube](https://github.com/kubernetes/minikube) cluster for development.

Build and run your API by installing the CRD into the cluster and starting the controller as a local
process on your dev machine.

Create a new instance of your API and look at the command output.

> Install the CRDs into the cluster

```bash
make install
```

> Run the command locally against the remote cluster.

```bash
make run
```

> In a new terminal - create an instance and expect the Controller to pick it up

```bash
kubectl apply -f config/samples/ships_v1beta1_sloop.yaml
```
{% endmethod %}

{% method %}

#### Adding Schema and Business Logic

Edit your API Schema and Controller, then re-run `make`.

```bash
nano -w pkg/apis/ship/v1beta1/sloop_types.go
...
nano -w pkg/controller/sloop/sloop_controller.go
...
make
```
{% endmethod %}

## Publishing

{% method %}

#### Controller-Manager Container and Manifests installtion

- Build and push a container image.
- Create installation manifests for your API
- Run in-cluster with kubectl apply

```bash
make
```

```bash
export IMG=gcr.io/kubeships/manager:v1
gcloud auth configure-docker
make docker-build
make docker-push
make deploy
```

{% endmethod %}

{% method %}

#### API Documentation

Generate documentation:

- Create an example of your API
- Generate the docs
- View the generated docs at `docs/reference/build/index.html`

```bash
kubebuilder create example  --version v1beta1 --group ships.k8s.io --kind Sloop
nano -w docs/reference/examples/sloop/sloop.yaml
...
```

```bash
kubebuilder docs
```
{% endmethod %}
