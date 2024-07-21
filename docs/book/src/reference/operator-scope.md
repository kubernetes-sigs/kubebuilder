# Operator Scope

This section dives into the two crucial concepts of Custom Resource Definitions (CRDs) and Operators, specifically their respective scope and how both impact resource management in your Kubernetes environment.


In the world of Kubernetes solutions, scope defines the reach of an projects's management capabilities. This essentially means deciding which resources across your Kubernetes cluster the manager can watch and manage. Here's a breakdown of the two main options:

- **Namespace-scoped**: Only watches and manages resources within a single namespace.
- **Cluster-scoped**: Watches and manages resources across the entire cluster.

## Choosing the Right Scope:

- **Cluster-scoped operators**: Ideal for managing resources that can be created in any namespace, like certificate management or centralized configuration.
- **Namespace-scoped operators**: Suitable for scenarios like:
- **Flexible deployment**: Allows for independent upgrades and isolation of failures within a namespace.
- **Differing API definitions**: Enables specific configurations for individual namespaces.

## Defaults and Considerations:

- The kubebuilder init command by default creates a cluster-scoped project.
- This document outlines steps to convert a cluster-scoped operator to a namespace-scoped one, but emphasizes that a cluster-scoped approach might be more suitable in certain situations.
- Important Note: When creating a Manager instance in the main.go file, the watched and cached namespaces are set using Manager Options. Remember, only clients provided by cluster-scoped Managers can manage cluster-scoped CRDs.

## Manager watching options
### Watching resources in all Namespaces (default)
A Manager is initialized with no Namespace option specified, or `Namespace: ""` will watch all Namespaces:

```go
...
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection,
    LeaderElectionID:   "f1c5ece8.example.com",
})
...
```
## Watching resources in a single Namespace

To restrict the scope of the Manager’s cache to a specific Namespace set the Namespace field in Options:

```go
...
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection,
    LeaderElectionID:   "f1c5ece8.example.com",
    Namespace:          "operator-namespace",
})
...
```
## Watching resources in a set of Namespaces

It is possible to use MultiNamespacedCacheBuilder from Options to watch and manage resources in a set of Namespaces:

```go
...
namespaces := []string{"foo", "bar"} // List of Namespaces
...
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection,
    LeaderElectionID:   "f1c5ece8.example.com",
    NewCache:           cache.MultiNamespacedCacheBuilder(namespaces),
})
...
```
In the above example, a CR created in a Namespace not in the set passed to Options will not be reconciled by its controller because the Manager does not manage that Namespace.

**IMPORTANT**: Note that this is not intended to be used for excluding Namespaces, this is better done via a Predicate.

## Restricting Roles and permissions

An operator’s scope defines its Manager’s cache’s scope but not the permissions to access the resources. After updating the Manager’s scope to be Namespaced, Role-Based Access Control (RBAC) permissions applied to the operator’s service account should be restricted accordingly.

These permissions are found in the directory config/rbac/. The ClusterRole in role.yaml and ClusterRoleBinding in role_binding.yaml are used to grant the operator permissions to access and manage its resources.

NOTE For changing the operator’s scope only the role.yaml and role_binding.yaml manifests need to be updated. For the purposes of this doc, the other RBAC manifests `<kind>_editor_role.yaml`, `<kind>_viewer_role.yaml`, and `auth_proxy_*.yaml` are not relevant to changing the operator’s resource permissions.

### Changing the permissions to Namespaced

To change the scope of the RBAC permissions from cluster-wide to a specific namespace, you will need to:

Use Roles instead of ClusterRoles.
RBAC markers defined in the controller (e.g controllers/memcached_controller.go) are used to generate the operator’s RBAC ClusterRole (e.g config/rbac/role.yaml). The default markers don’t specify a namespace property and will result in a ClusterRole.

Update the RBAC markers in <kind>_controller.go with namespace=<namespace> where the Role is to be applied, such as:

```go
//+kubebuilder:rbac:groups=cache.example.com,namespace=memcached-operator-system,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.example.com,namespace=memcached-operator-system,resources=memcacheds/status,verbs=get;update;patch
```
Then run make manifests to update config/rbac/role.yaml. In our example it would look like:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  creationTimestamp: null
  name: manager-role
  namespace: memcached-operator-system
```
- Use RoleBindings instead of ClusterRoleBindings. The config/rbac/role_binding.yaml needs to be manually updated:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  creationTimestamp: null
  name: manager-role
  namespace: memcached-operator-system
Use RoleBindings instead of ClusterRoleBindings. The config/rbac/role_binding.yaml needs to be manually updated:
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: manager-role
subjects:
- kind: ServiceAccount
  name: controller-manager
  namespace: system
```
## Configuring watch namespaces dynamically

Instead of having any Namespaces hard-coded in the main.go file a good practice is to use an environment variable to allow the restrictive configurations. The one suggested here is WATCH_NAMESPACE, a comma-separated list of namespaces passed to the manager at deploy time.

### Configuring Namespace scoped operators

- Add a helper function in the main.go file:

```go
// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {
    // WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
    // which specifies the Namespace to watch.
    // An empty value means the operator is running with cluster scope.
    var watchNamespaceEnvVar = "WATCH_NAMESPACE"

    ns, found := os.LookupEnv(watchNamespaceEnvVar)
    if !found {
        return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
    }
    return ns, nil
}
```

- Use the environment variable value:

```go
...
watchNamespace, err := getWatchNamespace()
if err != nil {
    setupLog.Error(err, "unable to get WatchNamespace, " +
       "the manager will watch and manage resources in all namespaces")
}

mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection,
    LeaderElectionID:   "f1c5ece8.example.com",
    Namespace:          watchNamespace, // namespaced-scope when the value is not an empty string
})
...
```
- Define the environment variable in the config/manager/manager.yaml:

```yaml
spec:
  containers:
  - command:
    - /manager
    args:
    - --leader-elect
    image: controller:latest
    name: manager
    resources:
      limits:
        cpu: 100m
        memory: 30Mi
      requests:
        cpu: 100m
        memory: 20Mi
    env:
      - name: WATCH_NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
  terminationGracePeriodSeconds: 10
```
**NOTE** WATCH_NAMESPACE here will always be set as the namespace where the operator is deployed.

## Configuring cluster-scoped operators with MultiNamespacedCacheBuilder

- Add a helper function to get the environment variable value in the main.go file as done in the previous example (e.g getWatchNamespace())
- Use the environment variable value and check if it is a multi-namespace scenario:

```go 
...
watchNamespace, err := getWatchNamespace()
if err != nil {
    setupLog.Error(err, "unable to get WatchNamespace, " +
        "the manager will watch and manage resources in all Namespaces")
}

options := ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection,
    LeaderElectionID:   "f1c5ece8.example.com",
    Namespace:          watchNamespace, // namespaced-scope when the value is not an empty string
}

// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
if strings.Contains(watchNamespace, ",") {
    setupLog.Info("manager set up with multiple namespaces", "namespaces", watchNamespace)
    // configure cluster-scoped with MultiNamespacedCacheBuilder
    options.Namespace = ""
    options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(watchNamespace, ","))
}
...
```
- Define the environment variable in the config/manager/manager.yaml:

```yaml
...
    env:
      - name: WATCH_NAMESPACE
        value: "ns1,ns2"
  terminationGracePeriodSeconds: 10
...
```
## Updating your CSV’s installModes

If your operator is integrated with OLM, you will want to update your CSV base’s spec.installModes list to support the desired namespacing requirements. Support for multiple types of namespacing is allowed, so supporting multiple install modes in a CSV is permitted. After doing so, update your bundle or package manifests by following the linked guides.

### Watching resources in all Namespaces (default)
Only the AllNamespaces install mode is supported: true by default, so no changes are required.

### Watching resources in a single Namespace
If the operator can watch its own namespace, set the following in your spec.installModes list:

```yaml
  - type: OwnNamespace
    supported: true
```
If the operator can watch a single namespace that is not its own, set the following in your spec.installModes list:

```yaml
  - type: SingleNamespace
    supported: true
```
## Watching resources in multiple Namespaces
If the operator can watch multiple namespaces, set the following in your spec.installModes list:

```yaml
  - type: MultiNamespace
    supported: true
```