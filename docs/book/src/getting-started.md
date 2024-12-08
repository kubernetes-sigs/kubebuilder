# Getting Started

We will create a sample project to let you know how it works. This sample will:

- Reconcile a Memcached CR - which represents an instance of a Memcached deployed/managed on cluster
- Create a Deployment with the Memcached image
- Not allow more instances than the size defined in the CR which will be applied
- Update the Memcached CR status

<aside class="note">
<h1>Why Operators?</h1>

By following the [Operator Pattern][k8s-operator-pattern], it’s possible not only to provide all expected resources
but also to manage them dynamically, programmatically, and at execution time. To illustrate this idea, imagine if
someone accidentally changed a configuration or removed a resource by mistake; in this case, the operator could fix it
without any human intervention.

</aside>

<aside class="note">
<h1>Following Along vs Jumping Ahead</h1>

Note that most of this tutorial is generated from literate Go files that
form a runnable project, and live in the book source directory:
[docs/book/src/getting-started/testdata/project][tutorial-source].

[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project

</aside>

## Create a project

First, create and navigate into a directory for your project. Then, initialize it using `kubebuilder`:

```shell
mkdir $GOPATH/memcached-operator
cd $GOPATH/memcached-operator
kubebuilder init --domain=example.com
```

<aside class="note">
<h1>Developing in $GOPATH</h1>

If your project is initialized within [`GOPATH`][GOPATH-golang-docs], the implicitly called `go mod init` will interpolate the module path for you.
Otherwise `--repo=<module path>` must be set.

Read the [Go modules blogpost][go-modules-blogpost] if unfamiliar with the module system.

</aside>

## Create the Memcached API (CRD):

Next, we'll create the API which will be responsible for deploying and
managing Memcached(s) instances on the cluster.

```shell
kubebuilder create api --group cache --version v1alpha1 --kind Memcached
```

### Understanding APIs

This command's primary aim is to produce the Custom Resource (CR) and Custom Resource Definition (CRD) for the Memcached Kind.
It creates the API with the group `cache.example.com` and version `v1alpha1`, uniquely identifying the new CRD of the Memcached Kind.
By leveraging the Kubebuilder tool, we can define our APIs and objects representing our solutions for these platforms.

While we've added only one Kind of resource in this example, we can have as many `Groups` and `Kinds` as necessary.
To make it easier to understand, think of CRDs as the definition of our custom Objects, while CRs are instances of them.

<aside class="note">
<h1> Please ensure that you check </h1>

[Groups and Versions and Kinds, oh my!][group-kind-oh-my].

</aside>

### Defining our API

#### Defining the Specs

Now, we will define the values that each instance of your Memcached resource on the cluster can assume. In this example,
we will allow configuring the number of instances with the following:

```go
type MemcachedSpec struct {
	...
	Size int32 `json:"size,omitempty"`
}
```

#### Creating Status definitions

We also want to track the status of our Operations which will be done to manage the Memcached CR(s).
This allows us to verify the Custom Resource's description of our own API and determine if everything
occurred successfully or if any errors were encountered,
similar to how we do with any resource from the Kubernetes API.

```go
// MemcachedStatus defines the observed state of Memcached
type MemcachedStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}
```

<aside class="note">
<h1> Status Conditions </h1>

Kubernetes has established conventions, and because of this, we use
Status Conditions here. We want our custom APIs and controllers to behave
like Kubernetes resources and their controllers, following these standards
to ensure a consistent and intuitive experience.

Please ensure that you review: [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)
</aside>


#### Markers and validations

Furthermore, we want to validate the values added in our CustomResource
to ensure that those are valid. To achieve this, we will use [markers][markers],
such as `+kubebuilder:validation:Minimum=1`.

Now, see our example fully completed.

{{#literatego ./getting-started/testdata/project/api/v1alpha1/memcached_types.go}}

#### Generating manifests with the specs and validations

To generate all required files:

1. Run `make generate` to create the DeepCopy implementations in `api/v1alpha1/zz_generated.deepcopy.go`.

2. Then, run `make manifests` to generate the CRD manifests under `config/crd/bases` and a sample for it under `config/crd/samples`.

Both commands use [controller-gen][controller-gen] with different flags for code and manifest generation, respectively.

<details><summary><code>config/crd/bases/cache.example.com_memcacheds.yaml</code>: Our Memcached CRD</summary>

```yaml
{{#include ./getting-started/testdata/project/config/crd/bases/cache.example.com_memcacheds.yaml}}
```

</details>

#### Sample of Custom Resources

The manifests located under the `config/samples` directory serve as examples of Custom Resources that can be applied to the cluster.
In this particular example, by applying the given resource to the cluster, we would generate
a Deployment with a single instance size (see `size: 1`).

```yaml
{{#include ./getting-started/testdata/project/config/samples/cache_v1alpha1_memcached.yaml}}
```

### Reconciliation Process

In a simplified way, Kubernetes works by allowing us to declare the desired state of our system, and then its controllers continuously observe the cluster and take actions to ensure that the actual state matches the desired state. For our custom APIs and controllers, the process is similar. Remember, we are extending Kubernetes' behaviors and its APIs to fit our specific needs.

In our controller, we will implement a reconciliation process.

Essentially, the reconciliation process functions as a loop, continuously checking conditions and performing necessary actions until the desired state is achieved. This process will keep running until all conditions in the system align with the desired state defined in our implementation.

Here's a pseudo-code example to illustrate this:

```go
reconcile App {

  // Check if a Deployment for the app exists, if not, create one
  // If there's an error, then restart from the beginning of the reconcile
  if err != nil {
    return reconcile.Result{}, err
  }

  // Check if a Service for the app exists, if not, create one
  // If there's an error, then restart from the beginning of the reconcile
  if err != nil {
    return reconcile.Result{}, err
  }

  // Look for Database CR/CRD
  // Check the Database Deployment's replicas size
  // If deployment.replicas size doesn't match cr.size, then update it
  // Then, restart from the beginning of the reconcile. For example, by returning `reconcile.Result{Requeue: true}, nil`.
  if err != nil {
    return reconcile.Result{Requeue: true}, nil
  }
  ...

  // If at the end of the loop:
  // Everything was executed successfully, and the reconcile can stop
  return reconcile.Result{}, nil

}
```

<aside class="note">
<h1> Return Options </h1>

The following are a few possible return options to restart the Reconcile:

- With the error:

```go
return ctrl.Result{}, err
```
- Without an error:

```go
return ctrl.Result{Requeue: true}, nil
```

- Therefore, to stop the Reconcile, use:

```go
return ctrl.Result{}, nil
```

- Reconcile again after X time:

```go
return ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())}, nil
```

</aside>

#### In the context of our example

When our sample Custom Resource (CR) is applied to the cluster (i.e. `kubectl apply -f config/sample/cache_v1alpha1_memcached.yaml`),
we want to ensure that a Deployment is created for our Memcached image and that it matches the number of replicas defined in the CR.

To achieve this, we need to first implement an operation that checks whether the Deployment for our Memcached instance already exists on the cluster.
If it does not, the controller will create the Deployment accordingly. Therefore, our reconciliation process must include an operation to ensure that
this desired state is consistently maintained. This operation would involve:

```go
	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: memcached.Name, Namespace: memcached.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForMemcached()
		// Create the Deployment on the cluster
		if err = r.Create(ctx, dep); err != nil {
            log.Error(err, "Failed to create new Deployment",
            "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
            return ctrl.Result{}, err
        }
		...
	}
```

Next, note that the `deploymentForMemcached()` function will need to define and return the Deployment that should be
created on the cluster. This function should construct the Deployment object with the necessary
specifications, as demonstrated in the following example:

```go
    dep := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           "memcached:1.6.26-alpine3.19",
						Name:            "memcached",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 11211,
							Name:          "memcached",
						}},
						Command: []string{"memcached", "--memory-limit=64", "-o", "modern", "-v"},
					}},
				},
			},
		},
	}
```

Additionally, we need to implement a mechanism to verify that the number of Memcached replicas
on the cluster matches the desired count specified in the Custom Resource (CR). If there is a
discrepancy, the reconciliation must update the cluster to ensure consistency. This means that
whenever a CR of the Memcached Kind is created or updated on the cluster, the controller will
continuously reconcile the state until the actual number of replicas matches the desired count.
The following example illustrates this process:

```go
	...
	size := memcached.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		if err = r.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update Deployment",
				"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
            return ctrl.Result{}, err
        }
    ...
```

Now, you can review the complete controller responsible for managing Custom Resources of the
Memcached Kind. This controller ensures that the desired state is maintained in the cluster,
making sure that our Memcached instance continues running with the number of replicas specified
by the users.

<details><summary><code>internal/controller/memcached_controller.go</code>: Our Controller Implementation </summary>

```go
{{#include ./getting-started/testdata/project/internal/controller/memcached_controller.go}}
```
</details>

### Diving Into the Controller Implementation

#### Setting Manager to Watching Resources

The whole idea is to be Watching the resources that matter for the controller.
When a resource that the controller is interested in changes, the Watch triggers the controller's
reconciliation loop, ensuring that the actual state of the resource matches the desired state
as defined in the controller's logic.

Notice how we configured the Manager to monitor events such as the creation, update, or deletion of a Custom Resource (CR) of the Memcached kind,
as well as any changes to the Deployment that the controller manages and owns:

```go
// SetupWithManager sets up the controller with the Manager.
// The Deployment is also watched to ensure its
// desired state in the cluster.
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
		// Watch the Memcached Custom Resource and trigger reconciliation whenever it
		//is created, updated, or deleted
		For(&cachev1alpha1.Memcached{}).
		// Watch the Deployment managed by the Memcached controller. If any changes occur to the Deployment
        // owned and managed by this controller, it will trigger reconciliation, ensuring that the cluster
        // state aligns with the desired state.
		Owns(&appsv1.Deployment{}).
		Complete(r)
    }
```

#### But, How Does the Manager Know Which Resources Are Owned by It?

We do not want our Controller to watch any Deployment on the cluster and trigger our
reconciliation loop. Instead, we only want to trigger reconciliation when the specific
Deployment running our Memcached instance is changed. For example,
if someone accidentally deletes our Deployment or changes the number of replicas, we want
to trigger the reconciliation to ensure that it returns to the desired state.

The Manager knows which Deployment to observe because we set the `ownerRef` (Owner Reference):

```go
if err := ctrl.SetControllerReference(memcached, dep, r.Scheme); err != nil {
    return nil, err
}
```

<aside class="note">

<h1>`ownerRef` and  cascading event</h1>

The ownerRef is crucial not only for allowing us to observe changes on the specific resource but also because,
if we delete the Memcached Custom Resource (CR) from the cluster, we want all resources owned by it to be automatically
deleted as well, in a cascading event.

This ensures that when the parent resource (Memcached CR) is removed, all associated resources
(like Deployments, Services, etc.) are also cleaned up, maintaining
a tidy and consistent cluster state.

For more information, see the Kubernetes documentation on [Owners and Dependents](https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/).

</aside>

### Granting Permissions

It's important to ensure that the Controller has the necessary permissions(i.e. to create, get, update, and list)
the resources it manages.

The [RBAC permissions][k8s-rbac] are now configured via [RBAC markers][rbac-markers], which are used to generate and update the
manifest files present in `config/rbac/`. These markers can be found (and should be defined) on the `Reconcile()` method of each controller, see
how it is implemented in our example:

```go
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
```

After making changes to the controller, run the make generate command. This will prompt [controller-gen][controller-gen]
to refresh the files located under `config/rbac`.

<details><summary><code>config/rbac/role.yaml</code>: Our RBAC Role generated </summary>

```yaml
{{#include ./getting-started/testdata/project/config/rbac/role.yaml}}
```
</details>

### Manager (main.go)

The [Manager][manager] in the `cmd/main.go` file is responsible for managing the controllers in your application.

<details><summary><code>cmd/main.go</code>: Our main.go </summary>

```go
{{#include ./getting-started/testdata/project/cmd/main.go}}
```
</details>

### Checking the Project running in the cluster

At this point you can check the steps to validate the project
on the cluster by looking the steps defined in the Quick Start,
see: [Run It On the Cluster](./quick-start#run-it-on-the-cluster)

## Next Steps

- To delve deeper into developing your solution, consider going through the [CronJob Tutorial][cronjob-tutorial]
- For insights on optimizing your approach, refer to the [Best Practices][best-practices] documentation.

<aside class="note">
<h1> Using Deploy Image plugin to generate APIs and source code </h1>

Now that you have a better understanding, you might want to check out the [Deploy Image][deploy-image] Plugin.
This plugin allows users to scaffold APIs/Controllers to deploy and manage an Operand (image) on the cluster.
It will provide scaffolds similar to the ones in this guide, along with additional features such as tests
implemented for your controller.

</aside>

[k8s-operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[group-kind-oh-my]: ./cronjob-tutorial/gvks.md
[controller-gen]: ./reference/controller-gen.md
[markers]: ./reference/markers.md
[rbac-markers]: ./reference/markers/rbac.md
[k8s-rbac]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[manager]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager
[options-manager]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Options
[quick-start]: ./quick-start.md
[best-practices]: ./reference/good-practices.md
[cronjob-tutorial]: https://book.kubebuilder.io/cronjob-tutorial/cronjob-tutorial.html
[deploy-image]: ./plugins/available/deploy-image-plugin-v1-alpha.md
[GOPATH-golang-docs]: https://golang.org/doc/code.html#GOPATH
[go-modules-blogpost]: https://blog.golang.org/using-go-modules
