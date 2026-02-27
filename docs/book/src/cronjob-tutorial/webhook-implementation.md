# Implementing defaulting/validating webhooks

If you want to implement [admission webhooks](../reference/admission-webhook.md)
for your CRD, the only thing you need to do is to implement the `CustomDefaulter`
and (or) the `CustomValidator` interface.

Kubebuilder takes care of the rest for you, such as

1. Creating the webhook server.
1. Ensuring the server has been added in the manager.
1. Creating handlers for your webhooks.
1. Registering each handler with a path in your server.

First, let's scaffold the webhooks for our CRD (CronJob). We'll need to run the following command with the `--defaulting` and `--programmatic-validation` flags (since our test project will use defaulting and validating webhooks):

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation
```

This will scaffold the webhook functions and register your webhook with the manager in your `main.go` for you.

## Custom Webhook Paths

You can specify custom HTTP paths for your webhooks using the `--defaulting-path` and `--validation-path` flags:

```bash
# Custom path for defaulting webhook
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --defaulting-path=/my-custom-mutate-path

# Custom path for validation webhook
kubebuilder create webhook --group batch --version v1 --kind CronJob --programmatic-validation --validation-path=/my-custom-validate-path

# Both webhooks with different custom paths
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation \
  --defaulting-path=/custom-mutate --validation-path=/custom-validate
```

This changes the path in the webhook marker annotation but does not change where the webhook files are scaffolded. The webhook files will still be created in `internal/webhook/v1/`.

<aside class="note">
<h4>Version Requirements</h4>

Custom webhook paths require **controller-runtime v0.21+**. In earlier versions (< `v0.21`), the webhook path must follow a specific pattern and cannot be customized. The path is automatically generated based on the resource's group, version, and kind (e.g., `/mutate-batch-v1-cronjob`).

</aside>

{{#literatego ./testdata/project/internal/webhook/v1/cronjob_webhook.go}}
