# Config-gen

`kubebuilder alpha config-gen` is a subcommand that generates configuration for kubebuilder projects as a configuration function.

Supports:

- Generating CRDs and RBAC from code
- Generating webhook certificates for development
- Selectively enabling / disabling components such as prometheus and webhooks
  - See [types.go](apis/v1alpha1/types.go) for a list of components

## Usage

`config-gen` may be run as a standalone command or from kustomize as a transformer plugin.

### Standalone command

config-gen may be run as a standalone program on the commandline.

See [examples/standalone](examples/standalone/README.md)

### From kustomize

config-gen may be run as a Kustomize plugin using kustomize.

See [examples/kustomize](examples/kustomize/README.md)

### Extending `config-gen`

config-gen may be extended by composing additional functions on top of it.

See examples of layering additional functions on:

- [examples/basicextension](examples/basicextension/README.md)
- [examples/advancedextension](examples/advancedextension/README.md)

## `KubebuilderConfigGen`

See [types.go](apis/v1alpha1/types.go) for KubebuilderConfigGen schema.

See [testdata](apis/v1alpha1/testdata) for examples of configuration options.
