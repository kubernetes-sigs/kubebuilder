# Getting started

This document covers building an API using CRDs and a controller
`kubebuilder`.  It is focused on how to use the most basic aspects of
the tooling to be productive quickly.

For information on the libraries, see the [libraries user guide](libraries_user_guide.md)

New API workflow:

- Bootstrap go vendor + initialize required directory structure and go packages
- Create an API group, version, resource + controller
- Build and run against a Kubernetes cluster
- Run tests

## Download the latest release

Make sure you downloaded and installed the latest release:
[here](https://github.com/kubernetes-sigs/kubebuilder/blob/master/docs/installing.md)

- Download the latest [release](https://github.com/kubernetes-sigs/kubebuilder/releases/)
- Extract the tar and move the kubebuilder/ directory to `/usr/local` (or somewhere else on your path)
- Add `/usr/local/kubebuilder/bin` to your path - `export PATH=$PATH:/usr/local/kubebuilder/bin`
- Set environment variables for starting test control planes
  - export `TEST_ASSET_KUBECTL=/usr/local/kubebuilder/bin/kubectl`
  - export `TEST_ASSET_KUBE_APISERVER=/usr/local/kubebuilder/bin/kube-apiserver`
  - export `TEST_ASSET_ETCD=/usr/local/kubebuilder/bin/etcd`


## Create your Go project

Create a Go project under GOPATH/src/

For example

> GOPATH/src/github.com/my-org/my-project

## Optional: Create a copyright header

Create a file called `boilerplate.go.txt`.  This file will contain the
copyright boilerplate appearing at the top of all generated files.

Under GOPATH/src/github.com/my-org/my-project/:

- `hack/boilerplate.go.txt`

e.g.

```go
/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
```

## Initialize your project

This will setup the initial structure for your project with:

- An empty boilerplate.go.txt (if one doesn't already exist)
- Base `vendor/` go libraries and Gopkg.toml / Gopkg.lock (extracted from the kubebuilder installation directory)
- Dockerfiles for creating your project's container images
- Optionally: A Bazel workspace and BUILD.bazel
  - use the `--bazel` flag to enable this

Flags:

- your-domain: unique namespace for your API groups

At the root of your go package under your GOPATH run the following command:

```sh
kubebuilder init --domain <your-domain>
```

## Create an API

An API resource provides REST endpoints for CRUD operations on a resource type and is defined by an API group
(e.g. package), version (v1alpha1, v1beta1, v1), and Kind (e.g. name).

Run the `kubebuilder create resource` command to create a new API resource definition and controller (optional).

Flags:

- your-group: name of the API group e.g. `batch`
- your-version: name of the API version e.g. `v1beta1` or `v1`
- your-kind: **Upper CamelCase** name of the type e.g. `MyKind`

At the root of your go package under your GOPATH run the following command:

```sh
kubebuilder create resource --group <yourgroup> --version <yourversion> --kind <YourKind>
```

## Setup the CRD + controller against a remote cluster (run locally)

```sh
GOBIN=$(pwd)/bin go install <PROJECT_PACKAGE>/cmd/controller-manager
bin/controller-manager --kubeconfig ~/.kube/config
```

> **Note:** by default the controller-manager will install or update the CRDs before starting.

Code generates and building executables maybe run separate using `kubebuilder build generated` or `kubebuilder build executables`.

> **Note:** The generators must be rerun after fields are added or removed from your resources

## Create a new instance of your CRD

A sample CRD for you to play with was created under hack/sample by `kubebuilder create resource`.

```sh
kubectl create -f hack/sample/<type>.yaml
kubectl get <type>s
```

Look at the controller logs to see the reconcile loop print a message

## Run the tests

A placeholder test was created for your resource to make sure it can be stored, read and reconciled by the controller.
The tests require the binaries for starting a local control plane to be defined with `TEST_ASSET_` env vars.

```sh
TEST_ASSET_KUBECTL=/usr/local/kubebuilder/bin/kubectl \
TEST_ASSET_KUBE_APISERVER=/usr/local/kubebuilder/bin/kube-apiserver \
TEST_ASSET_ETCD=/usr/local/kubebuilder/bin/etcd \
go test ./pkg/...
```

## Build and run an image for your CRD and Controller

`Dockerfile`s for the controller-manager and installer containers were created at the project root.
The controller-manager Dockerfile will build the controller-manager from source and also run the tests under
`./pkg/...` and `./cmd/...`.

```sh
docker build . -f Dockerfile.install -t <install-image>:<version> && docker push <install-image>:<version>
docker build . -f Dockerfile.controller -t <controller-image>:<version> && docker push <controller-image>:<version>
```

### Install using the container images

To install the controller-manager into a cluster using the installer container - create an installer serviceaccount
and run the installer container.

```sh
kubectl create serviceaccount installer
kubectl create clusterrolebinding installer-cluster-admin-binding --clusterrole=cluster-admin \
  --serviceaccount=default:installer

export NAME=my-project-name
export INSTALL_IMAGE=<install-image>
export CONTROLLER_IMAGE=<controller-image>
kubectl run $NAME --serviceaccount=installer --image=$INSTALL_IMAGE --restart=OnFailure -- ./installer \
  --controller-image=$CONTROLLER_IMAGE --name=$NAME
```
