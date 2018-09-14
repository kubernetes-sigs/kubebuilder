# Webhook

Webhooks are HTTP callbacks, providing a way for notifications to be delivered to an external web server.
A web application implementing webhooks will send an HTTP request (typically POST) to other application when certain event happens.
In the kubernetes world, there are 3 kinds of webhooks:
[admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks),
[authorization webhook](https://kubernetes.io/docs/reference/access-authn-authz/webhook/) and CRD conversion webhook.

In [controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook) libraries,
currently we only support admission webhooks.
CRD conversion webhooks will be supported after it is released in kubernetes 1.12.

## Admission Webhook

Admission webhooks are HTTP callbacks that receive admission requests, process them and return admission responses.
There are two types of admission webhooks: mutating admission webhook and validating admission webhook.
With mutating admission webhooks, you may change the request object before it is stored (e.g. for implementing defaulting of fields)
With validating admission webhooks, you may not change the request, but you can reject it (e.g. for implementing validation of the request).

#### Why Admission Webhooks are Important

Admission webhooks are the mechanism to enable kubernetes extensibility through CRD.
- Mutating admission webhook is the only way to do defaulting for CRDs.
- Validating admission webhook allows for more complex validation than pure schema-based validation.
e.g. cross-field validation or cross-object validation.

It can also be used to add custom logic in the core kubernetes API.

#### Mutating Admission Webhook

A mutating admission webhook receives an admission request which contains an object.
The webhook can either decline the request directly or returning JSON patches for modifying the original object.
- If admitting the request, the webhook is responsible for generating JSON patches and send them back in the
admission response.
- If declining the request, a reason message should be returned in the admission response.

#### Validating Admission Webhook

A validating admission webhook receives an admission request which contains an object.
The webhook can either admit or decline the request.
A reason message should be returned in the admission response if declining the request.

#### Authentication

The apiserver by default doesn't authenticate itself to the webhooks.
That means the webhooks don't authenticate the identities of the clients.

But if you want to authenticate the clients, you need to configure the apiserver to use basic auth, bearer token,
or a cert to authenticate itself to the webhooks. You can find detailed steps
[here](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers).

#### Configure Admission Webhooks Dynamically

{% method %}

Admission webhooks can be configured dynamically via the `admissionregistration.k8s.io/v1beta1` API.
So your cluster must be 1.9 or later and has enabled the API.

You can do CRUD operations on WebhookConfiguration objects as on other k8s objects.

{% sample lang="yaml" %}

```yaml
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: <name of itself>
webhooks:
- name: <webhook name, e.g. validate-deployment.example.com>
  rules:
  - apiGroups:
    - apps
    apiVersions:
    - v1
    operations:
    - CREATE
    resources:
    - deployments
  clientConfig:
    service:
      namespace: <namespace of the service>
      name: <name of the service>
    caBundle: <pem encoded ca cert that signs the server cert used by the webhook>
```

{% endmethod %}

