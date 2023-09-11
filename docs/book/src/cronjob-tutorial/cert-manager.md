# Deploying cert-manager

We suggest using [cert-manager](https://github.com/cert-manager/cert-manager) for
provisioning the certificates for the webhook server. Other solutions should
also work as long as they put the certificates in the desired location.

You can follow
[the cert-manager documentation](https://cert-manager.io/docs/installation/)
to install it.

cert-manager also has a component called [CA
Injector](https://cert-manager.io/docs/concepts/ca-injector/), which is responsible for
injecting the CA bundle into the [`MutatingWebhookConfiguration`](https://pkg.go.dev/k8s.io/api/admissionregistration/v1#MutatingWebhookConfiguration)
/ [`ValidatingWebhookConfiguration`](https://pkg.go.dev/k8s.io/api/admissionregistration/v1#ValidatingWebhookConfiguration).

To accomplish that, you need to use an annotation with key
`cert-manager.io/inject-ca-from`
in the [`MutatingWebhookConfiguration`](https://pkg.go.dev/k8s.io/api/admissionregistration/v1#MutatingWebhookConfiguration)
/ [`ValidatingWebhookConfiguration`](https://pkg.go.dev/k8s.io/api/admissionregistration/v1#ValidatingWebhookConfiguration) objects.
The value of the annotation should point to an existing [certificate request instance](https://cert-manager.io/docs/concepts/certificaterequest/)
in the format of `<certificate-namespace>/<certificate-name>`.

This is the [kustomize](https://github.com/kubernetes-sigs/kustomize) patch we
used for annotating the [`MutatingWebhookConfiguration`](https://pkg.go.dev/k8s.io/api/admissionregistration/v1#MutatingWebhookConfiguration)
/ [`ValidatingWebhookConfiguration`](https://pkg.go.dev/k8s.io/api/admissionregistration/v1#ValidatingWebhookConfiguration) objects.

```yaml
{{#include ./testdata/project/config/default/webhookcainjection_patch.yaml}}
```
