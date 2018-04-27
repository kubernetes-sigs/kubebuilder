[![Build Status](https://travis-ci.org/kubernetes-sigs/kubebuilder.svg?branch=master)](https://travis-ci.org/kubernetes-sigs/kubebuilder "Travis")
[![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes-sigs/kubebuilder)](https://goreportcard.com/report/github.com/kubernetes-sigs/kubebuilder)

*Don't use `go get` / `go install`, instead you MUST download a tar binary release or create your own release using
the release program.*  To build your own release see [CONTRIBUTING.md](CONTRIBUTING.md)

## Releases

### 1.9 Kubernetes

Release:

- [v1beta1.1](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v1beta1.1)

Latest:

- [darwin master HEAD](https://storage.googleapis.com/kubebuilder-release/kubebuilder_master_darwin_amd64.tar.gz)
- [linux master HEAD](https://storage.googleapis.com/kubebuilder-release/kubebuilder_master_linux_amd64.tar.gz)


## `kubebuilder`

Kubebuilder is a framework for building Kubernetes APIs.

**Note:** kubebuilder does not exist as an example to *copy-paste*, but instead provides powerful libraries and tools
to simplify building and publishing Kubernetes APIs from scratch.

## TL;DR

**First:** Download the latest `kubebuilder_<version>_<operating-system>_amd64.tar.gz` release. Extracting the archive will give `kubebuilder_<version>_<os>_amd64` directory. Move the extracted directory to /usr/local/kubebuilder and update your PATH to include /usr/local/kubebuilder/bin. Given below are the steps:

```shell

# Download the release
wget /path/to/kubebuilder_<version>_<operating-system>_amd64.tar.gz

# Extract the archive
tar -zxvf kubebuilder_<version>_<operating-system>_amd64.tar.gz

sudo mv kubebuilder_<version>_<operating-system>_amd64 /usr/local/kubebuilder

# Update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin

```

Create an _empty_ project under a new GOPATH.

```sh
# Initialize your project
kubebuilder init --domain example.com

# Create a new API and controller
kubebuilder create resource --group bar --version v1alpha1  --kind Foo
kubectl apply -f hack/sample/foo.yaml

# Install and run your API into the cluster for your current kubeconfig context
GOBIN=$(pwd)/bin go install <PROJECT_PACKAGE>/cmd/controller-manager
bin/controller-manager --kubeconfig ~/.kube/config
kubectl apply -f hack/sample/foo.yaml

# Build your documentation
kubebuilder create example --group bar --version v1alpha1 --kind Foo
kubebuilder docs
```

See the [user guide](docs/tools_user_guide.md) for more details

## Project structure

Following describes the project structure setup by kubebuilder commands.

### cmd/

*Most users do not need to edit this package.*

The `cmd` package contains the main function for launching a new controller manager.  It is responsible for parsing
a `rest.Config` and invoking the `inject` package to run the various controllers.  It may optionally install CRDs
as part of starting up.

This package is created automatically by running:

```sh
kubebuilder init --domain k8s.io
```

### pkg/apis

**Users must edit packages under this package**

The `apis` package contains the schema definitions for the resource types you define.  Resources are defined under
`pkg/apis/<group>/<version>/<kind>_types.go`.  Resources may be annotated with comments to identify resources
to code generators.  Comments are also used to generated field validation.

Notably client generation, CRD generation, docs generation and config generation are all driven off of annotations
on resources package.

See documentation and examples of annotations in godoc format here:
[gen/apis](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/apis#example-package)

Subpackages of apis are created when running the following command to create a new resource and controller:

```sh
kubebuilder create resource --group mygroup --version v1beta1 --kind MyKind
```

**Note:** While `create resource` automatically runs the code generators for the user, when
the user changes the resource file or adds `// +kubebuilder` annotations to the controller,
they will need to run `kubebuilder generate` to rerun the code generators.

### pkg/controllers

**Users must edit packages under this package**

The `controllers` package contains the controllers to implement the resource APIs.  Controllers are defined under
`pkg/controllers/<kind>/controller.go`.  Controllers may be annotated with comments to wire the controller into the
inject package, start informers they require and install RBAC rules they require.

Subpackages of controllers are created when running the following command to create a new resource and controller:

```sh
kubebuilder create resource --group mygroup --version v1beta1 --kind MyKind
```

See documentation and examples of annotations in godoc format here:
- [gen/controller](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package)

**Note:** While `create resource` automatically runs the code generators for the user, when
the user changes the resource file or adds `// +kubebuilder` annotations to the controller,
they will need to run `kubebuilder generate` to rerun the code generators.

### pkg/inject

*Most users do not need to edit this package.*

The `inject` package contains the `RunAll` function used to start all of the registered controllers and informers.
Wiring is autogenerated based on resource and controller annotations.

Generated wiring:
- Installing CRDs
- Instantiating and starting controllers
- Starting sharedinformers
- Installing RBAC rules

### pkg/inject/args

*Only some users need to edit this package.*

The `args` package contains the struct passed to the `ProvideController` function used to instantiating controllers.
The `InjectArgs` struct and `CreateInjectArgs` function in this package may be edited to pass additional information
to the controller provider functions.  This is typically an advanced use case.

The `args` package uses the `kubebuilder/pkg/inject/args` package documented here:
[inject/args](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/inject/args)

### hack/

The `hack` directory contains generated samples and config.

The hack/install.yaml config file for installing the API extension into a cluster will be created under this
directory by running.

```sh
kubebuilder create config --controller-image mycontrollerimage --name myextensionname
```

### docs/

The `docs` package contains your examples and content for generating reference documentation for your APIs.

The docs package is created when running either

```sh
kubebuilder docs
```

or

```sh
kubebuilder create example --group mygroup --version v1beta1 --kind MyKind
```

Example reference documentation lives under `docs/reference/examples`.  Conceptual reference documentation
lives under `docs/reference/static_includes`.

### /

The project root directory contains several files

- Dockerfile.controller

Running `docker build` on this file will build a container image for running your controller.

- Gopkg.toml / Gopkg.lock

These files are used to update vendored go dependencies.

## Available controller operations

Controllers watch Kubernetes resources and call a "Reconcile" function in response to events.  Reconcile functions
are typically "level-based", that is they are notified that something has changed for given resource namespace/name key,
but not specifically what changed (e.g. add, delete, update).  This allows Reconcile functions to reconcile multiple
events at a time and more easily self-heal, as actions are taken by comparing the current desired state (resource Spec)
and the observed state of the system.

### Creating a new controller

A new GenericController is created for you by kubebuilder when creating a resource.  If you are not using
kubebuilder to create resources and manage your project, you may create a controller directly by following
this example:

[GenericController](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller#example-GenericController)


To Watch additional resources from your controller do the following in your controller.go:

1. Add a `gc.Watch*` call to the `ProvideController`.  e.g. Call gc.[WatchTransformationKeyOf](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller#example-GenericController-WatchTransformationKeyOf)
  - This will trigger Reconcile calls for events
2. Add an [// +informers:](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package) annotation
   to the `type <Kind>Controller struct` with the type of the resource you are watching
  - This will make sure the informers that watch for events are started
3. Add an [// +rbac:](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package)
  annotation to the `type <Kind>Controller struct` with the type of the resource you are watching
  - This will make sure the RBAC rules that allow the controller to watch events in a cluster are generated

Example:

```go
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list
// +kubebuilder:informers:group=core,version=v1,kind=Pod
type FooController struct{}
```

### Watching resources

Controllers watch resources to trigger reconcile functions.  It is common for controllers to reconcile a single resource
type, but watch many other resource types to trigger a reconcile.  An example of this is a Deployment controller that
watches Deployments, ReplicaSets (created by the controller) and Pods (created by the ReplicaSet controller).  Pod
status events, such as becoming healthy, may trigger actions in the Deployment controller, such as continuing
a rolling update.

#### Watching the resource managed by the controller

Controllers typically watch for events on the resource they control and in response reconcile that resource instance.

Note: Kubebuilder will automatically call the Watch function for controllers it creates when creating a resources.  If
you are not using kubebuilder to create resources and manager your project, you may have your controller watch
a resource by following this example:

[GenericController.Watch](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller#example-GenericController-Watch)

#### Watching resources created by a controller and reconciling in the controller

Controllers frequently watch for events on the resources created by the controller or transitively created by the
controller (e.g. controller creates a ReplicaSet and the ReplicaSet creates a Pod).

Events for created resources need to be mapped back to the resource instance responsible for their creation -
e.g. if there is a Pod event -then reconcile the Deployment that owns the Pod.  The owning resource is found by
looking at the instance (Pod) controller reference, looking up the object with the same namespace and name as the
owner reference and comparing the UID of the found object to the UID in the owner reference.  This is check is done
to ensure the object is infact the owner and to disambiguate multiple resources with the same name and Kind that
are logically different (e.g. they are in different groups and are totally different things).

e.g. In response to a Pod event, find the owner reference for the Pod that has `controller=true`.
Lookup the ReplicaSet with this name and compare its UID to the ownerref UID.  If they are the same
find the owner reference for the ReplicaSet that has `controller=true`.  Lookup the Deployment
with this name and compare the UID to the ownerref UID.  If they are the same reconcile the Deployment with
this namespace/name.

In order to watch objects created by a controller and reconcile the owning resource in response, the functions
to lookup the ancestors object must be provided (e.g. lookup a ReplicaSet for a namespace/name,
lookup a Deployment for a namespace/name).

You may have you controller watch a resource created by the controller and then reconcile the owning object by
following this example:

[GenericController.WatchControllerOf](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller#example-GenericController-WatchControllerOf)

[Sample](https://github.com/kubernetes-sigs/kubebuilder/blob/master/samples/controller/controller.go#L91)

#### Watching arbitrary resources and mapping them so the are reconciled in the controller

In some cases it may be necessary to watch resources not owned or created by your controller, but respond to them.
An example would be taking some action in response to the deletion or creation of Nodes in the cluster.  To do
this, your controller must watch that object (Node) and map events to the resource type of the controller.

You may watch a resource and transform it into the key of the resource the controller manages by following this example:

[GenericController.WatchTransformationOf](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller#example-GenericController-WatchTransformationOf)

#### Watching arbitrary resources and handling the events that may caused a reconciliation in the controller

In some cases it may be necessary to directly handle events and enqueue keys to be reconciled.  You may do so
by following this example:

[GenericController.WatchEvents](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller#example-GenericController-WatchEvents)

### Watching a channel for resource names and reconciling them in the controller

In some cases it may be necessary to respond to external events such as webhooks.  You may enqueue reconcile events
from arbitrary sources by using a channel and following this example:

[GenericController.WatchChannel](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller#example-GenericController-WatchChannel)

## Running tests

In order to run the integration tests, the following environment variables must be set to bring up the test environment:

```sh
export TEST_ASSET_KUBECTL=/usr/local/kubebuilder/bin/kubectl
export TEST_ASSET_KUBE_APISERVER=/usr/local/kubebuilder/bin/kube-apiserver
export TEST_ASSET_ETCD=/usr/local/kubebuilder/bin/etcd
```

Tests can then be run with:

```sh
go test ./pkg/...
```

## Godoc Links

Many of the kubebuilder libraries can be used on their own without the kubebuilder code generation and scaffolding.

See examples of using the libraries directly below:

- [controller libraries](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller)
- [config libraries](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/config)
- [signals libraries](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/signals)

Kubebuilder generates codes for custom resource fields, and controller components such as watchers and informers. You have to add code generation tags in form of comment directives to initiate the code generation:

- [resource code generation tags](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/apis)
- [controller code generation tags](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller)

For example, you have to add controller code generation tags such as `+rbac` and `+informers` in `pkg/controller/foo/controller.go` file:
```
// +controller:group=foo,version=v1beta1,kind=Bar,resource=bars
// +rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +informers:group=apps,version=v1,kind=Deployment
// +rbac:groups="",resources=pods,verbs=get;watch;list
// +informers:group=core,version=v1,kind=Pod
type FooController struct{}
```

## Motivation

Building Kubernetes tools and APIs involves making a lot of decisions and writing a lot of boilerplate.

In order to facilitate easily building Kubernetes APIs and tools using the canonical approach, this framework
provides a collection of Kubernetes development tools to minimize toil.


Kubebuilder attempts to facilitate the following developer workflow for building APIs

1. Create a new project directory
2. Create one or more resource APIs as CRDs and then add fields to the resources
3. Implement reconcile loops in controllers and watch additional resources
4. Test by running against a cluster (self-installs CRDs and starts controllers automatically)
5. Update bootstrapped integration tests to test new fields and business logic
6. Build and publish a container from the provided Dockerfile

### Scope

Building APIs using CRDs, Controllers and Admission Webhooks.

### Philosophy

Provide clean library abstractions with clear and well exampled godocs.

- Prefer using go *interfaces* and *libraries* over relying on *code generation*
- Prefer using *code generation* over *1 time init* of stubs
- Prefer *1 time init* of stubs over forked and modified boilerplate
- Never fork and modify boilerplate

### Techniques

- Provide higher level libraries on top of low level client libraries
  - Protect developers from breaking changes in low level libraries
  - Start minimal and provide progressive discovery of functionality
  - Provide sane defaults and allow users to override when they exist
- Provide code generators to maintain common boilerplate that can't be addressed by interfaces
  - Driven off of `//+` comments
- Provide bootstrapping commands to initialize new packages
