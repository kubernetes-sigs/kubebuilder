
# Using Generated Informers

This chapter describes how to use the client-go generated Informers with controller-runtime Watches.

[Link to reference documentation](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/source#Informer)

**Note:** The `source.Informers` and `source.Kind` sources create different caches.  Using both
`source.Informers` and `source.Kind` sources in the same project will result in duplicate caches.
This will not impact correctness, but will double the cache memory.

{% method %}
## Creating client-go generated Informers and Adding them to the Manager

Instantiate the generated InformerFactory and add it to the Manager so it is started automatically.

**Note:** The generated Informer should be used with the generated client.

{% sample lang="go" %}
```go
// Create the InformerFactory
generatedClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
generatedInformers := kubeinformers.NewSharedInformerFactory(generatedClient, time.Minute*30)

err := mgr.Add(manager.RunnableFunc(func(s <-chan struct{}) error {
    generatedInformers.Start(s)
    return nil
}))
if err != nil {
    glog.Fatalf("error Adding InformerFactory to the Manager: %v", err)
}
```
{% endmethod %}


{% method %}
## Watching Resources using the client-go generated Informer

Controllers may watch Resources using client-go generated Informers though the`source.Informers` struct.

This example configures a Controller to watch for Services events, and to call Reconcile with
the Service key.

If Service *default/foo* is created, updated or deleted, then Reconcile will be called with
*namespace: default, name: foo*.

The generated InformerFactory must be manually wired into the Controller creation code.

{% sample lang="go" %}
```go
// Setup Watch using the client-go generated Informer
err := ctrl.Watch(
    &source.Informer{InformerProvider: generatedInformers.Core().V1().Services()},
    &handler.EnqueueRequestForObject{},
)
if err != nil {
    glog.Fatalf("error Watching Services: %v", err)
}
```
{% endmethod %}

{% method %}
## Starting the Manager

The InformerFactory will be started through the Manager.

{% sample lang="go" %}
```go
mgr.Start(signals.SetupSignalHandler())
```
{% endmethod %}
