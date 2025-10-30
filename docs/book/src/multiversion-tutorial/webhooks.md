# Setting up the webhooks

Our conversion is in place, so all that's left is to tell
controller-runtime about our conversion.

## Webhook setup for v1...

{{#literatego ./testdata/project/internal/webhook/v1/cronjob_webhook.go}}

## Webhook setup for v2...

Since v2 has a different Schedule structure (using CronSchedule instead of a string),
we need a different webhook implementation:

{{#literatego ./testdata/project/internal/webhook/v2/cronjob_webhook.go}}

## ...and `main.go`

Similarly, our existing main file is sufficient:

{{#literatego ./testdata/project/cmd/main.go}}

Everything's set up and ready to go!  All that's left now is to test out
our webhooks.

