{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

## Simple Main {#simple-world-main}

{% method %}
## Cmd {#simple-controller-manager-cmd}

The main program lives under the `cmd/` package created by `kubebuilder init`.
It does not need to be changed by the user for most cases.

The main program starts the controllers by calling the `inject` package.  It may installs CRDs.

1. Setup signal handlers
2. Get a kubeconfig to talk to an apiserver
3. Optionally create or update CRDs in the cluster
4. Create all controllers and start them.

The `inject` package is used to instantiate dependencies and allows the user to configure singletons
which may be used by controllers.  The advantage of doing this through inject instead of as
package variables is this allows the implementations to be faked when testing.

{% sample lang="go" %}

```go
// Note: This code lives under
// cmd/controller-manager/main.go

var installCRDs = flag.Bool("install-crds", true, 
	"install the CRDs used by the controller as part of startup")

// Controller-manager main.
func main() {
	flag.Parse()

    stopCh := signals.SetupSignalHandler()
	
    config := configlib.GetConfigOrDie()

    if *installCRDs {
        if err := install.NewInstaller(config).
        	Install(&InstallStrategy{crds: inject.Injector.CRDs});
        	err != nil {

            log.Fatalf("Could not create CRDs: %v", err)
        }
    }

    // Start the controllers
    if err := inject.RunAll(
    	run.RunArguments{Stop: stopCh},
    	args.CreateInjectArgs(config)); err != nil {
        
        log.Fatalf("%v", err)
    }
}

type InstallStrategy struct {
	install.EmptyInstallStrategy
	crds []*extensionsv1beta1.CustomResourceDefinition
}

func (s *InstallStrategy) GetCRDs()
    []*extensionsv1beta1.CustomResourceDefinition {
	return s.crds
}
```
{% endmethod %}


{% method %}
## Inject {#simple-controller-manager-inject}

The inject package is created by `kubebuilder init`.
It does not need to be changed by the user for most cases.

The inject package contains 2 files.

- A non-generated `inject.go` file
- A generated `zz_generated.kubebuiler.go` file

The generated file contains references to each of the Resources and Controllers defined in the
project.  It creates the controllers by calling each `ProvideController` function and then starting
the returned controller.  By modifying the `InjectArgs` and `RunArguments` definitions, users
may inject additional singletons.

{% sample lang="go" %}
```go
// Note: This code lives under
// pkg/inject/inject.go

// RunAll starts all of the informers and Controllers
func RunAll(rargs run.RunArguments, iargs args.InjectArgs) error {
    // Run functions to initialize injector
    for _, i := range Inject {
        if err := i(iargs); err != nil {
            return err
        }
    }
    Injector.Run(rargs)
    <-rargs.Stop
    return nil
}
```
{% endmethod %}