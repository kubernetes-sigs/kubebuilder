# Quick Start

This Quick Start guide will cover.

- Create a project
- Create an API
- Run the API

## Installation
{% method %}

- Install [dep](https://github.com/golang/dep)
- Download the latest release from the [releases page](https://github.com/kubernetes-sigs/kubebuilder/releases)
- Extract the tar and move+rename the extracted directory to `/usr/local/kubebuilder`
- Add `/usr/local/kubebuilder/bin` to your `PATH`

{% sample lang="bash" %}
```bash
curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v0.1.X/kubebuilder_0.1.X_<darwin|linux>_amd64.tar.gz

tar -zxvf kubebuilder_0.1.X_<darwin|linux>_amd64.tar.gz
sudo mv kubebuilder_0.1.X_<darwin|linux>_amd64 /usr/local/kubebuilder

export PATH=$PATH:/usr/local/kubebuilder/bin
```
{% endmethod %}

## Create a new API

{% method %}

#### Project Creation

Initialize the project directory with the canonical project structure and go dependencies.

{% sample lang="bash" %}
```bash
kubebuilder init --domain k8s.io
```
{% endmethod %}

{% method %}

#### API Creation

Create a new API called *Sloop*.  The will create files for you to edit under `pkg/apis/<group>/<version>` and under
`pkg/controller/<kind>`.

**Optional:** Edit the schema or reconcile business logic in the `pkg/apis` and `pkg/controller` respectively,
**then run** `kubebuilder generate`.  For more on this see [What is a Controller](basics/what_is_a_controller.md)
and [What is a Resource](basics/what_is_a_resource.md)

{% sample lang="bash" %}
```bash
kubebuilder create resource --group ships --version v1beta1 --kind Sloop
```
{% endmethod %}

{% method %}

#### Locally Running An API

**Optional:** Create a new [minikube](https://github.com/kubernetes/minikube) cluster for development.

Build and run your API by installing the CRD into the cluster and starting the controller as a local
process on your dev machine.

Create a new instance of your API and look at the controller-manager output.

{% sample lang="bash" %}
```bash
GOBIN=${PWD}/bin go install ${PWD#$GOPATH/src/}/cmd/controller-manager
bin/controller-manager --kubeconfig ~/.kube/config
```

> In a new terminal create an instance of your API

```bash
kubectl apply -f hack/sample/sloop.yaml
```
{% endmethod %}

{% method %}

#### Adding Schema and Business Logic

Further your API schema and resource, then run `kubebuilder generate`.

{% sample lang="bash" %}
```bash
nano -w pkg/apis/ship/v1beta1/sloop_types.go
...
nano -w pkg/controller/sloop/controller.go
...
kubebuilder generate
```
{% endmethod %}

## Publishing

{% method %}

#### Integration Testing

Run the generated integration tests for your APIS.

{% sample lang="bash" %}
```bash
go test ./pkg/...
```
{% endmethod %}

{% method %}

#### Controller-Manager Container and Installation YAML

- Build and push a container image.
- Create installation config for your API
- Install with kubectl apply

{% sample lang="bash" %}

```bash
docker build . -f Dockerfile.controller -t gcr.io/kubeships/controller-manager:v1
kubebuilder create config --controller-image gcr.io/kubeships/controller-manager:v1 --name kubeships
```

```bash
gcloud auth configure-docker
docker push gcr.io/kubeships/controller-manager:v1
```

```bash
kubectl apply -f hack/install.yaml
```
{% endmethod %}

{% method %}

#### API Documentation

Generate documentation:

- Create an example of your API
- Generate the docs
- View the generated docs at `docs/reference/build/index.html`

{% sample lang="bash" %}
```bash
kubebuilder create example  --version v1beta1 --group ships.k8s.io --kind Sloop
nano -w docs/reference/examples/sloop/sloop.yaml
...
```

```bash
kubebuilder docs
```
{% endmethod %}