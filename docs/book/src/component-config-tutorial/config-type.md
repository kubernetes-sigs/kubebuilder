# Adding a new Config Type

<aside class="note warning">
<h1>Component Config is deprecated</h1>

The ComponentConfig has been deprecated in the Controller-Runtime since its version 0.15.0.  [More info](https://github.com/kubernetes-sigs/controller-runtime/issues/895)
Moreover, it has undergone breaking changes and is no longer functioning as intended.
As a result, Kubebuilder, which heavily relies on the Controller Runtime, has also deprecated this feature,
no longer guaranteeing its functionality from version 3.11.0 onwards. You can find additional details on this issue [here](https://github.com/kubernetes-sigs/controller-runtime/issues/2370).

Please, be aware that it will force Kubebuilder remove this option soon in future release.

</aside>

To scaffold out a new config Kind, we can use `kubebuilder create api`.

```bash
kubebuilder create api --group config --version v2 --kind ProjectConfig --resource --controller=false --make=false
```

Then, run `make build` to implement the interface for your API type, which would generate the file `zz_generated.deepcopy.go`.

<aside class="note">

<h1>Use --controller=false</h1>

You may notice this command from the `CronJob` tutorial although here we
explicitly setting `--controller=false` because `ProjectConfig` is not
intended to be an API extension and cannot be reconciled.

</aside>

This will create a new type file in `api/config/v2/` for the `ProjectConfig`
kind. We'll need to change this file to embed the
[v1alpha1.ControllerManagerConfigurationSpec](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/config/v1alpha1/#ControllerManagerConfigurationSpec)

{{#literatego ./testdata/projectconfig_types.go}}

Lastly, we'll change the `main.go` to reference this type for parsing the file.
