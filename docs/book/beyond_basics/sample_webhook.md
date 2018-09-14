# Webhook Example

This chapter walks through a simple webhook implementation.

It uses the [controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook) libraries to implement
a Webhook Server and Manager.

Same as controllers, a Webhook Server is a
[`Runable`](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/manager#Runnable) which needs to be registered to a manager.
Arbitrary number of `Runable`s can be registered to a manager,
so a webhook server can run with other controllers in the same manager.
They will share the same dependencies provided by the manager. For example, shared cache, client, scheme, etc.

## Setup

#### Way to Deploy your Webhook Server

There are various ways to deploy the webhook server in terms of

1. Where the serving certificates live.
1. In what environment the webhook server runs, in a pod or directly on a VM, etc.
1. If in a pod, on what type of node, worker nodes or master node.

The recommended way to deploy the webhook server is

1. Run the webhook server as a regular pod on worker nodes through a workload API, e.g. Deployment or StatefulSet.
1. Put the certificate in a k8s secret in the same namespace as the webhook server
1. Mount the secret as a volume in the pod
1. Create a k8s service to front the webhook server.

#### Creating a Handler

{% method %}

The business logic for a Webhook exists in a Handler.
A Handler implements the `admission.Handler` interface, which contains a single `Handle` method.

If a Handler implements `inject.Client` and `inject.Decoder` interfaces,
the manager will automatically inject the client and the decoder into the Handler.

Note: The `client.Client` provided by the manager reads from a cache which is lazily initialized.
To eagerly initialize the cache, perform a read operation with the client before starting the server.

`podAnnotator` is a Handler, which implements the `admission.Handler`, `inject.Client` and `inject.Decoder` interfaces.

Details about how to implement an admission webhook podAnnotator is covered in a later section.

{% sample lang="go" %}
```go
type podAnnotator struct {
    client  client.Client
    decoder types.Decoder
}

// podAnnotator implements admission.Handler.
var _ admission.Handler = &podAnnotator{}

func (a *podAnnotator) Handle(ctx context.Context, req types.Request) types.Response {
    ...
}

// podAnnotator implements inject.Client.
var _ inject.Client = &podAnnotator{}

// InjectClient injects the client into the podAnnotator
func (a *podAnnotator) InjectClient(c client.Client) error {
    a.client = c
    return nil
}

// podAnnotator implements inject.Decoder.
var _ inject.Decoder = &podAnnotator{}

// InjectDecoder injects the decoder into the podAnnotator
func (a *podAnnotator) InjectDecoder(d types.Decoder) error {
    a.decoder = d
    return nil
}
```
{% endmethod %}


#### Configuring a Webhook and Registering the Handler

{% method %}

A Webhook configures what type of requests the Handler should accept from the apiserver. Options include:
- The type of the Operations (CRUD)
- The type of the Targets (Deployment, Pod, etc)
- The type of the Handler (Mutating, Validating)

When the Server starts, it will register all Webhook Configurations with the apiserver to start accepting and
routing requests to the Handlers.

[controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder) provides a useful package for
building a webhook.
You can incrementally set the configuration of a webhook and then invoke `Build` to complete building a webhook.

If you want to specify the name and(or) path for your webhook instead of using the default, you can invoke
`Name("yourname")` and `Path("/yourpath")` respectively.

{% sample lang="go" %}
```go
wh, err := builder.NewWebhookBuilder().
    Mutating().
    Operations(admissionregistrationv1beta1.Create).
    ForType(&corev1.Pod{}).
    Handlers(&podAnnotator{}).
    WithManager(mgr).
    Build()
if err != nil {
    // handle error
}
```
{% endmethod %}


#### Creating a Server

{% method %}

A Server registers Webhook Configuration with the apiserver and creates an HTTP server to route requests to the handlers.

The server is behind a Kubernetes Service and provides a certificate to the apiserver when serving requests.

The Server depends on a Kubernetes Secret containing this certificate to be mounted under `CertDir`.

If the Secret is empty, during bootstrapping the Server will generate a certificate and write it into the Secret.

A new webhook server can be created by invoking `webhook.NewServer`.
The Server will be registered to the provided manager.
You can specify `Port`, `CertDir` and various `BootstrapOptions`.
For the full list of Server options, please see [GoDoc](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook).

{% sample lang="go" %}
```go
svr, err := webhook.NewServer("foo-admission-server", mgr, webhook.ServerOptions{
    CertDir: "/tmp/cert",
    BootstrapOptions: &webhook.BootstrapOptions{
        Secret: &types.NamespacedName{
            Namespace: "default",
            Name:      "foo-admission-server-secret",
        },

        Service: &webhook.Service{
            Namespace: "default",
            Name:      "foo-admission-server-service",
            // Selectors should select the pods that runs this webhook server.
            Selectors: map[string]string{
                "app": "foo-admission-server",
            },
        },
    },
})
if err != nil {
    // handle error
}
```
{% endmethod %}

#### Registering a Webhook with the Server

You can register webhook(s) in the webhook server by invoking `svr.Register(wh)`.


## Implementing Webhook Handler


#### Implementing the Handler Business Logic

{% method %}

`decoder types.Decoder` is a decoder that knows how the decode all core type and your CRD types.

`client client.Client` is a client that knows how to talk to the API server.

The guideline of returning HTTP status code is that:
- If the server decides to admit the request, it should return 200 and set
[`Allowed`](https://github.com/kubernetes/api/blob/f456898a08e4bbc5891694118f3819f324de12ff/admission/v1beta1/types.go#L86-L87)
to `true`.
- If the server rejects the request due to an admission policy reason, it should return 200, set
[`Allowed`](https://github.com/kubernetes/api/blob/f456898a08e4bbc5891694118f3819f324de12ff/admission/v1beta1/types.go#L86-L87)
to `false` and provide an informational message as reason.
- If the request is not well formatted, the server should reject it with 400 (Bad Request) and an error message.
- If the server encounters an unexpected error during processing, it should reject the request with 500 (Internal Error).

`controller-runtime` provides various helper methods for constructing Response.
- `ErrorResponse` for rejecting a request due to an error.
- `PatchResponse` for mutating webook to admit a request with patches.
- `ValidationResponse` for admitting or rejecting a request with a reason message.

{% sample lang="go" %}
```go
type podAnnotator struct {
    client  client.Client
    decoder types.Decoder
}

// podAnnotator Iimplements admission.Handler.
var _ admission.Handler = &podAnnotator{}

// podAnnotator adds an annotation to every incoming pods.
func (a *podAnnotator) Handle(ctx context.Context, req types.Request) types.Response {
    pod := &corev1.Pod{}

    err := a.decoder.Decode(req, pod)
    if err != nil {
        return admission.ErrorResponse(http.StatusBadRequest, err)
    }
    copy := pod.DeepCopy()

    err = a.mutatePodsFn(ctx, copy)
    if err != nil {
        return admission.ErrorResponse(http.StatusInternalServerError, err)
    }
    // admission.PatchResponse generates a Response containing patches.
    return admission.PatchResponse(pod, copy)
}

// mutatePodsFn add an annotation to the given pod
func (a *podAnnotator) mutatePodsFn(ctx context.Context, pod *corev1.Pod) error {
    if pod.Annotations == nil {
        pod.Annotations = map[string]string{}
    }
    pod.Annotations["example-mutating-admission-webhook"] = "foo"
    return nil
}
```
{% endmethod %}
