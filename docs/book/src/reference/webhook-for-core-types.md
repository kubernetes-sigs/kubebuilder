# Admission Webhook for Core Types

It is very easy to build admission webhooks for CRDs, which has been covered in
the [CronJob tutorial][cronjob-tutorial]. Given that kubebuilder doesn't support webhook scaffolding
for core types, you have to use the library from controller-runtime to handle it.
There is an [example](https://github.com/kubernetes-sigs/controller-runtime/tree/master/examples/builtins)
in controller-runtime.

It is suggested to use kubebuilder to initialize a project, and then you can
follow the steps below to add admission webhooks for core types.

## Implementing Your Handler Using `Handle`

Your handler must implement the [admission.Handler](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/webhook/admission#Handler) interface. This function is responsible for both mutating and validating the incoming resource.

### Update your webhook:

**Example**

```go
package v1

import (
    "context"
    "encoding/json"
    "net/http"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
    corev1 "k8s.io/api/core/v1"
)

// **Note**: in order to have controller-gen generate the webhook configuration for you, you need to add markers. For example:

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io

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

    // Mutate the fields in pod
    pod.Annotations["example.com/mutated"] = "true"

    marshaledPod, err := json.Marshal(pod)
    if err != nil {
        return admission.Errored(http.StatusInternalServerError, err)
    }
    return admission.Patched(req.Object.Raw, marshaledPod)
}
```
<aside class="note">
<h1>Markers for Webhooks</h1>

Notice that we use kubebuilder markers to generate webhook manifests.
This marker is responsible for generating a mutating webhook manifest.

The meaning of each marker can be found [here](./markers/webhook.md).

To have controller-gen automatically generate the webhook configuration for you, you need to add the appropriate markers in your code. These markers should follow a specific format, especially when defining the webhook path.

The format for the webhook path is as follows:

```go
/mutate-<group>-<version>-<kind>
```

Since this documentation example uses Pod from the core API group, the group should be an empty string.

For example, the marker for a mutating webhook for Pod might look like this:

```go
// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io
```
</aside>

## Update main.go

Now you need to register your handler in the webhook server.

```go
mgr.GetWebhookServer().Register("/mutate--v1-pod", &webhook.Admission{
    Handler: &podAnnotator{Client: mgr.GetClient()},
})
```

You need to ensure the path here match the path in the marker.

### Client/Decoder

If you need a client and/or decoder, just pass them in at struct construction time.

```go
mgr.GetWebhookServer().Register("/mutate--v1-pod", &webhook.Admission{
    Handler: &podAnnotator{
        Client:   mgr.GetClient(),
        decoder:  admission.NewDecoder(mgr.GetScheme()),
    },
})
```

## By using Custom interfaces instead of Handle

### Update your webhook:

**Example**

```go
package v1

import (
  "context"
  "fmt"

  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/runtime"
  ctrl "sigs.k8s.io/controller-runtime"
  logf "sigs.k8s.io/controller-runtime/pkg/log"
  "sigs.k8s.io/controller-runtime/pkg/webhook"
  "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var podlog = logf.Log.WithName("pod-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *corev1.Pod) SetupWebhookWithManager(mgr ctrl.Manager) error {
  runAsNonRoot := true
  allowPrivilegeEscalation := false

  return ctrl.NewWebhookManagedBy(mgr).
    For(r).
    WithValidator(&PodCustomValidator{}).
    WithDefaulter(&PodCustomDefaulter{
      DefaultSecurityContext: &corev1.SecurityContext{
        RunAsNonRoot:             &runAsNonRoot,             // Set to true
        AllowPrivilegeEscalation: &allowPrivilegeEscalation, // Set to false
      },
    }).
    Complete()
}

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io,admissionReviewVersions=v1

// +kubebuilder:object:generate=false
// PodCustomDefaulter struct is responsible for setting default values on the Pod resource
// when it is created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type PodCustomDefaulter struct {
  // Default security context to be applied to Pods
  DefaultSecurityContext *corev1.SecurityContext

  // TODO: Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &PodCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type Pod
func (d *PodCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
  pod, ok := obj.(*corev1.Pod)
  if !ok {
    return fmt.Errorf("expected a Pod object but got %T", obj)
  }
  podlog.Info("CustomDefaulter for corev1.Pod", "name", pod.GetName())

  // Apply the default security context if it's not set
  for i := range pod.Spec.Containers {
    if pod.Spec.Containers[i].SecurityContext == nil {
      pod.Spec.Containers[i].SecurityContext = d.DefaultSecurityContext
    }
  }

  // Mutate the fields in Pod (e.g., adding an annotation)
  if pod.Annotations == nil {
    pod.Annotations = map[string]string{}
  }
  pod.Annotations["example.com/mutated"] = "true"

  // TODO: Add any additional defaulting logic here.

  return nil
}

// +kubebuilder:webhook:path=/validate--v1-pod,mutating=false,failurePolicy=fail,groups="",resources=pods,verbs=create;update;delete,versions=v1,name=vpod.kb.io,admissionReviewVersions=v1

// +kubebuilder:object:generate=false
// PodCustomValidator struct is responsible for validating the Pod resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type PodCustomValidator struct {
}

var _ webhook.CustomValidator = &PodCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Pod
func (v *PodCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
  pod, ok := obj.(*corev1.Pod)
  if !ok {
    return nil, fmt.Errorf("expected a Pod object but got %T", obj)
  }
  podlog.Info("Validation for  corev1.Pod upon creation", "name", pod.GetName())

  // Ensure the Pod has at least one container
  if len(pod.Spec.Containers) == 0 {
    return nil, fmt.Errorf("pod must have at least one container")
  }

  // TODO: Add any additional creation validation logic here.

  return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Pod
func (v *PodCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
  pod, ok := newObj.(*corev1.Pod)
  if !ok {
    return nil, fmt.Errorf("expected a Pod object but got %T", newObj)
  }
  podlog.Info("Validation for corev1.Pod upon Update", "name", pod.GetName())

  oldPod := oldObj.(*corev1.Pod)
  // Prevent changing a specific annotation
  if oldPod.Annotations["example.com/protected"] != pod.Annotations["example.com/protected"] {
    return nil, fmt.Errorf("the annotation 'example.com/protected' cannot be changed")
  }

  // Prevent changing the security context after creation
  for i := range pod.Spec.Containers {
    if !equalSecurityContexts(oldPod.Spec.Containers[i].SecurityContext, pod.Spec.Containers[i].SecurityContext) {
      return nil, fmt.Errorf("security context of containers cannot be changed after creation")
    }
  }

  // TODO: Add any additional update validation logic here.

  return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Pod
func (v *PodCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
  pod, ok := obj.(*corev1.Pod)
  if !ok {
    return nil, fmt.Errorf("expected a Pod object but got %T", obj)
  }
  podlog.Info("Deletion for corev1.Pod upon Update", "name", pod.GetName())

  // Prevent deletion of protected Pods
  if pod.Annotations["example.com/protected"] == "true" {
    return nil, fmt.Errorf("protected pods cannot be deleted")
  }

  // TODO: Add any additional deletion validation logic here.

  return nil, nil
}

// equalSecurityContexts checks if two SecurityContexts are equal
func equalSecurityContexts(a, b *corev1.SecurityContext) bool {
  // Implement your logic to compare SecurityContexts here
  // For example, you can compare specific fields:
  return a.RunAsNonRoot == b.RunAsNonRoot &&
          a.AllowPrivilegeEscalation == b.AllowPrivilegeEscalation
}

```

### Update the main.go

```go
if os.Getenv("ENABLE_WEBHOOKS") != "false" {
  if err := (&corev1.Pod{}).SetupWebhookWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create webhook", "webhook", "corev1.Pod")
    os.Exit(1)
  }
}
```

## Deploy

Deploying it is just like deploying a webhook server for CRD. You need to
1) provision the serving certificate
2) deploy the server

You can follow the [tutorial](/cronjob-tutorial/running.md).

## What are `Handle` and Custom Interfaces?

In the context of Kubernetes admission webhooks, the `Handle` function and the custom interfaces (`CustomValidator` and `CustomDefaulter`) are two different approaches to implementing webhook logic. Each serves specific purposes, and the choice between them depends on the needs of your webhook.

## Purpose of the `Handle` Function

The `Handle` function is a core part of the admission webhook process. It is responsible for directly processing the incoming admission request and returning an `admission.Response`. This function is particularly useful when you need to handle both validation and mutation within the same function.

### Mutation

If your webhook needs to modify the resource (e.g., add or change annotations, labels, or other fields), the `Handle` function is where you would implement this logic. Mutation involves altering the resource before it is persisted in Kubernetes.

### Response Construction

The `Handle` function is also responsible for constructing the `admission.Response`, which determines whether the request should be allowed or denied, or if the resource should be patched (mutated). The `Handle` function gives you full control over how the response is built and what changes are applied to the resource.

## Purpose of Custom Interfaces (`CustomValidator` and `CustomDefaulter`)

The `CustomValidator` and `CustomDefaulter` interfaces provide a more modular approach to implementing webhook logic. They allow you to separate validation and defaulting (mutation) into distinct methods, making the code easier to maintain and reason about.

## When to Use Each Approach

- **Use `Handle` when**:
  - You need to both mutate and validate the resource in a single function.
  - You want direct control over how the admission response is constructed and returned.
  - Your webhook logic is simple and doesn’t require a clear separation of concerns.

- **Use `CustomValidator` and `CustomDefaulter` when**:
  - You want to separate validation and defaulting logic for better modularity.
  - Your webhook logic is complex, and separating concerns makes the code easier to manage.
  - You don’t need to perform mutation and validation in the same function.

[cronjob-tutorial]: /cronjob-tutorial/cronjob-tutorial.md