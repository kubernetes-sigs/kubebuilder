# Defining your Config

<aside class="note warning">
<h1>Component Config is deprecated</h1>

The ComponentConfig has been deprecated in the Controller-Runtime since its version 0.15.0.  [More info](https://github.com/kubernetes-sigs/controller-runtime/issues/895)
Moreover, it has undergone breaking changes and is no longer functioning as intended.
As a result, Kubebuilder, which heavily relies on the Controller Runtime, has also deprecated this feature,
no longer guaranteeing its functionality from version 3.11.0 onwards. You can find additional details on this issue [here](https://github.com/kubernetes-sigs/controller-runtime/issues/2370).

Please, be aware that it will force Kubebuilder remove this option soon in future release.

</aside>

Now that you have a component config base project we need to customize the
values that are passed into the controller, to do this we can take a look at 
`config/manager/controller_manager_config.yaml`.

{{#literatego ./testdata/controller_manager_config.yaml}}

To see all the available fields you can look at the `v1alpha` Controller
Runtime config [ControllerManagerConfiguration][configtype]

[configtype]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/config/v1alpha1#ControllerManagerConfigurationSpec