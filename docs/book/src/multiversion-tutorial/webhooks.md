# Setting up the webhooks

Our conversion is in place, so all that's left is to tell
controller-runtime about our conversion.

Normally, we'd run

```shell
kubebuilder create webhook --group batch --version v1 --kind CronJob --conversion
```

to scaffold out the webhook setup.  However, we've already got webhook
setup, from when we built our defaulting and validating webhooks!

## Webhook setup...

{{#literatego ./testdata/project/api/v1/cronjob_webhook.go}}

## ...and `main.go`

Similarly, our existing main file is sufficient:

{{#literatego ./testdata/project/main.go}}

Everything's set up and ready to go!  All that's left now is to test out
our webhooks.
