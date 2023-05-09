# Admission Webhook for Core Types

It is very easy to build admission webhooks for CRDs, which has been covered in
the [CronJob tutorial][cronjob-tutorial]. Given that kubebuilder doesn't support webhook scaffolding
for core types, you have to use the library from controller-runtime to handle it.
There is an [example](https://github.com/kubernetes-sigs/controller-runtime/tree/master/examples/builtins)
in controller-runtime.

It is suggested to use kubebuilder to initialize a project, and then you can
follow the steps below to add admission webhooks for core types. The example shows
how to set up a mutating webhook following the controller-runtime's webhook builder.

## Implement Your Webhook

You need to have your webhook to implement the
[admission.CustomDefaulter](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/webhook/admission?tab=doc#CustomDefaulter)
interface.

```go
type podAnnotator struct {}

func (a *podAnnotator) setupWebhookWithManager(mgr ctrl.Manager) error {
       return ctrl.NewWebhookManagedBy(mgr).
               For(&corev1.Pod{}).
               WithDefaulter(a).
               Complete()
}

func (a *podAnnotator) Default(ctx context.Context, obj runtime.Object) error {
    log := logf.FromContext(ctx)
    pod, ok := obj.(*corev1.Pod)
    if !ok {
        return fmt.Errorf("expected a Pod but got a %T", obj)
    }

	// mutate the fields in pod
    if pod.Annotations == nil {
        pod.Annotations = map[string]string{}
    }
    pod.Annotations["example-mutating-admission-webhook"] = "foo"
    log.Info("Annotated pod")

    return nil
}
```

**Note**: in order to have controller-gen generate the webhook configuration for
you, you need to add markers. For example,
`// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io`
(for webhooks the `path` is of format `/mutate-<group>-<version>-<kind>`. Since this documentation uses `Pod` from the core API group, the group needs to be an empty string).

## Update main.go

Now you need to register your handler in the webhook server.

```go
    if err := (&podAnnotator{}).setupWebhookWithManager(mgr); err != nil {
        entryLog.Error(err, "unable to create webhook", "webhook", "Pod")
        os.Exit(1)
    }
```

You need to ensure the path here match the path in the marker.

### Client/Decoder

If you need a client and/or decoder, just pass them in at struct construction time.

```go
mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{
	Handler: &podAnnotator{
		Client:   mgr.GetClient(),
		decoder:  admission.NewDecoder(mgr.GetScheme()),
	},
})
```

## Deploy

Deploying it is just like deploying a webhook server for CRD. You need to
1) provision the serving certificate
2) deploy the server

You can follow the [tutorial](/cronjob-tutorial/running.md).


[cronjob-tutorial]: /cronjob-tutorial/cronjob-tutorial.md
