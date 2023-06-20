# Using a Custom Type

<aside class="note warning">
<h1>Component Config is deprecated</h1>

The ComponentConfig has been deprecated in the Controller-Runtime since its version 0.15.0.  [More info](https://github.com/kubernetes-sigs/controller-runtime/issues/895)
Moreover, it has undergone breaking changes and is no longer functioning as intended.
As a result, Kubebuilder, which heavily relies on the Controller Runtime, has also deprecated this feature,
no longer guaranteeing its functionality from version 3.11.0 onwards. You can find additional details on this issue [here](https://github.com/kubernetes-sigs/controller-runtime/issues/2370).

Please, be aware that it will force Kubebuilder remove this option soon in future release.

</aside>

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