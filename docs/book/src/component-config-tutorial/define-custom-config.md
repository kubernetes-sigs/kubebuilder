# Defining your Custom Config

<aside class="note warning">
<h1>Component Config is deprecated</h1>

The ComponentConfig has been deprecated in the Controller-Runtime since its version 0.15.0.  [More info](https://github.com/kubernetes-sigs/controller-runtime/issues/895)
Moreover, it has undergone breaking changes and is no longer functioning as intended.
As a result, Kubebuilder, which heavily relies on the Controller Runtime, has also deprecated this feature,
no longer guaranteeing its functionality from version 3.11.0 onwards. You can find additional details on this issue [here](https://github.com/kubernetes-sigs/controller-runtime/issues/2370).

Please, be aware that it will force Kubebuilder remove this option soon in future release.

</aside>

Now that you have a custom component config we change the 
`config/manager/controller_manager_config.yaml` to use the new GVK you defined.

{{#literatego ./testdata/project/config/manager/controller_manager_config.yaml}}

This type uses the new `ProjectConfig` kind under the GVK
`config.tutorial.kubebuilder.io/v2`, with these custom configs we can add any
`yaml` serializable fields that your controller needs and begin to reduce the
reliance on `flags` to configure your project.