# Quick Start

This Quick Start guide will cover:

- [Creating a project](#create-a-project)
- [Creating an API](#create-an-api)
- [Running locally](#test-it-out)
- [Running in-cluster](#run-it-on-the-cluster)

## Prerequisites

- [go](https://golang.org/dl/) version v1.15+ and < 1.16.
- [docker](https://docs.docker.com/install/) version 17.03+.
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

<aside class="note">
<h1>Versions and Supportability</h1>

Projects created by Kubebuilder contain a Makefile that will install tools at versions defined at creation time. Those tools are:
- [kustomize](https://kubernetes-sigs.github.io/kustomize/)
- [controller-gen](https://github.com/kubernetes-sigs/controller-tools)

The versions which are defined in the `Makefile` and `go.mod` files are the versions tested and therefore is recommend to use the specified versions.

</aside>

## Installation

Install [kubebuilder](https://sigs.k8s.io/kubebuilder):

```bash
# download kubebuilder and install locally.
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
```

<aside class="note">
<h1>Using master branch</h1>

You can work with a master snapshot by installing from `https://go.kubebuilder.io/dl/master/$(go env GOOS)/$(go env GOARCH)`.

</aside>

<aside class="note">
<h1>Enabling shell autocompletion</h1>

Kubebuilder provides autocompletion support for Bash and Zsh via the command `kubebuilder completion <bash|zsh>`, which can save you a lot of typing. For further information see the [completion](./reference/completion.md) document.

</aside>

## Create a Project

Create a directory, and then run the init command inside of it to initialize a new project. Follows an example.

```bash
mkdir -p ~/projects/guestbook
cd ~/projects/guestbook
kubebuilder init --domain my.domain --repo my.domain/guestbook
```

<aside class="note">
<h1>Developing in $GOPATH</h1>

If your project is initialized within [`GOPATH`][GOPATH-golang-docs], the implicitly called `go mod init` will interpolate the module path for you.
Otherwise `--repo=<module path>` must be set.

Read the [Go modules blogpost][go-modules-blogpost] if unfamiliar with the module system.

</aside>


## Create an API

Run the following command to create a new API (group/version) as `webapp/v1` and the new Kind(CRD) `Guestbook` on it:

```bash
kubebuilder create api --group webapp --version v1 --kind Guestbook
```

<aside class="note">
<h1>Press Options</h1>

If you press `y` for Create Resource [y/n] and for Create Controller [y/n] then this will create the files `api/v1/guestbook_types.go` where the API is defined
and the `controllers/guestbook_controller.go` where the reconciliation business logic is implemented for this Kind(CRD).

</aside>


**OPTIONAL:** Edit the API definition and the reconciliation business
logic. For more info see [Designing an API](/cronjob-tutorial/api-design.md) and [What's in
a Controller](cronjob-tutorial/controller-overview.md).

<details><summary>Click here to see an example. `(api/v1/guestbook_types.go)` </summary>
<p>

```go
// GuestbookSpec defines the desired state of Guestbook
type GuestbookSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Quantity of instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	Size int32 `json:"size"`

	// Name of the ConfigMap for GuestbookSpec's configuration
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=1
	ConfigMapName string `json:"configMapName"`

	// +kubebuilder:validation:Enum=Phone;Address;Name
	Type string `json:"alias,omitempty"`
}

// GuestbookStatus defines the observed state of Guestbook
type GuestbookStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PodName of the active Guestbook node.
	Active string `json:"active"`

	// PodNames of the standby Guestbook nodes.
	Standby []string `json:"standby"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// Guestbook is the Schema for the guestbooks API
type Guestbook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GuestbookSpec   `json:"spec,omitempty"`
	Status GuestbookStatus `json:"status,omitempty"`
}
```

</p>
</details>


## Test It Out

You'll need a Kubernetes cluster to run against.  You can use
[KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or
run against a remote cluster.

<aside class="note">
<h1>Context Used</h1>

Your controller will automatically use the current context in your
kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

</aside>

Install the CRDs into the cluster:
```bash
make install
```

Run your controller (this will run in the foreground, so switch to a new
terminal if you want to leave it running):
```bash
make run
```

## Install Instances of Custom Resources

If you pressed `y` for Create Resource [y/n] then you created an (CR)Custom Resource for your (CRD)Custom Resource Definition in your samples (make sure to edit them first if you've changed the
API definition):

```bash
kubectl apply -f config/samples/
```

## Run It On the Cluster

Build and push your image to the location specified by `IMG`:

```bash
make docker-build docker-push IMG=<some-registry>/<project-name>:tag
```

Deploy the controller to the cluster with image specified by `IMG`:

```bash
make deploy IMG=<some-registry>/<project-name>:tag
```

<aside class="note">
<h1>RBAC errors</h1>

If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin. See [Prerequisites for using Kubernetes RBAC on GKE cluster v1.11.x and older][pre-rbc-gke] which may be your case.  

</aside>

## Uninstall CRDs

To delete your CRDs from the cluster:

```bash
make uninstall
```

## Undeploy controller

UnDeploy the controller to the cluster:

```bash
make undeploy
```

## Next Step

Now, see the [architecture concept diagram][architecture-concept-diagram] for a better overview and follow up the [CronJob tutorial][cronjob-tutorial] to better understand how it works by developing a demo example project.

[pre-rbc-gke]: https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control#iam-rolebinding-bootstrap
[cronjob-tutorial]: https://book.kubebuilder.io/cronjob-tutorial/cronjob-tutorial.html
[GOPATH-golang-docs]: https://golang.org/doc/code.html#GOPATH
[go-modules-blogpost]: https://blog.golang.org/using-go-modules
[envtest]: https://book.kubebuilder.io/reference/testing/envtest.html
[architecture-concept-diagram]: architecture.md
