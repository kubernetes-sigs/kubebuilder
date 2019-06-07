# Quick Start

This Quick Start guide will cover:

- [Creating a project](#create-a-project)
- [Creating an API](#create-an-api)
- [Running locally](#test-it-out-locally)
- [Running in-cluster](#run-it-on-the-cluster)

## Installation

Install [kubebuilder](https://sigs.k8s.io/kubebuilder):

```bash
os=$(go env GOOS)
arch=$(go env GOARCH)

# download kubebuilder and extract it to tmp
curl -sL https://go.kubebuilder.io/dl/2.0.0-alpha.3/${os}/${arch} | tar -xz -C /tmp/

# move to a long-term location and put it on your path
# (you'll need to set the KUBEBUILDER_ASSETS env var if you put it somewhere else)
sudo mv /tmp/kubebuilder_2.0.0-alpha.3_${os}_${arch} /usr/local/kubebuilder
export PATH=$PATH:/usr/local/kubebuilder/bin
```

You can also install a KubeBuilder master snapshot from
`https://go.kubebuilder.io/dl/latest/${os}/${arch}`.

Install [kustomize](https://sigs.k8s.io/kustomize):

```bash
os=$(go env GOOS)
arch=$(go env GOARCH)

# download kustomize to the kubebuilder assets folder
curl -o /usr/local/kubebuilder/bin/kustomize -sL https://go.kubebuilder.io/kustomize/${os}/${arch}
```

## Create a Project

Initialize a new project and Go module for your controllers:

```bash
kubebuilder init --domain my.domain
```

If you're not in `GOPATH`, you'll need to run `go mod init <modulename>`
in order to tell kubebuilder and Go the base import path of your module.

## Create an API

Create a new API group-version called `webapp/v1`, and a kind `Guestbook`
in that API group-version:

```bash
kubebuilder create api --group webapp --version v1 --kind Guestbook
```

The will create the files `api/v1/guestbook_types.go` and
`controller/guestbook_controller.go` for you to edit.

**Optional:** Edit the API definition or the reconcilation business logic.
For more on this see [What is
a Controller](TODO.md) and [What is
a Resource](TODO.md)

## Test It Out Locally

You'll need a Kubernetes cluster to run against.  You can use
[KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or
run against a remote cluster.

Your controller will automatically use the current context in your
kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

Install the CRDs into the cluster:

```bash
make install
```

Run your controller (this will run in the foreground, so switch to a new
terminal if you want to leave it running):

```bash
make run
```

## Install Samples

Create your samples (make sure to edit them first if you've changed the
API definition):

```bash
kubectl apply -f config/samples/
```

## Run It On the Cluster

Build and push your image to the location specified by `IMG`:

```bash
make docker-build docker-push IMG=<some-registry>/controller
```

Deploy the controller to the cluster:

```bash
make deploy
```

If you encouter RBAC errors, you may need to grant yourself cluster-admin
privileges:

<!-- TODO(directxman12): fill this in -->
