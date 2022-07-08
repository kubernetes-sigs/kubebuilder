# Adding a new Config Type

To scaffold out a new config Kind, we can use `kubebuilder create api`.

```bash
kubebuilder create api --group config --version v2 --kind ProjectConfig --resource --controller=false --make=false
```


<aside class="note">

<h1>Use --controller=false</h1>

You may notice this command from the `CronJob` tutorial although here we
explicity setting `--controller=false` because `ProjectConfig` is not
intended to be an API extension and cannot be reconciled.

</aside>

This will create a new type file in `api/config/v2/` for the `ProjectConfig`
kind. We'll need to change this file to embed the 
[v1alpha1.ControllerManagerConfigurationSpec](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/config/v1alpha1/#ControllerManagerConfigurationSpec)

Note that because you create a new CustomType you will need to update the default
scaffold to be able ti use the new Kind created instead, see:

{{#literatego ./testdata/projectconfig_types.go}}

Lastly, we'll change the `main.go` to reference this type for parsing the file.