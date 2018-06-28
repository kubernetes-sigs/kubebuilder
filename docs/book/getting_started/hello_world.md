# Hello World

A new project may be scaffolded for a user by running `kubebuilder init` and then scaffolding a
new API with `kubebuilder create api`. More on this topic in
[Project Creation and Structure](../basics/project_creation_and_structure.md) 

This chapter shows a kubebuilder project for a simple API using the
[controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime/pkg) libraries
to implement the Controller and Manager.

Kubernetes APIs have 3 components.  Typically these components live in separate go packages:

* The API schema definition, or *Resource*, as a go struct containing ObjectMeta and TypeMeta.
* The API implementation, or *Controller*, as an implementation of the reconcile.Reconciler interface.
* The executable, or *Manager*, as a go main.

{% method %}
## FirstMate API Resource Definition {#hello-world-api}

FirstMate is a simple Resource (API) definition.  Resources are implemented as go structs containing:

- metav1.TypeMeta
- metav1.ObjectMeta
- API schema Spec - e.g. the user specified state
- API schema Status - e.g. the cluster reported state

Not shown here: Resources also have boilerplate for managing a runtime.Scheme which is scaffolded by
kubebuilder for users.

{% sample lang="go" %}
```go
// FirstMate is the API schema definition
type FirstMate struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec Spec `json:"spec"`
    Status Status `json:"status"`
}

// Spec is set by users and may also be defaulted or set by the cluster if left unspecified.
type Spec struct {
    Responsibilities []string `json:"responsibilities"`
}

// Status is set by the cluster to communicate the status of an object.
type Status struct {
	CompletedResponsibilities []string `json:"completedResponsibilities"`
}
```
{% endmethod %}

{% method %}
## FirstMate Controller {#hello-world-controller}

FirstMateController is a Reconciler implementation.  Reconcile takes an object
Namespace and Name as an argument and makes the state of the cluster match what is specified in the object
at the time Reconcile is called.

Reconcile will be trigger in response to events (create / update / delete) for FirstMate objects (and for any
objects owned by a FirstMate object through additional Watch calls).

{% sample lang="go" %}
```go
// FirstMateController implements the FirstMate API
type FirstMateController struct {
	client.Client
}

// Add creates a new Controller and adds it to the Manager
func func (c *FirstMateController) Reconcile(req reconcile.Request) (reconcile.Result, error) {
    fm := &v1beta1.FirstMate{}
	err := c.Get(context.TODO(), req.NamespacedName, fm)
	if err != nil {
		return reconcile.Result{}, err
	}
	
	// Implement your Controller logic here to read and write additional
	// objects using FirstMateController.Client
	
	// Return the result
	return reconcile.Result{}, nil
}

// Add is called by the main function to add a Controller to the Manager
func Add(mrg manager.Manager) error {
	// Read the FirstMate object
	c, err := controller.New("firstmate-controller", mrg,
		controller.Options{Reconcile: &FirstMateController{Client: mrg.GetClient()}})
	if err != nil {
		return err
	}

	// Trigger FirstMateController.Reconcile in response to FirstMate create/update/delete events
	err = c.Watch(&source.Kind{Type: &v1beta1.FirstMate{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	
	// Additional Watches for created objects go here.

	return nil
}
```
{% endmethod %}

{% method %}
## Manager {#hello-world-manager}

Manager is responsible for starting and providing dependencies to Controllers.  The main function creates
a new manager and adds Controllers to it.

{% sample lang="go" %}
```go
func main() {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mrg, err := manager.New(cfg, manager.Options{})
	if err != nil {
		log.Fatal(err)
	}

	// Setup Scheme for all resources
	if err := crewv1beta1.AddToScheme(mrg.GetScheme()); err != nil {
		log.Fatal(err)
	}

	// Setup all Controllers
	if err := v1beta1.Add(mrg); err != nil {
		log.Fatal(err)
	}

	// Start the Cmd
	log.Fatal(mrg.Start(signals.SetupSignalHandler()))
}
```
{% endmethod %}