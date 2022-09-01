# Kustomize Example

This directory contains an example of using `kubebuilder alpha config-gen` as a Kustomize transformer
plugin.

This enables using `config-gen` in traditional kustomize workflows (patch, bases, etc).

## Summary

`config-gen` may be used from `kustomize` as a transformer plugin.  This allows the output
to be customized using `commonLabels`, `commonAnnotations`, `namespace`, etc.

When invoked from `kustomize`, `config-gen` will generate resources from the project code
if they do not already exist as `resources` inputs.  If the resources that would have been
generated are provided as `resources` input, the inputs will be modified by the transformer.

## Install kustomize

Install the latest version of `kustomize`.

```sh
curl -Ss "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash
```

## Configure `kubebuilder alpha config-gen` as a plugin

```sh
# create the script under $HOME/.config/kustomize/plugin/kubebuilder.sigs.k8s.io/kubebuilderconfiggen
kubebuilder alpha config-gen install-as-plugin
```

## Use `kustomize` to invoke the plugin

Kustomize will invoke the `kubebuilder alpha config-gen` subcommand as a transformer plugin.

```sh
kustomize build --enable-alpha-plugins .
```

See [types.go](../../types.go) for the KubebuilderConfigGen schema.
