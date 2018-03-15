
**Note:** Kubebuilder is still in EAP and not ready for production use.

*Don't use `go get` / `go install`, instead you MUST download a tar binary release or create your own release using
the release program.*  To build your own release see [CONTRIBUTING.md](CONTRIBUTING.md)

## `kubebuilder`

Kubebuilder is a framework for building Kubernetes APIs.

**Note:** kubebuilder does not exist as an example to *copy-paste*, but instead provides powerful libraries and tools
to simplify building and publishing Kubernetes APIs from scratch.

## TL;DR

**First:** Download the latest release and extract the tar.gz into /usr/local/ and update your PATH to
include /usr/local/kubebuilder/bin.

Create a project under a new GOPATH.

```sh
# Initialize your project
kubebuilder init --domain example.com

# Create a new API and controller
kubebuilder create resource --group bar --version v1alpha1  --kind Foo

# Install and run your API into the cluster for your current kubeconfig context
GOBIN=$(pwd)/bin go install <PROJECT_PACKAGE>/cmd/controller-manager
bin/controller-manager --kubeconfig ~/.kube/config
```

See the [user guide](docs/tools_user_guide.md) for more details

## Godocs

Many of the kubebuilder libraries can be used on their own without the kubebuilder code generation and scaffolding.

See examples of using the libraries directly below:

- [controller libraries](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/controller)
- [config libraries](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/config)
- [signals libraries](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/signals)

Kubebuilder code generation tags can be found here:

- [resource code generation tags](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/apis)
- [controllers code generation tags](https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller)

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
