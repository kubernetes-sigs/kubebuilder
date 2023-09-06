# Getting Started

## Overview

By following the [Operator Pattern][k8s-operator-pattern], it’s possible not only to provide all expected resources 
but also to manage them dynamically, programmatically, and at execution time. To illustrate this idea, imagine if 
someone accidentally changed a configuration or removed a resource by mistake; in this case, the operator could fix it
without any human intervention.

## Sample Project

We will create a sample project to let you know how it works. This sample will:

- Reconcile a Memcached CR - which represents an instance of a Memcached deployed/managed on cluster
- Create a Deployment with the Memcached image
- Not allow more instances than the size defined in the CR which will be applied
- Update the Memcached CR status

Following the steps.

## Create a project

First, create and navigate into a directory for your project. Then, initialize it using `kubebuilder`:

```shell
mkdir $GOPATH/memcached-operator
cd $GOPATH/memcached-operator
kubebuilder init --domain=example.com
```

## Create the Memcached API (CRD):

Next, we'll create a new API responsible for deploying and managing our Memcached solution. In this instance, we will utilize the [Deploy Image Plugin][deploy-image] to get a comprehensive code implementation for our solution.

```
kubebuilder create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:1.4.36-alpine --image-container-command="memcached,-m=64,-o,modern,-v" --image-container-port="11211" --run-as-user="1001" --plugins="deploy-image/v1-alpha" --make=false
```

### Understanding APIs

This command's primary aim is to produce the Custom Resource (CR) and Custom Resource Definition (CRD) for the Memcached Kind. It creates the API with the group `cache.example.com` and version `v1alpha1`, uniquely identifying the new CRD of the Memcached Kind. By leveraging the Kubebuilder tool, we can define our APIs and objects representing our solutions for these platforms. While we've added only one Kind of resource in this example, you can have as many `Groups` and `Kinds` as necessary. Simply put, think of CRDs as the definition of our custom Objects, while CRs are instances of them.

<aside class="note">
<h1>Getting a better idea</h1>

Consider a typical scenario where the objective is to run an application and its database on a Kubernetes platform. In this context, one object might represent the Frontend App, while another denotes the backend Data Base. If we define one CRD for the App and another for the DB, we uphold essential concepts like encapsulation, the single responsibility principle, and cohesion. Breaching these principles might lead to complications, making extension, reuse, or maintenance challenging.

In essence, the App CRD and the DB CRD will have their controller. Let's say, for instance, that the application requires a Deployment and Service to run. In this example, the App’s Controller will cater to these needs. Similarly, the DB’s controller will manage the business logic of its items.

Therefore, for each CRD, there should be one distinct controller, adhering to the design outlined by the [controller-runtime][controller-runtime]. For further information see [Groups and Versions and Kinds, oh my!][group-kind-oh-my].

</aside>

### Define your API

In this example, observe that the Memcached Kind (CRD) possesses certain specifications. These were scaffolded by the Deploy Image plugin, building upon the default scaffold for management purposes:

#### Status and Specs

The `MemcachedSpec` section is where we encapsulate all the available specifications and configurations for our Custom Resource (CR). Furthermore, it's worth noting that we employ Status Conditions. This ensures proficient management of the Memcached CR. When any change transpires, these conditions equip us with the necessary data to discern the current status of this resource within the Kubernetes cluster. This is akin to the status insights we obtain for a Deployment resource.

From: `api/v1alpha1/memcached_types.go`

```go
// MemcachedSpec defines the desired state of Memcached
type MemcachedSpec struct {
// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
// Important: Run "make" to regenerate code after modifying this file

// Size defines the number of Memcached instances
// The following markers will use OpenAPI v3 schema to validate the value
// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=3
// +kubebuilder:validation:ExclusiveMaximum=false
Size int32 `json:"size,omitempty"`

// Port defines the port that will be used to init the container with the image
ContainerPort int32 `json:"containerPort,omitempty"`
}

// MemcachedStatus defines the observed state of Memcached
type MemcachedStatus struct {
// Represents the observations of a Memcached's current state.
// Memcached.status.conditions.type are: "Available", "Progressing", and "Degraded"
// Memcached.status.conditions.status are one of True, False, Unknown.
// Memcached.status.conditions.reason the value should be a CamelCase string and producers of specific
// condition types may define expected values and meanings for this field, and whether the values
// are considered a guaranteed API.
// Memcached.status.conditions.Message is a human readable message indicating details about the transition.
// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}
```

Thus, when we introduce new specifications to this file and execute the `make generate` command, we utilize [controller-gen][controller-gen] to generate the CRD manifest, which is located under the `config/crds` directory.

#### Markers and validations

Moreover, it's important to note that we're employing `markers`, such as `+kubebuilder:validation:Minimum=1`. These markers help in defining validations and criteria, ensuring that data provided by users—when they create or edit a Custom Resource for the Memcached Kind—is properly validated. For a comprehensive list and details of available markers, refer [here][markers].
Observe the validation schema within the CRD; this schema ensures that the Kubernetes API properly validates the Custom Resources (CRs) that are applied:

From: `config/crd/bases/example.com.testproject.org_memcacheds.yaml`
```yaml
            description: MemcachedSpec defines the desired state of Memcached
            properties:
              containerPort:
                description: Port defines the port that will be used to init the container
                  with the image
                format: int32
                type: integer
              size:
                description: 'Size defines the number of Memcached instances The following
                  markers will use OpenAPI v3 schema to validate the value More info:
                  https://book.kubebuilder.io/reference/markers/crd-validation.html'
                format: int32
                maximum: 3 ## See here from the marker +kubebuilder:validation:Maximum=3
                minimum: 1 ## See here from the marker +kubebuilder:validation:Minimum=1
                type: integer
            type: object

```

#### Sample of Custom Resources

The manifests located under the "config/samples" directory serve as examples of Custom Resources that can be applied to the cluster.
In this particular example, by applying the given resource to the cluster, we would generate a Deployment with a single instance size (see `size: 1`).

From: `config/samples/example.com_v1alpha1_memcached.yaml`

```shell
apiVersion: example.com.testproject.org/v1alpha1
kind: Memcached
metadata:
  name: memcached-sample
spec:
  # TODO(user): edit the following value to ensure the number
  # of Pods/Instances your Operand must have on cluster
  size: 1
# TODO(user): edit the following value to ensure the container has the right port to be initialized
  containerPort: 11211
```

### Reconciliation Process

The reconciliation function plays a pivotal role in ensuring synchronization between resources and their specifications based on the business logic embedded within them. Essentially, it operates like a loop, continuously checking conditions and performing actions until all conditions align with its implementation. Here's a pseudo-code to illustrate this:

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

#### Return Options

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

#### In the context of our example

When a Custom Resource is applied to the cluster, there's a designated controller to manage the Memcached Kind. You can check its reconciliation implemented:

From `testdata/project-v4-with-deploy-image/internal/controller/memcached_controller.go`:

```go
func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
log := log.FromContext(ctx)

// Fetch the Memcached instance
// The purpose is to check if the Custom Resource for the Kind Memcached
// is applied on the cluster if not we return nil to stop the reconciliation
memcached := &examplecomv1alpha1.Memcached{}
err := r.Get(ctx, req.NamespacedName, memcached)
if err != nil {
if apierrors.IsNotFound(err) {
// If the custom resource is not found then, it usually means that it was deleted or not created
// In this way, we will stop the reconciliation
log.Info("memcached resource not found. Ignoring since object must be deleted")
return ctrl.Result{}, nil
}
// Error reading the object - requeue the request.
log.Error(err, "Failed to get memcached")
return ctrl.Result{}, err
}

// Let's just set the status as Unknown when no status are available
if memcached.Status.Conditions == nil || len(memcached.Status.Conditions) == 0 {
meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeAvailableMemcached, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
if err = r.Status().Update(ctx, memcached); err != nil {
log.Error(err, "Failed to update Memcached status")
return ctrl.Result{}, err
}

// Let's re-fetch the memcached Custom Resource after update the status
// so that we have the latest state of the resource on the cluster and we will avoid
// raise the issue "the object has been modified, please apply
// your changes to the latest version and try again" which would re-trigger the reconciliation
// if we try to update it again in the following operations
if err := r.Get(ctx, req.NamespacedName, memcached); err != nil {
log.Error(err, "Failed to re-fetch memcached")
return ctrl.Result{}, err
}
}

// Let's add a finalizer. Then, we can define some operations which should
// occurs before the custom resource to be deleted.
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
if !controllerutil.ContainsFinalizer(memcached, memcachedFinalizer) {
log.Info("Adding Finalizer for Memcached")
if ok := controllerutil.AddFinalizer(memcached, memcachedFinalizer); !ok {
log.Error(err, "Failed to add finalizer into the custom resource")
return ctrl.Result{Requeue: true}, nil
}

if err = r.Update(ctx, memcached); err != nil {
log.Error(err, "Failed to update custom resource to add finalizer")
return ctrl.Result{}, err
}
}

// Check if the Memcached instance is marked to be deleted, which is
// indicated by the deletion timestamp being set.
isMemcachedMarkedToBeDeleted := memcached.GetDeletionTimestamp() != nil
if isMemcachedMarkedToBeDeleted {
if controllerutil.ContainsFinalizer(memcached, memcachedFinalizer) {
log.Info("Performing Finalizer Operations for Memcached before delete CR")

// Let's add here an status "Downgrade" to define that this resource begin its process to be terminated.
meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeDegradedMemcached,
Status: metav1.ConditionUnknown, Reason: "Finalizing",
Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", memcached.Name)})

if err := r.Status().Update(ctx, memcached); err != nil {
log.Error(err, "Failed to update Memcached status")
return ctrl.Result{}, err
}

// Perform all operations required before remove the finalizer and allow
// the Kubernetes API to remove the custom resource.
r.doFinalizerOperationsForMemcached(memcached)

// TODO(user): If you add operations to the doFinalizerOperationsForMemcached method
// then you need to ensure that all worked fine before deleting and updating the Downgrade status
// otherwise, you should requeue here.

// Re-fetch the memcached Custom Resource before update the status
// so that we have the latest state of the resource on the cluster and we will avoid
// raise the issue "the object has been modified, please apply
// your changes to the latest version and try again" which would re-trigger the reconciliation
if err := r.Get(ctx, req.NamespacedName, memcached); err != nil {
log.Error(err, "Failed to re-fetch memcached")
return ctrl.Result{}, err
}

meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeDegradedMemcached,
Status: metav1.ConditionTrue, Reason: "Finalizing",
Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", memcached.Name)})

if err := r.Status().Update(ctx, memcached); err != nil {
log.Error(err, "Failed to update Memcached status")
return ctrl.Result{}, err
}

log.Info("Removing Finalizer for Memcached after successfully perform the operations")
if ok := controllerutil.RemoveFinalizer(memcached, memcachedFinalizer); !ok {
log.Error(err, "Failed to remove finalizer for Memcached")
return ctrl.Result{Requeue: true}, nil
}

if err := r.Update(ctx, memcached); err != nil {
log.Error(err, "Failed to remove finalizer for Memcached")
return ctrl.Result{}, err
}
}
return ctrl.Result{}, nil
}

// Check if the deployment already exists, if not create a new one
found := &appsv1.Deployment{}
err = r.Get(ctx, types.NamespacedName{Name: memcached.Name, Namespace: memcached.Namespace}, found)
if err != nil && apierrors.IsNotFound(err) {
// Define a new deployment
dep, err := r.deploymentForMemcached(memcached)
if err != nil {
log.Error(err, "Failed to define new Deployment resource for Memcached")

// The following implementation will update the status
meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeAvailableMemcached,
Status: metav1.ConditionFalse, Reason: "Reconciling",
Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", memcached.Name, err)})

if err := r.Status().Update(ctx, memcached); err != nil {
log.Error(err, "Failed to update Memcached status")
return ctrl.Result{}, err
}

return ctrl.Result{}, err
}

log.Info("Creating a new Deployment",
"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
if err = r.Create(ctx, dep); err != nil {
log.Error(err, "Failed to create new Deployment",
"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
return ctrl.Result{}, err
}

// Deployment created successfully
// We will requeue the reconciliation so that we can ensure the state
// and move forward for the next operations
return ctrl.Result{RequeueAfter: time.Minute}, nil
} else if err != nil {
log.Error(err, "Failed to get Deployment")
// Let's return the error for the reconciliation be re-trigged again
return ctrl.Result{}, err
}

// The CRD API is defining that the Memcached type, have a MemcachedSpec.Size field
// to set the quantity of Deployment instances is the desired state on the cluster.
// Therefore, the following code will ensure the Deployment size is the same as defined
// via the Size spec of the Custom Resource which we are reconciling.
size := memcached.Spec.Size
if *found.Spec.Replicas != size {
found.Spec.Replicas = &size
if err = r.Update(ctx, found); err != nil {
log.Error(err, "Failed to update Deployment",
"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

// Re-fetch the memcached Custom Resource before update the status
// so that we have the latest state of the resource on the cluster and we will avoid
// raise the issue "the object has been modified, please apply
// your changes to the latest version and try again" which would re-trigger the reconciliation
if err := r.Get(ctx, req.NamespacedName, memcached); err != nil {
log.Error(err, "Failed to re-fetch memcached")
return ctrl.Result{}, err
}

// The following implementation will update the status
meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeAvailableMemcached,
Status: metav1.ConditionFalse, Reason: "Resizing",
Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", memcached.Name, err)})

if err := r.Status().Update(ctx, memcached); err != nil {
log.Error(err, "Failed to update Memcached status")
return ctrl.Result{}, err
}

return ctrl.Result{}, err
}

// Now, that we update the size we want to requeue the reconciliation
// so that we can ensure that we have the latest state of the resource before
// update. Also, it will help ensure the desired state on the cluster
return ctrl.Result{Requeue: true}, nil
}

// The following implementation will update the status
meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeAvailableMemcached,
Status: metav1.ConditionTrue, Reason: "Reconciling",
Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", memcached.Name, size)})

if err := r.Status().Update(ctx, memcached); err != nil {
log.Error(err, "Failed to update Memcached status")
return ctrl.Result{}, err
}

return ctrl.Result{}, nil
}
```

#### Observing changes on cluster

This controller is persistently observant, monitoring any events associated with this Kind. As a result, pertinent changes
instantly set off the controller's reconciliation process. It's worth noting that we have implemented the `watches` feature. [(More info)][watches].
This allows us to monitor events related to creating, updating, or deleting a Custom Resource of the Memcached kind, as well as the Deployment
which is orchestrated and owned by its respective controller. Observe:

```go
// SetupWithManager sets up the controller with the Manager.
// Note that the Deployment will be also watched in order to ensure its
// desirable state on the cluster
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
return ctrl.NewControllerManagedBy(mgr).
For(&examplecomv1alpha1.Memcached{}). ## Create watches for the Memcached Kind
Owns(&appsv1.Deployment{}). ## Create watches for the Deployment which has its controller owned reference
Complete(r)
}
```

<aside class="note">
<h1>Set the ownerRef for the Deployment</h1>

See that when we create the Deployment to run the Memcached image we are setting the reference:

```go
// Set the ownerRef for the Deployment
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
if err := ctrl.SetControllerReference(memcached, dep, r.Scheme); err != nil {
return nil, err
}

```

</aside>

### Setting the RBAC permissions

The [RBAC permissions][k8s-rbac] are now configured via [RBAC markers][rbac-markers], which are used to generate and update the
manifest files present in `config/rbac/`. These markers can be found (and should be defined) on the `Reconcile()` method of each controller, see
how it is implemented in our example:

```go
//+kubebuilder:rbac:groups=example.com.testproject.org,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.com.testproject.org,resources=memcacheds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.com.testproject.org,resources=memcacheds/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

```

It's important to highlight that if you wish to add or modify RBAC rules, you can do so by updating or adding the respective markers in the controller.
After making the necessary changes, run the `make generate` command. This will prompt [controller-gen][controller-gen] to refresh the files located under `config/rbac`.

<aside class="note">
<h1>RBAC generate under config/rbac</h1>

For each Kind generate Kubebuilder will either scaffold rules with view and edit permissions. (i.e. `memcached_editor_role.yaml` and `memcached_viewer_role.yaml`)
Those rules are not applied on the cluster when you deploy your solution with `make deploy IMG=myregistery/example:1.0.0`.
Those rules are aimed to help our system admins to allow them grant permissions to group of users.

</aside>

### Manager (main.go)

The [Manager][manager] plays a crucial role in overseeing Controllers, which in turn enable operations on the cluster side.
If you inspect the `cmd/main.go` file, you'll come across the following:

```go
...
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
Scheme:                 scheme,
Metrics:                metricsserver.Options{BindAddress: metricsAddr},
HealthProbeBindAddress: probeAddr,
LeaderElection:         enableLeaderElection,
LeaderElectionID:       "1836d577.testproject.org",
// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
// when the Manager ends. This requires the binary to immediately end when the
// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
// speeds up voluntary leader transitions as the new leader don't have to wait
// LeaseDuration time first.
//
// In the default scaffold provided, the program ends immediately after
// the manager stops, so would be fine to enable this option. However,
// if you are doing or is intended to do any operation such as perform cleanups
// after the manager stops then its usage might be unsafe.
// LeaderElectionReleaseOnCancel: true,
})
if err != nil {
setupLog.Error(err, "unable to start manager")
os.Exit(1)
}
```

The code snippet above outlines the configuration [options][options-manager] for the Manager. While we won't be altering this in our current example,
it's crucial to understand its location and the initialization process of your operator-based image. The Manager is responsible for overseeing the controllers
that are produced for your operator's APIs.

### Checking the Project running in the cluster

At this point, you can primarily execute the commands highlighted in the [quick-start][quick-start].
By executing `make build IMG=myregistry/example:1.0.0`, you'll build the image for your project. For testing purposes, it's recommended to publish this image to a
public registry. This ensures easy accessibility, eliminating the need for additional configurations. Once that's done, you can deploy the image
to the cluster using the `make deploy IMG=myregistry/example:1.0.0` command.

## Next Steps

- To delve deeper into developing your solution, consider going through the provided tutorials.
- For insights on optimizing your approach, refer to the [Best Practices][best-practices] documentation.

[k8s-operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[group-kind-oh-my]: ./cronjob-tutorial/gvks.md
[controller-gen]: ./reference/controller-gen.md
[markers]: ./reference/markers.md
[watches]: ./reference/watching-resources.md
[rbac-markers]: ./reference/markers/rbac.md
[k8s-rbac]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[manager]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager
[options-manager]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Options
[quick-start]: ./quick-start.md
[best-practices]: ./reference/good-practices.md