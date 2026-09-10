# Webhook

Webhooks are requests for information sent in a blocking fashion. A web
application implementing webhooks sends an HTTP request to other applications
when a certain event happens.

In the Kubernetes world, there are 3 kinds of webhooks:
[admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks),
[authorization webhook](https://kubernetes.io/docs/reference/access-authn-authz/webhook/) and
[CRD conversion webhook](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion).

In [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/webhook?tab=doc)
libraries, we support admission webhooks and CRD conversion webhooks.

Kubernetes supports these dynamic admission webhooks as of version 1.9 (when the
feature entered beta).

Kubernetes supports the conversion webhooks as of version 1.15 (when the
feature entered beta).

## Disabling the webhook server

Set `--webhook-port=-1` to disable the webhook server:

```bash
./bin/manager --webhook-port=-1
```

controller-runtime logs `Webhook server is disabled` and does not listen on any port. This needs controller-runtime `v0.25.0` or later. Older releases treat `-1` as the default `9443`.

Only disable the server when no webhook configuration or CRD conversion points at the manager. Otherwise the API server cannot reach the webhooks and rejects the requests they guard.

To skip webhook registration when running the manager locally with `make run`, set `ENABLE_WEBHOOKS=false` instead.
