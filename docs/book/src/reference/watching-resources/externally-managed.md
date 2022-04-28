# Watching Externally Managed Resources

By default, Kubebuilder and the Controller Runtime libraries allow for controllers
to easily watch the resources that they manage as well as dependent resources that are `Owned` by the controller.
However, those are not always the only resources that need to be watched in the cluster.

## User Specified Resources

There are many examples of Resource Specs that allow users to reference external resources.
- Ingresses have references to Service objects
- Pods have references to ConfigMaps, Secrets and Volumes
- Deployments and Services have references to Pods

This same functionality can be added to CRDs and custom controllers.
This will allow for resources to be reconciled when another resource it references is changed.

As an example, we are going to create a `ConfigDeployment` resource.
The `ConfigDeployment`'s purpose is to manage a `Deployment` whose pods are always using the latest version of a `ConfigMap`.
While ConfigMaps are auto-updated within Pods, applications may not always be able to auto-refresh config from the file system.
Some applications require restarts to apply configuration updates.
- The `ConfigDeployment` CRD will hold a reference to a ConfigMap inside its Spec.
- The `ConfigDeployment` controller will be in charge of creating a deployment with Pods that use the ConfigMap.
These pods should be updated anytime that the referenced ConfigMap changes, therefore the ConfigDeployments will need to be reconciled on changes to the referenced ConfigMap.

### Allow for linking of resources in the `Spec`

{{#literatego ./testdata/external-indexed-field/api.go}}

### Watch linked resources

{{#literatego ./testdata/external-indexed-field/controller.go}}
