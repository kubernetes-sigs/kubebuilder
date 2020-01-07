# Quick Start

This Quick Start guide will cover:

- [Creating a project](#create-a-project)
- [Creating an API](#create-an-api)
- [Running locally](#test-it-out)
- [Running in-cluster](#run-it-on-the-cluster)

## Prerequisites

- [go](https://golang.org/dl/) version v1.13+.
- [docker](https://docs.docker.com/install/) version 17.03+.
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) version v1.11.3+.
- [kustomize](https://sigs.k8s.io/kustomize/docs/INSTALL.md) v3.1.0+
- Access to a Kubernetes v1.11.3+ cluster.

## Installation

Install [kubebuilder](https://sigs.k8s.io/kubebuilder):

```bash
os=$(go env GOOS)
arch=$(go env GOARCH)

# download kubebuilder and extract it to tmp
curl -L https://go.kubebuilder.io/dl/2.2.0/${os}/${arch} | tar -xz -C /tmp/

# move to a long-term location and put it on your path
# (you'll need to set the KUBEBUILDER_ASSETS env var if you put it somewhere else)
sudo mv /tmp/kubebuilder_2.2.0_${os}_${arch} /usr/local/kubebuilder
export PATH=$PATH:/usr/local/kubebuilder/bin
```

<aside class="note">
<h1>Using master branch</h1>

Also, you can install a master snapshot from `https://go.kubebuilder.io/dl/latest/${os}/${arch}`.

</aside>

## Create a Project

Create a directory, and then run the init command inside of it to initialize a new project. Follows an example.

```bash
mkdir $GOPATH/src/example
cd $GOPATH/src/example
kubebuilder init --domain my.domain
```

<aside class="note">
<h1>Not in $GOPATH</h1>

If you're not in `GOPATH`, you'll need to run `go mod init <modulename>` in order to tell kubebuilder and Go the base import path of your module. 

For a further understanding of `GOPATH` see [The GOPATH environment variable][GOPATH-golang-docs] in the [How to Write Go Code][how-to-write-go-code-golang-docs] golang page doc.   

</aside>

<aside class="note">
<h1>Go package issues</h1>

Ensure that you activate the module support by running `$ export GO111MODULE=on` 
to solve issues as `cannot find package ... (from $GOROOT)`.

</aside>


## Create an API

Run the following command to create a new API (group/version) as `webapp/v1` and the new Kind(CRD) `Guestbook` on it:

```bash
kubebuilder create api --group webapp --version v1 --kind Guestbook
```

<aside class="note">
<h1>Press Options</h1>

If you press `y` for Create Resource [y/n] and for Create Controller [y/n] then this will create the files `api/v1/guestbook_types.go` where the API is defined 
and the `controller/guestbook_controller.go` where the reconciliation business logic is implemented for this Kind(CRD).

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

## Next Step 

Now, follow up the [CronJob tutorial][cronjob-tutorial] to better understand how it works by developing a demo example project. 

[pre-rbc-gke]:https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control#iam-rolebinding-bootstrap
[cronjob-tutorial]: https://book.kubebuilder.io/cronjob-tutorial/cronjob-tutorial.html
[GOPATH-golang-docs]: https://golang.org/doc/code.html#GOPATH
[how-to-write-go-code-golang-docs]: https://golang.org/doc/code.html 

