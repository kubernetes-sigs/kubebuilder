# Defining your Config

Now that you have a component config base project we need to customize the
values that are passed into the controller, to do this we can take a look at 
`config/manager/controller_manager_config.yaml`.

{{#literatego ./testdata/controller_manager_config.yaml}}

To see all the available fields you can look at the `v1alpha` Controller
Runtime config [ControllerManagerConfiguration][configtype]

[configtype]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/config/v1alpha1#ControllerManagerConfigurationSpec