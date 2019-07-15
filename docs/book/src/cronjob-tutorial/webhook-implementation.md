# Implementing admission webhooks

If you want to implement [admission webhooks](../reference/admission-webhook.md)
for your CRD, the only thing you need to do is to implement the `Defaulter`
and (or) the `Validator` interface.

Kubebuilder takes care of the rest for you, such as

1. Creating the webhook server.
1. Ensuring the server has been added in the manager.
1. Creating handlers for your webhooks.
1. Registering each handler with a path in your server.

{{#literatego ./testdata/project/api/v1/cronjob_webhook.go}}
