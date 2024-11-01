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
| `+kubebuilder:scaffold:crdkustomizecainjectionns`                            | `config/default`             | Marks where CA injection patches are added for the conversion webhooks.                                                                                                                |
| `+kubebuilder:scaffold:crdkustomizecainjectioname`                           | `config/default`             | Marks where CA injection patches are added for the conversion webhooks.                                                                                                                |
| **(No longer supported)** `+kubebuilder:scaffold:crdkustomizecainjectionpatch` | `config/crd`                 | Marks where CA injection patches are added for the webhooks. Replaced by `+kubebuilder:scaffold:crdkustomizecainjectionns` and `+kubebuilder:scaffold:crdkustomizecainjectioname`  |
| `+kubebuilder:scaffold:manifestskustomizesamples` | `config/samples`           | Marks where Kustomize sample manifests are injected.                            |
| `+kubebuilder:scaffold:e2e-webhooks-checks` | `test/e2e`                   | Adds e2e checks for webhooks depending on the types of webhooks scaffolded.      |

<aside class="note warning">
<h1> **(No longer supported)** `+kubebuilder:scaffold:crdkustomizecainjectionpatch` </h1>

If you find this marker in your code please:

1. **Remove the CERTMANAGER Section from `config/crd/kustomization.yaml`:**

   Delete the `CERTMANAGER` section to prevent unintended CA injection patches for CRDs. Ensure the following lines are removed or commented out:

   ```yaml
   # [CERTMANAGER] To enable cert-manager, uncomment all the sections with [CERTMANAGER] prefix.
   # patches here are for enabling the CA injection for each CRD
   #- path: patches/cainjection_in_firstmates.yaml
   # +kubebuilder:scaffold:crdkustomizecainjectionpatch
   ```

2. **Ensure CA Injection Configuration in `config/default/kustomization.yaml`:**

   Under the `[CERTMANAGER]` replacement in `config/default/kustomization.yaml`, add the following code for proper CA injection generation:

   **NOTE:** You must ensure that the code contains the following target markers:
    - `+kubebuilder:scaffold:crdkustomizecainjectionns`
    - `+kubebuilder:scaffold:crdkustomizecainjectioname`

   ```yaml
   # - source: # Uncomment the following block if you have a ConversionWebhook (--conversion)
   #     kind: Certificate
   #     group: cert-manager.io
   #     version: v1
   #     name: serving-cert # This name should match the one in certificate.yaml
   #     fieldPath: .metadata.namespace # Namespace of the certificate CR
   #   targets: # Do not remove or uncomment the following scaffold marker; required to generate code for target CRD.
   # +kubebuilder:scaffold:crdkustomizecainjectionns
   # - source:
   #     kind: Certificate
   #     group: cert-manager.io
   #     version: v1
   #     name: serving-cert # This name should match the one in certificate.yaml
   #     fieldPath: .metadata.name
   #   targets: # Do not remove or uncomment the following scaffold marker; required to generate code for target CRD.
   # +kubebuilder:scaffold:crdkustomizecainjectioname
   ```

3. **Ensure Only Conversion Webhook Patches in `config/crd/patches`:**

   The `config/crd/patches` directory and the corresponding entries in `config/crd/kustomization.yaml` should only contain files for conversion webhooks. Previously, a bug caused the patch file to be generated for any webhook, but only patches for webhooks scaffolded with the `--conversion` option should be included.

For further guidance, you can refer to examples in the `testdata/` directory in the Kubebuilder repository.

> **Alternatively**: You can use the [`alpha generate`](./../rescaffold.md) command to re-generate the project from scratch
> using the latest release available. Afterward, you can re-add only your code implementation on top to ensure your project
> includes all the latest bug fixes and enhancements.

</aside>

<aside class="note">
<h1>Creating Your Own Markers</h1>

If you are using Kubebuilder as a library to create [your own plugins](./../../plugins/creating-plugins.md) and extend its CLI functionalities,
you have the flexibility to define and use your own markers. To implement your own markers, refer to the [kubebuilder/v4/pkg/machinery](https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/machinery),
which provides tools to create and manage markers effectively.

</aside>



