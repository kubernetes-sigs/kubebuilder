# Setting up the webhooks

Our conversion is in place, so all that's left is to tell
controller-runtime about our conversion.

## Webhook setup for v1...

The v1 webhook handles conversion (as the hub) and provides validation/defaulting
for the v1 CronJob format with a string-based schedule:

{{#literatego ./testdata/project/internal/webhook/v1/cronjob_webhook.go}}

## Webhook setup for v2...

The v2 webhook provides validation and defaulting for the v2 CronJob format
with the structured CronSchedule type. Note how the validation logic differs
from v1 - it builds a cron expression from the individual schedule fields:

{{#literatego ./testdata/project/internal/webhook/v2/cronjob_webhook.go}}

## ...and `main.go`

Similarly, our existing main file is sufficient:

{{#literatego ./testdata/project/cmd/main.go}}

Everything's set up and ready to go!  All that's left now is to test out
our webhooks.
