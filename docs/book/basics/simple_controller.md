{% panel style="danger", title="STAGING" %}
Staging Environment - Not Official Documentation!

This book contains APIs, libraries and tools that are proposals only and have not been ratified!
{% endpanel %}


# Simple Controller Example

This chapter walks through a simple Controller implementation.

This is a simple controller example for the ContainerSet API shown in *Simple Resource Example*.

The controller reads the 

> $ kubebuilder create resource --group workloads --version v1beta1 --kind ContainerSet
> pkg/controller/containerset/controller.go

{% method %}
## Setup

The controller is setup in the package `init` function.  Any errors during setup should be
be returned when starting the controller manager, not in the init function.

- Create a new `ControllerMux` with the Reconcile function specified.
- Watch for ContainerSet events and reconcile the corresponding ContainerSet object
- Watch for Deployment events and reconcile the Owner object if the reference has "controller: true",
  and the Owner type is a ContainerSet

{% sample lang="go" %}
```go
func init() {
	NewController(kb.Default, kb.Default)
}

func NewController(cb kb.ControllerBuilder, c kb.Client) {
  c := &kb.ControllerMux{Reconcile: &Controller{c}.Reconcile}

  kb.Handle(&v1beta1.ContainerSet{}, c)

  kb.Handle(watch.ToOwner{
  	Generated: &v1.Deployment{},
  	Owner: &v1beta1.ContainerSet{},
  	Controller: true}}, c)
}

type Controller struct {
	client kb.Client
}
```
{% endmethod %}

{% method %}
## Implementation

The controller is implemented in the `Reconcile` function.  This function takes the namespace/name
key of the ContainerSet object to reconcile.  It then reads the ContainerSet object, checks
if a matching Deployment already exists, and either creates or updates the Deployment.

Finally the controller updates the Status on the ContainerSet.  Because the Deployment and ContainerSet
cannot be written in a single transaction, in the event the Status update fails the controller will
need to set the Status during the following Reconcilation.

Note that if the Deployment is deleted or changed by some other actor in the system, the controller
will receive and event and recreate / update the Deployment.

{% sample lang="go" %}

```go
func (c *Controller) Reconcile(k sdk.ReconcileKey) error {
  s := &v1beta1.ContainerSet{ObjectMeta: v1.ObjectMeta{
  	Name: k.Name, Namespace: k.Namespace,
  }}
  if err := c.client.Get(s); err != nil {
    if apierrors.IsNotFound(err) {
      return nil
    }
    return err
  }
  
  d := &v1.Deployment{
  	ObjectMeta: v1.ObjectMeta{
      	Name: k.Name, Namespace: k.Namespace,
    },
    Spec: v1.DeploymentSpec{...},
  }
  kb.SetOwnerReference(d, s)
  err := c.client.Get(d)
  if err != nil && !apierrors.IsNotFound(err) {
    return err
  }
  if apierrors.IsNotFound(err) {
      if err := c.client.Create(d); err != nil {
        return err
      }  	
  } else {
      d.Spec = v1.DeploymentSpec{...}
      kb.SetOwnerReference(d, s)
      if c.client.Update(d); err != nil {
        return err
      }  
  }
  
  s.Status.HealthyReplicas = d.Status.ReadyReplicas
  if err := c.client.Update(s); err != nil {
      return err
  }
  return nil
}
```
{% endmethod %}


