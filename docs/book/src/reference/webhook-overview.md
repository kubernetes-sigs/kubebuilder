# Webhook

Webhooks are requests for information sent in a blocking fashion. A web
application implementing webhooks will send an HTTP request to other application
when certain event happens.

In the kubernetes world, there are 3 kinds of webhooks:
[admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks),
[authorization webhook](https://kubernetes.io/docs/reference/access-authn-authz/webhook/) and
[CRD conversion webhook](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/#webhook-conversion).

In [controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/webhook)
libraries, we support admission webhooks and CRD conversion webhooks.

Kubernetes supports these dynamic admission webhooks as of version 1.9 (when the
feature entered beta).

Kubernetes supports the conversion webhooks as of version 1.15 (when the
feature entered beta).
