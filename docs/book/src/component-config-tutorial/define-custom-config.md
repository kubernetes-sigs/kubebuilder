# Defining your Custom Config

Now that you have a custom component config we change the 
`config/manager/controller_manager_config.yaml` to use the new GVK you defined.

{{#literatego ./testdata/project/config/manager/controller_manager_config.yaml}}

This type uses the new `ProjectConfig` kind under the GVK
`config.tutorial.kubebuilder.io/v2`, with these custom configs we can add any
`yaml` serializable fields that your controller needs and begin to reduce the
reliance on `flags` to configure your project.