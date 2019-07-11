# Admission Webhooks

Admission webhooks are HTTP callbacks that receive admission requests, process
them and return admission responses.

Kubernetes provides the following types of admission webhooks:

- **Mutating Admission Webhook**:
These can mutate the object while it's being created or updated, before it gets
stored. It can be used to default fields in a resource requests, e.g. fields in
Deployment that are not specified by the user. It can be used to inject sidecar
containers.

- **Validating Admission Webhook**:
These can validate the object while it's being created or updated, before it gets
stored. It allows more complex validation than pure schema-based validation.
e.g. cross-field validation and pod image whitelisting.

The apiserver by default doesn't authenticate itself to the webhooks. However,
if you want to authenticate the clients, you can configure the apiserver to use
basic auth, bearer token, or a cert to authenticate itself to the webhooks.
You can find detailed steps
[here](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers).
