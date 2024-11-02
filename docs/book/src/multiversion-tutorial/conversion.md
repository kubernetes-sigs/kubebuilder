# Implementing conversion

With our model for conversion in place, it's time to actually implement
the conversion functions.  We'll create a conversion webhook
for our CronJob API version `v1` (Hub) to Spoke our CronJob API version
`v2` see:

```go
kubebuilder create webhook --group batch --version v1 --kind CronJob --conversion --spoke v2
```

The above command will generate the `cronjob_conversion.go` next to our
`cronjob_types.go` file, to avoid
cluttering up our main types file with extra functions.

## Hub...

First, we'll implement the hub.  We'll choose the v1 version as the hub:

{{#literatego ./testdata/project/api/v1/cronjob_conversion.go}}

## ... and Spokes

Then, we'll implement our spoke, the v2 version:

{{#literatego ./testdata/project/api/v2/cronjob_conversion.go}}

Now that we've got our conversions in place, all that we need to do is
wire up our main to serve the webhook!
