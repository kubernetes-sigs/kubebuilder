# Deploying the cert manager

We suggest using [cert manager](https://github.com/jetstack/cert-manager) for
provisioning the certificates for the webhook server. Other solutions should
also work as long as they put the certificates in the desired location.

You can follow
[the cert manager documentation](https://docs.cert-manager.io/en/latest/getting-started/install/kubernetes.html)
to install it.

Cert manager also has a component called CA injector, which is responsible for
injecting the CA bundle into the Mutating|ValidatingWebhookConfiguration.

To accomplish that, you need to use an annotation with key
`certmanager.k8s.io/inject-ca-from`
in the Mutating|ValidatingWebhookConfiguration objects.
The value of the annotation should point to an existing certificate CR instance
in the format of `<certificate-namespace>/<certificate-name>`.

This is the [kustomize](https://github.com/kubernetes-sigs/kustomize) patch we
used for annotating the Mutating|ValidatingWebhookConfiguration objects.
```yaml
{{#include ./testdata/project/config/default/webhookcainjection_patch.yaml}}
```
