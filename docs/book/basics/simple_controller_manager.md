## Simple Main {#simple-world-main}

{% method %}
## Cmd {#simple-controller-manager-cmd}

The main program lives under the `cmd/` package created by `kubebuilder init`.
It does not need to be changed by the user for most cases.

The main program starts the Controllers that have been registered with the Manager.
Scaffolded Controllers are automatically registered with the Manager by scaffolding
an init function to the `controller` package.  Scaffolded Resources are 
automatically registered with the Manager Scheme by scaffolding an init
function to the `apis` package.

1. Get a kubeconfig to talk to an apiserver
2. Add APIs to the Manager's Scheme
3. Add Controllers to the Manager
4. Start the Manager

{% sample lang="go" %}
```go
func main() {
  // Get a config to talk to the apiserver
  cfg, err := config.GetConfig()
  if err != nil {
    log.Fatal(err)
  }

  // Create a new Cmd to provide shared dependencies and start components
  mgr, err := manager.New(cfg, manager.Options{})
  if err != nil {
    log.Fatal(err)
  }

  // Setup Scheme for all resources
  if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
    log.Fatal(err)
  }

  // Setup all Controllers
  if err := controller.AddToManager(mgr); err != nil {
    log.Fatal(err)
  }

  // Start the Cmd
  log.Fatal(mgr.Start(signals.SetupSignalHandler()))}
```
{% endmethod %}

{% method %}
## Leader Election

Controllers are usually designed to only have one active instance at any time, otherwise unexpected issues might occur. For example:

* kubernetes scheduler will have duplicated resource allocation with multiple instance active
* deployment controller will have unexpected number of pods running with multiple instances active

In practice, it is suggested to run controller in active-backup mode to achieve high availability. Leader Election is how we do this.

If three replicas of a certain controller pod is running, only the one wins leader election will run, and the other two pods is idle.
When leader pod is unavailable for some reason(network disconnectivity, server failure etc), the other two pods will start another
leader election round, and whoever becomes leader can continue running.

This example enables leader election, with two other parameters:

* LeaderElectionNamespace: leaderelection need to create configmap underhood, this field indicates which namespace should be used to create the configmap. If not provided, namespace the controller pod is running in will be used
* LeaderElectionID: a unique id for leaderelection. The default value is a constant, so it's better provided with a unique value.

{% sample lang="go" %}
```go
mgr, err := manager.New(cfg, manager.Options{
	MetricsBindAddress:      metricsAddr,
	LeaderElection:          true,
	LeaderElectionNamespace: "my-namespace",
	LeaderElectionID:        "awesome-controller-leader-election",
})
```

{% endmethod %}
