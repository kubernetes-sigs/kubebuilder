# Webhook

Webhooks are requests for information sent in a blocking fashion. A web
application implementing webhooks will send a HTTP request to other applications
when a certain event happens.

In the kubernetes world, there are 3 kinds of webhooks:
[admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks),
[authorization webhook](https://kubernetes.io/docs/reference/access-authn-authz/webhook/) and
[CRD conversion webhook](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion).

In [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/webhook?tab=doc)
libraries, we support admission webhooks and CRD conversion webhooks.

Kubernetes supports these dynamic admission webhooks as of version 1.9 (when the
feature entered beta).

Kubernetes supports the conversion webhooks as of version 1.15 (when the
feature entered beta).

