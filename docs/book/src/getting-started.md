# Getting started

This guide creates a sample project to show you how it works. This sample:

- Reconcile a Memcached CR - which represents an instance of a Memcached deployed/managed on cluster
- Create a Deployment with the Memcached image
- Not allow more instances than the size defined in the CR which is applied
- Update the Memcached CR status

<aside class="note" role="note">
<p class="note-title">Why Operators?</p>

By following the [Operator Pattern][k8s-operator-pattern], it’s possible not only to provide all expected resources
but also to manage them dynamically, programmatically, and at execution time. To illustrate this idea, imagine if
someone accidentally changed a configuration or removed a resource by mistake; in this case, the operator could fix it
without any human intervention.

</aside>

<aside class="note" role="note">
<p class="note-title">Following Along vs Jumping Ahead</p>

Note that most of this tutorial is generated from literate Go files that
form a runnable project, and live in the book source directory:
[docs/book/src/getting-started/testdata/project][tutorial-source].

</aside>

## Create a project

First, create and navigate into a directory for your project. Then, initialize it using `kubebuilder`:

```shell
mkdir $GOPATH/memcached-operator
cd $GOPATH/memcached-operator
kubebuilder init --domain=example.com
```

<aside class="note" role="note">
<p class="note-title">Developing in $GOPATH</p>

If your project is initialized within [`GOPATH`][GOPATH-golang-docs], the implicitly called `go mod init` will interpolate the module path for you.
Otherwise `--repo=<module path>` must be set.

Read the [Go modules blogpost][go-modules-blogpost] if unfamiliar with the module system.

</aside>

## Create the Memcached API (CRD)

Next, create the API which is responsible for deploying and
managing Memcached(s) instances on the cluster.

```shell
kubebuilder create api --group cache --version v1alpha1 --kind Memcached
```

### Understanding APIs

This command's primary aim is to produce the Custom Resource (CR) and Custom Resource Definition (CRD) for the Memcached Kind.
It creates the API with the group `cache.example.com` and version `v1alpha1`, uniquely identifying the new CRD of the Memcached Kind.
By leveraging the Kubebuilder tool, you can define your APIs and objects representing your solutions for these platforms.

While this example adds only one Kind of resource, you can have as many `Groups` and `Kinds` as necessary.
To make it easier to understand, think of CRDs as the definition of our custom Objects, while CRs are instances of them.

<aside class="note" role="note">
<p class="note-title"> Please ensure that you check </p>

[Groups and Versions and Kinds, oh my!][group-kind-oh-my].

</aside>

### Defining our API

#### Defining the specs

Now, define the values that each instance of your Memcached resource on the cluster can assume. In this example,
the configuration allows setting the number of instances with the following:

```go
type MemcachedSpec struct {
	...
	// +kubebuilder:validation:Minimum=0
	// +required
	Size *int32 `json:"size,omitempty"`
}
```

#### Creating status definitions

The controller also needs to track the status of operations done to manage the Memcached CR(s).
This allows verification of the Custom Resource's description of your API and determines if everything
occurred successfully or if any errors were encountered,
similar to how you would with any resource from the Kubernetes API.

```go
// MemcachedStatus defines the observed state of Memcached
type MemcachedStatus struct {
    // +listType=map
    // +listMapKey=type
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

<aside class="note" role="note">
<p class="note-title"> Status Conditions </p>

Kubernetes has established conventions, and because of this, use
Status Conditions here. Your custom APIs and controllers should behave
like Kubernetes resources and their controllers, following these standards
to ensure a consistent and intuitive experience.

Please ensure that you review: [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)
</aside>


#### Markers and validations

Furthermore, validate the values added in your CustomResource
to ensure that those are valid. To achieve this, use [markers][markers],
such as `+kubebuilder:validation:Minimum=1`.

Now, see our example fully completed.

{{#literatego ./getting-started/testdata/project/api/v1alpha1/memcached_types.go}}

#### Generating manifests with the specs and validations

To generate all required files:

1. Run `make generate` to create the DeepCopy implementations in `api/v1alpha1/zz_generated.deepcopy.go`.

2. Then, run `make manifests` to generate the CRD manifests under `config/crd/bases` and a sample for it under `config/samples`.

Both commands use [controller-gen][controller-gen] with different flags for code and manifest generation, respectively.

<details><summary><code>config/crd/bases/cache.example.com_memcacheds.yaml</code>: Our Memcached CRD</summary>

```yaml
{{#include ./getting-started/testdata/project/config/crd/bases/cache.example.com_memcacheds.yaml}}
```

</details>

#### Sample of custom resources

The manifests located under the `config/samples` directory serve as examples of Custom Resources that can be applied to the cluster.
In this particular example, by applying the given resource to the cluster, we would generate
a Deployment with a single instance size (see `size: 1`).

```yaml
{{#include ./getting-started/testdata/project/config/samples/cache_v1alpha1_memcached.yaml}}
```

### Reconciliation process

In a simplified way, Kubernetes works by allowing you to declare the desired state of your system, and then its controllers continuously observe the cluster and take actions to ensure that the actual state matches the desired state. For your custom APIs and controllers, the process is similar. Remember, you are extending Kubernetes' behaviors and its APIs to fit your specific needs.

In our controller, we implement a reconciliation process.

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

<aside class="note" role="note">
<p class="note-title"> Return Options </p>

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

When the sample Custom Resource (CR) is applied to the cluster (i.e. `kubectl apply -f config/sample/cache_v1alpha1_memcached.yaml`),
ensure that a Deployment is created for the Memcached image and that it matches the number of replicas defined in the CR.

To achieve this, first implement an operation that checks whether the Deployment for the Memcached instance already exists on the cluster.
If it does not, the controller creates the Deployment accordingly. Therefore, our reconciliation process must include an operation to ensure that
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

Next, note that the `deploymentForMemcached()` function needs to define and return the Deployment that should be
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

Additionally, implement a mechanism to verify that the number of Memcached replicas
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

### Diving into the controller implementation

#### Setting manager to watching resources

The whole idea is to be Watching the resources that matter for the controller.
When a resource that the controller is interested in changes, the Watch triggers the controller's
reconciliation loop, ensuring that the actual state of the resource matches the desired state
as defined in the controller's logic.

Notice how the Manager is configured to monitor events such as the creation, update, or deletion of a Custom Resource (CR) of the Memcached kind,
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
        // owned and managed by this controller, it triggers reconciliation, ensuring that the cluster
        // state aligns with the desired state.
		Owns(&appsv1.Deployment{}).
		Complete(r)
    }
```

#### But, how does the manager know which resources are owned by it?

The Controller should not watch any Deployment on the cluster and trigger the
reconciliation loop. Instead, trigger reconciliation only when the specific
Deployment running the Memcached instance is changed. For example,
if someone accidentally deletes the Deployment or changes the number of replicas, trigger
the reconciliation to ensure that it returns to the desired state.

The Manager knows which Deployment to observe because the `ownerRef` (Owner Reference) is set:

```go
if err := ctrl.SetControllerReference(memcached, dep, r.Scheme); err != nil {
    return nil, err
}
```

<aside class="note" role="note">

<p class="note-title"><code>ownerRef</code> and Cascading Events</p>

The ownerRef is crucial not only for allowing the controller to observe changes on the specific resource but also because,
if you delete the Memcached Custom Resource (CR) from the cluster, all resources owned by it are automatically
deleted as well, in a cascading event.

This ensures that when the parent resource (Memcached CR) is removed, all associated resources
(like Deployments, Services, etc.) are also cleaned up, maintaining
a tidy and consistent cluster state.

For more information, see the Kubernetes documentation on [Owners and Dependents](https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/).

</aside>

### Granting permissions

It's important to ensure that the Controller has the necessary permissions(i.e. to create, get, update, and list)
the resources it manages.

The [RBAC permissions][k8s-rbac] are now configured via [RBAC markers][rbac-markers], which are used to generate and update the
manifest files present in `config/rbac/`. These markers can be found (and should be defined) on the `Reconcile()` method of each controller, see
how it is implemented in our example:

```go
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
```

After making changes to the controller, run the make manifests command. This will prompt [controller-gen][controller-gen]
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

### Use Kubebuilder plugins to scaffold additional options

Now that you have a better understanding of how to create your own API and controller,
let’s scaffold in this project the plugin [`autoupdate.kubebuilder.io/v1-alpha`][autoupdate-plugin]
so that your project can be kept up to date with the latest Kubebuilder releases scaffolding changes
and consequently adopt improvements from the ecosystem.

```shell
kubebuilder edit --plugins="autoupdate/v1-alpha"
```

Inspect the file `.github/workflows/auto-update.yml` to see how it works.

### Checking the project running in the cluster

At this point you can check the steps to validate the project
on the cluster by looking the steps defined in the Quick Start,
see: [Run It On the Cluster](./quick-start#run-it-on-the-cluster)

## Next steps

- To delve deeper into developing your solution, consider going through the [CronJob Tutorial][cronjob-tutorial]
- For insights on optimizing your approach, refer to the [Best Practices][best-practices] documentation.

<aside class="note" role="note">
<p class="note-title"> Using Deploy Image plugin to generate APIs and source code </p>

Now that you have a better understanding, you might want to check out the [Deploy Image][deploy-image] Plugin.
This plugin allows users to scaffold APIs/Controllers to deploy and manage an Operand (image) on the cluster.
It provides scaffolds similar to the ones in this guide, along with additional features such as tests
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
[cronjob-tutorial]: ./cronjob-tutorial/cronjob-tutorial.md
[deploy-image]: ./plugins/available/deploy-image-plugin-v1-alpha.md
[GOPATH-golang-docs]: https://golang.org/doc/code.html#GOPATH
[go-modules-blogpost]: https://blog.golang.org/using-go-modules
[autoupdate-plugin]: ./plugins/available/autoupdate-v1-alpha.md
[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project