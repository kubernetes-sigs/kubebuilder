# Creating Events

It is often useful to publish *Event* objects from the controller Reconcile function.  Events
allow users to see what is going on with a particular object, and allow automated processes
to see and respond to them.

{% panel style="success", title="Getting Events" %}
Recent Events for an object may be viewed by running `kubectl describe`
{% endpanel %}

{% method %}

Events are published from a Controller using an [EventRecorder](https://github.com/kubernetes/client-go/blob/master/tools/record/event.go#L56),
which can be created for a Controller by calling `GetRecorder(name string)` on a Manager.

```go
Event(object runtime.Object, eventtype, reason, message string)
```

- `object` is the object this event is about.
- `eventtype` is the type of this event, and is either *Normal* or *Warning*.
- `reason` is the reason this event is generated.  It should be short and unique with
  `UpperCamelCase` format.   The value could appear in *switch* statements by automation.
- `message` is intended to be consumed by humans.

{% endmethod %}
