# Admission Webhook for Core Types

It is very easy to build admission webhooks for CRDs, which has been covered in
the CronJob tutorial. Given that kubebuilder doesn't support webhook scaffolding
for core types, you have to use the library from controler-runtime to handle it.
There is an [example](https://github.com/kubernetes-sigs/controller-runtime/tree/master/examples/builtins)
in controller-runtime.

It is suggested to use kubebuilder to initialize a project, and then you can
follow the steps below to add admission webhooks for core types.

## Implement Your Handler

You need to have your handler implements the
[admission.Handler](https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/webhook/admission#Handler)
interface.

```go
type podAnnotator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (a *podAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// mutate the fields in pod

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
```

If you need a client, just pass in the client at struct construction time.

If you add the `InjectDecoder` method for your handler, a decoder will be
injected for you.

```go
func (a *podAnnotator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
```

**Note**: in order to have controller-gen generate the webhook configuration for
you, you need to add markers. For example,
`// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io`

## Update main.go

Now you need to register your handler in the webhook server.

```go
mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{Handler: &podAnnotator{Client: mgr.GetClient()}})
```

You need to ensure the path here match the path in the marker.

## Deploy

Deploying it is just like deploying a webhook server for CRD. You need to
1) provision the serving certificate 2) deploy the server

You can follow the [tutorial](/cronjob-tutorial/running.md).
