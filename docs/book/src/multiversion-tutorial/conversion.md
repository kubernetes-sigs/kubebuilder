# Implementing conversion

With the model for conversion in place, it is time to actually implement
the conversion functions.  Create a conversion webhook
for the CronJob API version `v1` (Hub) to Spoke the CronJob API version
`v2` see:

```go
kubebuilder create webhook --group batch --version v1 --kind CronJob --conversion --spoke v2
```

The above command generates the `cronjob_conversion.go` next to the
`cronjob_types.go` file, to avoid
cluttering up the main types file with extra functions.

<aside class="note" role="note">
<p class="note-title">Conversion Webhooks and Custom Paths</p>

Unlike defaulting and validation webhooks, conversion webhooks do not support custom paths
via command-line flags. Conversion webhooks use CRD conversion configuration
(`.spec.conversion.webhook.clientConfig.service.path` in the CRD) rather than webhook
marker annotations. The path for conversion webhooks is managed differently and cannot
be customized through kubebuilder flags.
</aside>

## Hub...

First, implement the hub.  Choose the v1 version as the hub:

{{#literatego ./testdata/project/api/v1/cronjob_conversion.go}}

## ... and Spokes

Then, implement the spoke, the v2 version:

{{#literatego ./testdata/project/api/v2/cronjob_conversion.go}}

Now that the conversions are in place, all that is needed is to
wire up main to serve the webhook!
