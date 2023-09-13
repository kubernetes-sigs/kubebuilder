# Watching Operator Managed Resources

Kubebuilder and the Controller Runtime libraries allow for controllers
to implement the logic of their CRD through easy management of Kubernetes resources.

## Controlled & Owned Resources

Managing dependency resources is fundamental to a controller, and it's not possible to manage them without watching for changes to their state.
- Deployments must know when the ReplicaSets that they manage are changed
- ReplicaSets must know when their Pods are deleted, or change from healthy to unhealthy.

Through the `Owns()` functionality, Controller Runtime provides an easy way to watch dependency resources for changes.
A resource can be defined as dependent on another resource through the 'ownerReferences' field.

As an example, we are going to create a `SimpleDeployment` resource.
The `SimpleDeployment`'s purpose is to manage a `Deployment` that users can change certain aspects of, through the `SimpleDeployment` Spec.
The `SimpleDeployment` controller's purpose is to make sure that it's owned `Deployment` (has an ownerReference which points to `SimpleDeployment` resource) always uses the settings provided by the user.

### Provide basic templating in the `Spec`

{{#literatego ./testdata/owned-resource/api.go}}

### Manage the Owned Resource

{{#literatego ./testdata/owned-resource/controller.go}}