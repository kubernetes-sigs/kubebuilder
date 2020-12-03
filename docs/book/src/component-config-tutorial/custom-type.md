# Using a Custom Type

<aside class="note warning">

<h1>Built-in vs Custom Type</h1>

If you don't need to add custom fields to configure your project you can stop
now and move on, if you'd like to be able to pass additional information keep
reading.

</aside>

If your project needs to accept additional non-controller runtime specific
configurations, e.g. `ClusterName`, `Region` or anything serializable into
`yaml` you can do this by using `kubebuilder` to create a new type and then
updating your `main.go` to setup the new type for parsing.

The rest of this tutorial will walk through implementing a custom component
config type. 