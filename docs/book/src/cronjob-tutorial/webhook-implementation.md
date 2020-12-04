# Implementing defaulting/validating webhooks

If you want to implement [admission webhooks](../reference/admission-webhook.md)
for your CRD, the only thing you need to do is to implement the `Defaulter`
and (or) the `Validator` interface.

Kubebuilder takes care of the rest for you, such as

1. Creating the webhook server.
1. Ensuring the server has been added in the manager.
1. Creating handlers for your webhooks.
1. Registering each handler with a path in your server.

First, let's scaffold the webhooks for our CRD (CronJob). We’ll need to run the following command with the `--defaulting` and `--programmatic-validation` flags (since our test project will use defaulting and validating webhooks):

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation
```

This will scaffold the webhook functions and register your webhook with the manager in your `main.go` for you.

<aside class="note">

<h1>Supporting older cluster versions</h1>

The default WebhookConfiguration manifests created alongside your Go webhook implementation
use API version `v1`. If your project intends to support Kubernetes cluster versions older
than v1.16, set `--webhook-version v1beta1`. See the [webhook reference][webhook-reference]
for more information.

[webhook-reference]: /reference/webhook-overview.md#supporting-older-cluster-versions

</aside>

{{#literatego ./testdata/project/api/v1/cronjob_webhook.go}}
