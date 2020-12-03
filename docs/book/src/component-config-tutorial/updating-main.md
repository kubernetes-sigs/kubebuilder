# Updating main

Once you have defined your new custom component config type we need to make
sure our new config type has been imported and the types are registered with
the scheme. _If you used `kubebuilder create api` this should have been
automated._

```go
import (
    // ... other imports
    configv2 "tutorial.kubebuilder.io/project/apis/config/v2"
    // +kubebuilder:scaffold:imports
)
```
With the package imported we can confirm the types have been added.

```go
func init() {
	// ... other scheme registrations
	utilruntime.Must(configv2.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}
```

Lastly, we need to change the options parsing in
`main.go` to use this new type. To do this we'll chain `OfKind` onto
`ctrl.ConfigFile()` and pass in a pointer to the config kind.

```go
var err error
ctrlConfig := configv2.ProjectConfig{}
options := ctrl.Options{Scheme: scheme}
if configFile != "" {
    options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
    if err != nil {
        setupLog.Error(err, "unable to load the config file")
        os.Exit(1)
    }
}
```

Now if you need to use the `.clusterName` field we defined in our custom kind
you can call `ctrlConfig.ClusterName` which will be populated from the config
file supplied.