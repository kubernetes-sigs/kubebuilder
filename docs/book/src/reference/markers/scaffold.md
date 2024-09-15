# Scaffold

The `+kubebuilder:scaffold` marker is a key part of the Kubebuilder scaffolding system. It marks locations in generated
files where additional code will be injected as new resources (such as controllers, webhooks, or APIs) are scaffolded.
This enables Kubebuilder to seamlessly integrate newly generated components into the project without affecting
user-defined code.

<aside class="note warning">
<H1>If you delete or change the `+kubebuilder:scaffold` markers</H1>

The Kubebuilder CLI specifically looks for these markers in expected
files during code generation. If the marker is moved or removed, the CLI will
not be able to inject the necessary code, and the scaffolding process may
fail or behave unexpectedly.

</aside>

## How It Works

When you scaffold a new resource using the Kubebuilder CLI (e.g., `kubebuilder create api`),
the CLI identifies `+kubebuilder:scaffold` markers in key locations and uses them as placeholders
to insert the required imports and registration code.

## Example Usage in `main.go`

Here is how the `+kubebuilder:scaffold` marker is used in a typical `main.go` file. To illustrate how it works, consider the following command to create a new API:

```shell
kubebuilder create api --group crew --version v1 --kind Admiral --controller=true --resource=true
```

### To Add New Imports

The `+kubebuilder:scaffold:imports` marker allows the Kubebuilder CLI to inject additional imports,
such as for new controllers or webhooks. When we create a new API, the CLI automatically adds the required import paths
in this section.

For example, after creating the `Admiral` API in a single-group layout,
the CLI will add `crewv1 "<repo-path>/api/v1"` to the imports:

```go
import (
    "crypto/tls"
    "flag"
    "os"

    // Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
    // to ensure that exec-entrypoint and run can make use of them.
    _ "k8s.io/client-go/plugin/pkg/client/auth"
    ...
    crewv1 "sigs.k8s.io/kubebuilder/testdata/project-v4/api/v1"
    // +kubebuilder:scaffold:imports
)
```

### To Register a New Scheme

The `+kubebuilder:scaffold:scheme` marker is used to register newly created API versions with the runtime scheme,
ensuring the API types are recognized by the manager.

For example, after creating the Admiral API, the CLI will inject the
following code into the `init()` function to register the scheme:


```go
func init() {
    ...
    utilruntime.Must(crewv1.AddToScheme(scheme))
    // +kubebuilder:scaffold:scheme
}
```

## To Set Up a Controller

When we create a new controller (e.g., for Admiral), the Kubebuilder CLI injects the controller
setup code into the manager using the `+kubebuilder:scaffold:builder` marker. This marker indicates where
the setup code for new controllers should be added.

For example, after creating the `AdmiralReconciler`, the CLI will add the following code
to register the controller with the manager:

```go
if err = (&crewv1.AdmiralReconciler{
    Client: mgr.GetClient(),
    Scheme: mgr.GetScheme(),
}).SetupWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create controller", "controller", "Admiral")
    os.Exit(1)
}
// +kubebuilder:scaffold:builder
```

The `+kubebuilder:scaffold:builder` marker ensures that newly scaffolded controllers are
properly registered with the manager, so that the controller can reconcile the resource.

## List of `+kubebuilder:scaffold` Markers

| Marker                                     | Usual Location               | Function                                                                        |
|--------------------------------------------|------------------------------|---------------------------------------------------------------------------------|
| `+kubebuilder:scaffold:imports`            | `main.go`                    | Marks where imports for new controllers, webhooks, or APIs should be injected.   |
| `+kubebuilder:scaffold:scheme`             | `init()` in `main.go`         | Used to add API versions to the scheme for runtime.                             |
| `+kubebuilder:scaffold:builder`            | `main.go`                    | Marks where new controllers should be registered with the manager.              |
| `+kubebuilder:scaffold:webhook`            | `webhooks suite tests` files  | Marks where webhook setup functions are added.                                  |
| `+kubebuilder:scaffold:crdkustomizeresource`| `config/crd`                 | Marks where CRD custom resource patches are added.                              |
| `+kubebuilder:scaffold:crdkustomizewebhookpatch` | `config/crd`              | Marks where CRD webhook patches are added.                                      |
| `+kubebuilder:scaffold:crdkustomizecainjectionpatch` | `config/crd`           | Marks where CA injection patches are added for the webhook.                     |
| `+kubebuilder:scaffold:manifestskustomizesamples` | `config/samples`           | Marks where Kustomize sample manifests are injected.                            |
| `+kubebuilder:scaffold:e2e-webhooks-checks` | `test/e2e`                   | Adds e2e checks for webhooks depending on the types of webhooks scaffolded.      |

<aside class="note">
<h1>Creating Your Own Markers</h1>

If you are using Kubebuilder as a library to create [your own plugins](./../../plugins/creating-plugins.md) and extend its CLI functionalities,
you have the flexibility to define and use your own markers. To implement your own markers, refer to the [kubebuilder/v4/pkg/machinery](https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/machinery),
which provides tools to create and manage markers effectively.

</aside>



