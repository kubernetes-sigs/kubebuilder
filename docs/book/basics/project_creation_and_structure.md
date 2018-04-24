{% panel style="danger", title="STAGING" %}
Staging Environment - Not Official Documentation!

This book contains APIs, libraries and tools that are proposals only and have not been ratified!
{% endpanel %}


# Project Creation and Structure {#project-creation-and-structure}

## Go package Structure

Kubebuilder projects closely mirror the core Kubernetes project structure, and have 3 important
packages.  These packages and are automatically created and populated by kubebuilder commands.

##### cmd/...

The `cmd` package contains the controller-manager main program with will run your controllers
and may install the APIs.  Users typically will not need to modify this package unless they
are doing something special.

##### pkg/apis/...

The `pkg/apis/...` packages contain the *go* structs the define the resource API schemas.  This
is where users modify `*_types.go` files to add fields to APIs.

Each API lives in `pkg/apis/<api-group-name>/<api-version-name>/<api-kind-name>_types.go`.

More information on API Group, Version and Kinds in the *What is a Resource* chapter.

{% panel style="info", title="Generated code" %}
Kubebuilder generates boilerplate code to add required types and register APIs in this package.
Boilerplate code is written to `zz_generated.*` files, and should only be written to
by kubebuilder.  More on this in *Generated Code*.
{% endpanel %}

##### pkg/controller/...

The `pkg/controller/...` packages contain the *go* types and functions that implement the
APIs as *controllers*.

More information on Controllers in the *What is a Controller* chapter.

## Additional directories

Kubebuilder manages other directories users may interact with

##### hack/...

Kubebuilder puts various files into the hack directory, such as a Dockerfile for building the
controller-manager binary, installation yaml files, and samples configs to create resources.

##### docs/...

Kubebuilder stores generated API reference documentation, and user authored samples and
conceptual documentation under the docs directory.

{% method %}
## Create a new project

Initialize a kubebuilder project from your go project directory.  This will automatically create the
vendored go libaries needed to build your controller-manager.

{% sample lang="bash" %}
```bash
$ kubebuilder init --domain k8s.io
```
{% endmethod %}

{% method %}
## Create a new API

Create the *_types.go file and controller.go files for a new API and register them with
the main program.

{% sample lang="bash" %}
```bash
$ kubebuilder create resource --group mygroup --version v1beta1 --kind MyKind
```
{% endmethod %}

{% method %}
## Run your controller-manager localy against a Kubernetes cluster

Users may run the controller-manager locally against a Kubernetes cluster.  This will
install the APIs and being the controller-manager watching resources and reconciling events.

{% sample lang="bash" %}
```bash
$ GOBIN=${PWD}/bin go install ${PWD#$GOPATH/src/}/cmd/controller-manager
$ bin/controller-manager --kubeconfig ~/.kube/config
```
{% endmethod %}

{% method %}
## Create a resource

Kubebuilder creates the scaffolding object for new resources.  Create a new instance
of your API.

{% sample lang="bash" %}
```bash
$ kubectl apply -f hack/sample/<resource>.yaml
```
{% endmethod %}
