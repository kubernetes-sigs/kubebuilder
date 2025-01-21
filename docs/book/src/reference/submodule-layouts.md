# Sub-Module Layouts

This part describes how to modify a scaffolded project for use with multiple `go.mod` files for APIs and Controllers.

Sub-Module Layouts (in a way you could call them a special form of [Monorepo's][monorepo]) are a special use case and can help in scenarios that involve reuse of APIs without introducing indirect dependencies that should not be available in the project consuming the API externally.

<aside class="note">
<h1>Using External Resources/APIs</h1>

If you are looking to do operations and reconcile via a controller a Type(CRD) which are owned by another project
or By Kubernetes API then, please see [Using an external Resources/API](/reference/using_an_external_type.md)
for more info.

</aside>

## Overview

Separate `go.mod` modules for APIs and Controllers can help for the following cases:

- There is an enterprise version of an operator available that wants to reuse APIs from the Community Version
- There are many (possibly external) modules depending on the API and you want to have a more strict separation of transitive dependencies
- If you want to reduce impact of transitive dependencies on your API being included in other projects
- If you are looking to separately manage the lifecycle of your API release process from your controller release process.
- If you are looking to modularize your codebase without splitting your code between multiple repositories.

They introduce however multiple caveats into typical projects which is one of the main factors that makes them hard to recommend in a generic use-case or plugin:

- Multiple `go.mod` modules are not recommended as a go best practice and [multiple modules are mostly discouraged][multi-module-repositories]
- There is always the possibility to extract your APIs into a new repository and arguably also have more control over the release process in a project spanning multiple repos relying on the same API types.
- It requires at least one [replace directive][replace-directives] either through `go.work` which is at least 2 more files plus an environment variable for build environments without GO_WORK or through `go.mod` replace, which has to be manually dropped and added for every release.

<aside class="note warning">
<h1>Implications on Maintenance efforts</h1>

When deciding to deviate from the standard kubebuilder `PROJECT` setup or the extended layouts offered by its plugins, it can result in increased maintenance overhead as there can be breaking changes in upstream that could break with the custom module structure described here.

Splitting your codebase to multiple repos and/or multiple modules incurs costs that will grow over time. You'll need to define clear version dependencies between your own modules, do phased upgrades carefully, etc. Especially for small-to-medium projects, one repo and one module is the best way to go.

Bear in mind, that it is not recommended to deviate from the proposed layout unless you know what you are doing.
You may also lose the ability to use some of the CLI features and helpers. For further information on the project layout, see the doc [What's in a basic project?][basic-project-doc]

</aside>

## Adjusting your Project

For a proper Sub-Module layout, we will use the generated APIs as a starting point.

For the steps below, we will assume you created your project in your `GOPATH` with

```shell
kubebuilder init
```

and created an API & controller with

```shell
kubebuilder create api --group operator --version v1alpha1 --kind Sample --resource --controller --make
```

### Creating a second module for your API

Now that we have a base layout in place, we will enable you for multiple modules.

1. Navigate to `api/v1alpha1`
2. Run `go mod init` to create a new submodule
3. Run `go mod tidy` to resolve the dependencies

Your api go.mod file could now look like this:

```go.mod
module YOUR_GO_PATH/test-operator/api/v1alpha1

go 1.21.0

require (
        k8s.io/apimachinery v0.28.4
        sigs.k8s.io/controller-runtime v0.16.3
)

require (
        github.com/go-logr/logr v1.2.4 // indirect
        github.com/gogo/protobuf v1.3.2 // indirect
        github.com/google/gofuzz v1.2.0 // indirect
        github.com/json-iterator/go v1.1.12 // indirect
        github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
        github.com/modern-go/reflect2 v1.0.2 // indirect
        golang.org/x/net v0.17.0 // indirect
        golang.org/x/text v0.13.0 // indirect
        gopkg.in/inf.v0 v0.9.1 // indirect
        gopkg.in/yaml.v2 v2.4.0 // indirect
        k8s.io/klog/v2 v2.100.1 // indirect
        k8s.io/utils v0.0.0-20230406110748-d93618cff8a2 // indirect
        sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
        sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)
```

As you can see it only includes apimachinery and controller-runtime as dependencies and any dependencies you have
declared in your controller are not taken over into the indirect imports.

### Using replace directives for development

When trying to resolve your main module in the root folder of the operator, you will notice an error if you use a VCS path:

```shell
go mod tidy
go: finding module for package YOUR_GO_PATH/test-operator/api/v1alpha1
YOUR_GO_PATH/test-operator imports
	YOUR_GO_PATH/test-operator/api/v1alpha1: cannot find module providing package YOUR_GO_PATH/test-operator/api/v1alpha1: module YOUR_GO_PATH/test-operator/api/v1alpha1: git ls-remote -q origin in LOCALVCSPATH: exit status 128:
	remote: Repository not found.
	fatal: repository 'https://YOUR_GO_PATH/test-operator/' not found
```

The reason for this is that you may have not pushed your modules into the VCS yet and resolving the main module will fail as it can no longer
directly access the API types as a package but only as a module.

To solve this issue, we will have to tell the go tooling to properly `replace` the API module with a local reference to your path.

You can do this with 2 different approaches: go modules and go workspaces.

#### Using go modules

For go modules, you will edit the main `go.mod` file of your project and issue a replace directive.

You can do this by editing the `go.mod` with
``
```shell
go mod edit -require YOUR_GO_PATH/test-operator/api/v1alpha1@v0.0.0 # Only if you didn't already resolve the module
go mod edit -replace YOUR_GO_PATH/test-operator/api/v1alpha1@v0.0.0=./api/v1alpha1
go mod tidy
```

Note that we used the placeholder version `v0.0.0` of the API Module. In case you already released your API module once,
you can use the real version as well. However this will only work if the API Module is already available in the VCS.

<aside class="note warning">
<h1>Implications on controller releases</h1>

Since the main `go.mod` file now has a replace directive, it is important to drop it again before releasing your controller module.
To achieve this you can simply run

```shell
go mod edit -dropreplace YOUR_GO_PATH/test-operator/api/v1alpha1
go mod tidy
```

</aside>

#### Using go workspaces

For go workspaces, you will not edit the `go.mod` files yourself, but rely on the workspace support in go.

To initialize a workspace for your project, run `go work init` in the project root.

Now let us include both modules in our workspace:
```shell
go work use . # This includes the main module with the controller
go work use api/v1alpha1 # This is the API submodule
go work sync
```

This will lead to commands such as `go run` or `go build` to respect the workspace and make sure that local resolution is used.

You will be able to work with this locally without having to build your module.

When using `go.work` files, it is recommended to not commit them into the repository and add them to `.gitignore`.

```gitignore
go.work
go.work.sum
```

When releasing with a present `go.work` file, make sure to set the environment variable `GOWORK=off` (verifiable with `go env GOWORK`) to make sure the release process does not get impeded by a potentially commited `go.work` file.

#### Adjusting the Dockerfile

When building your controller image, kubebuilder by default is not able to work with multiple modules.
You will have to manually add the new API module into the download of dependencies:

```dockerfile
# Build the manager binary
FROM docker.io/golang:1.20 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Copy the Go Sub-Module manifests
COPY api/v1alpha1/go.mod api/go.mod
COPY api/v1alpha1/go.sum api/go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
```

### Creating a new API and controller release

Because you adjusted the default layout, before releasing your first version of your operator, make sure to [familiarize yourself with mono-repo/multi-module releases][multi-module-repositories] with multiple `go.mod` files in different subdirectories.

Assuming a single API was created, the release process could look like this:

```sh
git commit
git tag v1.0.0 # this is your main module release
git tag api/v1.0.0 # this is your api release
go mod edit -require YOUR_GO_PATH/test-operator/api@v1.0.0 # now we depend on the api module in the main module
go mod edit -dropreplace YOUR_GO_PATH/test-operator/api/v1alpha1 # this will drop the replace directive for local development in case you use go modules, meaning the sources from the VCS will be used instead of the ones in your monorepo checked out locally.
git push origin main v1.0.0 api/v1.0.0
```

After this, your modules will be available in VCS and you do not need a local replacement anymore. However if youre making local changes,
make sure to adopt your behavior with `replace` directives accordingly.

### Reusing your extracted API module

Whenever you want to reuse your API module with a separate kubebuilder, we will assume you follow the guide for [using an external Type](/reference/using_an_external_type.md).
When you get to the step `Edit the API files` simply import the dependency with

```shell
go get YOUR_GO_PATH/test-operator/api@v1.0.0
```

and then use it as explained in the guide.

[basic-project-doc]: ./../cronjob-tutorial/basic-project.md
[monorepo]: https://en.wikipedia.org/wiki/Monorepo
[replace-directives]: https://go.dev/ref/mod#go-mod-file-replace
[multi-module-repositories]: https://github.com/golang/go/wiki/Modules#faqs--multi-module-repositories
