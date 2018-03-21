# Building reference documentation

## Build reference documentation

You may build Kubernetes style reference documentation for your APIs to `docs/reference/build/index.html`


```go
kubebuilder docs
```

**Note:** There is currently an issue where building docs does not work if multiple versions of APIs for the
same group are defined.

## Create an example for your API

You may create an example that will be included in the reference documentation by running the following command
and editing the newly created file:

```sh
kubebuilder create example --group <group> --version <version> --kind <kind>
```

## Add overview or API group documentation

You may add information about the API groups or creating an overview by editing the .md files
under `docs/reference/static_includes`.