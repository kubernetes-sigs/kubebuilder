{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Creating Events

It is often useful to publish *Event* objects from the controller Reconcile function.  Events
allow users to see what is going on with a particular object, and allow automated processes
to see and respond to them.

{% panel style="success", title="Getting Events" %}
Recent Events for an object may be viewed by running `kubectl describe`
{% endpanel %}

{% method %}

Events are published from a controller using an [EventRecorder](https://github.com/kubernetes/client-go/blob/master/tools/record/event.go#L56),
which is automatically configured by `kubebuilder create resource`.  The event recorded is
intended to be used from the `Reconcile` function.

```go
Event(object runtime.Object, eventtype, reason, message string)
```

- `object` is the object this event is about.
- `eventtype` is the type of this event, and is either *Normal* or *Warning*.
- `reason` is the reason this event is generated.  It should be short and unique with
  `UpperCamelCase` format.   The value could appear in *switch* statements by automation.
- `message` is intended to be consumed by humans.

{% sample lang="go" %}
```go
import (
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/api/core/v1"
)

func (bc *BeeController) Reconcile(k types.ReconcileKey) error {
  b, err := bc.beeclient.
  	Bees(k.Namespace).
  	Get(k.Name, metav1.GetOptions{})
  if err != nil {
    return err
  }
  bc.beerecorder.Event(
  	b, v1.EventTypeNormal, "ReconcileBee", b.Name)
  return nil
}
```
{% endmethod %}
