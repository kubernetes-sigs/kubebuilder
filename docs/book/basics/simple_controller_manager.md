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
