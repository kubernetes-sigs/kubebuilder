# Hello World

A new project may be scaffolded for a user by running `kubebuilder init` and then scaffolding a
new API with `kubebuilder create api`. More on this topic in
[Project Creation and Structure](../basics/project_creation_and_structure.md) 

This chapter shows a simple Controller implementation using the
[controller-runtime builder](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/builder)
libraries to do most of the Controller configuration.

While Kubernetes APIs have typically have 3 components, (Resource, Controller, Manager), this
example uses an existing Resource (ReplicaSet) and the `builder` package to hide many of the
 setup details.

For a more detailed look at creating Resources and Controllers that may be more complex,
see the [Resource](../basics/simple_resource.md), [Controller](../basics/simple_controller.md) and
[Manager](../basics/simple_controller_manager.md) examples.

{% method %}
## ReplicaSet Controller Setup {#hello-world-controller}

The example main program configures a new ReplicaSetController to watch for
create/update/delete events for ReplicaSets and Pods.

- On ReplicaSet create/update/delete events - Reconcile the *ReplicaSet*
- On Pod create/update/delete events - Reconcile the *ReplicaSet* that created the Pod
- Reconcile by calling `ReplicaSetController.Reconcile` with the Namespace and Name of
  ReplicaSet

{% sample lang="go" %}
```go
func main() {
  a, err := builder.SimpleController()
    // ReplicaSet is the Application type that
    // is Reconciled Respond to ReplicaSet events.
    ForType(&appsv1.ReplicaSet{}).
    // ReplicaSet creates Pods. Trigger
    // ReplicaSet Reconciles for Pod events.
    Owns(&corev1.Pod{}).
    // Call ReplicaSetController with the
    // Namespace / Name of the ReplicaSet
    Build(&ReplicaSetController{})
  if err != nil {
    log.Fatal(err)
  }
  log.Fatal(mrg.Start(signals.SetupSignalHandler()))
}

// ReplicaSetController is a simple Controller example implementation.
type ReplicaSetController struct {
  client.Client
}
```
{% endmethod %}

{% method %}
## ReplicaSet Implementation {#hello-world-controller}

ReplicaSetController implements reconcile.Reconciler.  It takes the Namespace and Name for
a ReplicaSet object and makes the state of the cluster match what is specified in the ReplicaSet
at the time Reconcile is called.  This typically means using a `client.Client` to read
the same of multiple objects, and perform create / update / delete as needed.

- Implement `InjectClient` to get a `client.Client` from the `application.Builder`
- Read the ReplicaSet object using the provided Namespace and Name
- List the Pods matching the ReplicaSet selector
- Set a Label on the ReplicaSet with the matching Pod count

Because the Controller watches for Pod events, the count will be updated any time
a Pod is created or deleted.

{% sample lang="go" %}
```go
// InjectClient is called by the application.Builder
// to provide a client.Client
func (a *ReplicaSetController) InjectClient(
  c client.Client) error {
  a.Client = c
  return nil
}

// Reconcile reads the Pods for a ReplicaSet and writes
// the count back as an annotation
func (a *ReplicaSetController) Reconcile(
  req reconcile.Request) (reconcile.Result, error) {
  // Read the ReplicaSet
  rs := &appsv1.ReplicaSet{}
  err := a.Get(context.TODO(), req.NamespacedName, rs)
  if err != nil {
    return reconcile.Result{}, err
  }

  // List the Pods matching the PodTemplate Labels
  pods := &corev1.PodList{}
  err = a.List(context.TODO(), 
    client.InNamespace(req.Namespace).
        MatchingLabels(rs.Spec.Template.Labels),
    pods)
  if err != nil {
    return reconcile.Result{}, err
  }

  // Update the ReplicaSet
  rs.Labels["selector-pod-count"] = 
    fmt.Sprintf("%v", len(pods.Items))
  err = a.Update(context.TODO(), rs)
  if err != nil {
    return reconcile.Result{}, err
  }

  return reconcile.Result{}, nil
}
```
{% endmethod %}
