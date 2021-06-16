# Standalone Example

This directory contains an example of using `kubebuilder alpha config-gen` as a stand alone
config generator.

## Summary

`config-gen` may be used to generate configuration for a kubebuilder project by invoking the
subcommand on a `.yaml` file containing a `KubebuilderConfigGen` resource.

## Run the command

Specify the `KubebuilderConfigGen` configuration file as the first argument:

```sh
# emit the config to stdout
kubebuilder alpha config-gen kubebuilderconfiggen.yaml
```

```sh
# write the config to a file
kubebuilder alpha config-gen kubebuilderconfiggen.yaml > _output/config.yaml
```

```sh
# apply the config to a cluster
kubebuilder alpha config-gen kubebuilderconfiggen.yaml | kubectl apply -f -
```

## Run with patch overrides

`config-gen` will automatically apply any additional resource files provided as patches to the output.

```sh
kubebuilder alpha config-gen kubebuilderconfiggen.yaml patch.yaml
```

## Also see

See [types.go](../../types.go) for the KubebuilderConfigGen schema.
