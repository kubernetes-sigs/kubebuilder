# FAQ

Frequently Ask Question

## How to use Kubernetes client in webhook?

You can declare the Kubernetes client in webhook from the controller manager easily.

<details>
  <summary>Read more</summary>

To begin, you need to import the client and declare the variable:

```go
import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	C client.Client
)
```

Next, it is very simple to fill the previous client variable from the controller manager.

```go
func (r *MyAwesomeCRD) SetupWebhookWithManager(mgr ctrl.Manager) error {
	C = mgr.GetClient()

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}
```

Finally, you can use the client in your validator for example:

```go
func (r *MyAwesomeCRD) ValidateDelete() error {
	return C.Get(// ... //)
}
```
</details>
