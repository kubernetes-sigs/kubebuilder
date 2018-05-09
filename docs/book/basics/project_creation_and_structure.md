{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Project Creation and Structure {#project-creation-and-structure}

## Go package Structure

Kubebuilder projects contain 4 important packages.

##### cmd/...

The `cmd` package contains the controller-manager main program which runs controllers.  Users typically
will not need to modify this package unless they are doing something special.

##### pkg/apis/...

The `pkg/apis/...` packages contains the *go* structs that define the resource schemas.
Users edit `*_types.go` files under this director to implement their API definitions.

Each resource lives in a `pkg/apis/<api-group-name>/<api-version-name>/<api-kind-name>_types.go`
file.

For more information on API Group, Version and Kinds, see the *What is a Resource* chapter.

{% panel style="info", title="Generated code" %}
Kubebuilder will generate boilerplate code required for Resources by running
`kubebuilder generate`.  The generated files are named `zz_generated.*`.
{% endpanel %}

##### pkg/controller/...

The `pkg/controller/...` packages contain the *go* types and functions that implement the
business logic for APIs in *controllers*.

More information on Controllers in the *What is a Controller* chapter.

##### pkg/inject/...

The `pkg/inject/...` packages contain the generated code that registers annotated
Controllers and Resources.

*Note*: This package is unique to kubebuilder.

## Additional directories

In addition to the packages above, a Kubebuilder project has several other directories.

##### hack/...

Kubebuilder puts miscellaneous files into the hack directory.

- API installation yaml
- Samples resource configs
- Headers for generated files: `boilerplate.go.txt`

##### docs/...

API reference documentation, user defined API samples and API conceptual documentation go here.

{% panel style="success", title="Providing boilerplate headers" %}
To prepend boilerplate comments at the top of generated and bootstrapped files,
add the boilerplate to a `hack/boilerplate.go.txt` file before creating a project.
{% endpanel %}

{% method %}
## Create a new project

Create a new kubebuilder project.  This will automatically initialize the vendored go libraries
that will be required to build your project.

{% sample lang="bash" %}
```bash
$ kubebuilder init --domain k8s.io
```
{% endmethod %}

{% method %}
## Create a new API (Resource)

Create the *_types.go file and controller.go files.

For more on resources and controllers see [What Is A Resource](../basics/what_is_a_resource.md) 
and [What Is A Controller](../basics/what_is_a_controller.md) 

{% sample lang="bash" %}
```bash
$ kubebuilder create resource --group mygroup --version v1beta1 --kind MyKind
```
{% endmethod %}

{% method %}
## Run your controller-manager locally against a Kubernetes cluster

Users may run the controller-manager binary locally against a Kubernetes cluster.  This will
install the APIs into the cluster and begin watching and reconciling the resources.

{% sample lang="bash" %}
```bash
# Create a minikube cluster
$ minikube start

# Install the APIs into the minikube cluster and begin watching it
$ GOBIN=${PWD}/bin go install ${PWD#$GOPATH/src/}/cmd/controller-manager
$ bin/controller-manager --kubeconfig ~/.kube/config
```
{% endmethod %}

{% method %}
## Create an object

Create a new instance of your Resource.  Observe the controller-manager logs after creating the object.

{% sample lang="bash" %}
```bash
$ kubectl apply -f hack/sample/<resource>.yaml
```
{% endmethod %}
