# Project Creation and Structure {#project-creation-and-structure}

## Go package Structure

Kubebuilder projects contain 3 important packages.

##### cmd/...

The `cmd` package contains the manager main program.  Manager is responsible for initializing
shared dependencies and starting / stopping Controllers.  Users typically
will not need to edit this package and can rely on the scaffolding.

The `cmd` package is scaffolded automatically by `kubebuilder init`.

##### pkg/apis/...

The `pkg/apis/...` packages contains the API resource definitions.
Users edit the `*_types.go` files under this director to implement their API definitions.

Each resource lives in a `pkg/apis/<api-group-name>/<api-version-name>/<api-kind-name>_types.go`
file.

The `pkg/apis` package is scaffolded automatically by `kubebuilder create api` when creating a Resource.

##### pkg/controller/...

The `pkg/controller/...` packages contain the Controller implementations.
Users edit the `*_controller.go` files under this directory to implement their Controllers.

The `pkg/controller` package is scaffolded automatically by `kubebuilder create api` when creating a Controller.

## Additional directories and files

In addition to the packages above, a Kubebuilder project has several other directories and files.

##### Makefile

A Makefile is created with targets to build, test, run and deploy the controller artifacts
for development as well as production workflows

##### Dockerfile

A Dockerfile is scaffolded to build a container image for your Manager.

##### config/...

Kubebuilder creates yaml config for installing the CRDs and related objects under config/.

- config/crds
- config/manager

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
$ kubebuilder init --domain k8s.io --license apache2 --owners "The Kubernetes Authors"
```
{% endmethod %}

{% method %}
## Create a new API (Resource)

Create the *_types.go file and controller.go files.

For more on resources and controllers see [What Is A Resource](../basics/what_is_a_resource.md) 
and [What Is A Controller](../basics/what_is_a_controller.md) 

{% sample lang="bash" %}
```bash
$ kubebuilder create api --group mygroup --version v1beta1 --kind MyKind
```
{% endmethod %}

{% method %}
## Run your manager locally against a Kubernetes cluster

Users may run the controller-manager binary locally against a Kubernetes cluster.  This will
install the APIs into the cluster and begin watching and reconciling the resources.

{% sample lang="bash" %}
```bash
# Create a minikube cluster
$ minikube start

# Install the CRDs into the cluster
$ make install

# Build and run the manager
$ make run
```
{% endmethod %}

{% method %}
## Create an instance

Create a new instance of your Resource.  Observe the manager logs printed to the console after creating the object.

{% sample lang="bash" %}
```bash
$ kubectl apply -f sample/<resource>.yaml
```
{% endmethod %}
