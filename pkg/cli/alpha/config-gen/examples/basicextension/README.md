# Kustomize Example

This directory contains an example of extending the kubebuilder config-gen by creating
a second `kustomize` transformer plugin which is composed with `config-gen`.

## Prerequisites 

- [kustomize example](../kustomize/README.md)
- KubebuilderConfigGen schema [types.go](../../types.go)

## Install the extension as a plugin

The extension is a separate plugin composed with the `config-gen` plugin.

```sh
# build the extension
go build -o ~/go/bin/basicextension .

# setup the extension plugin
export XDG_CONFIG_HOME=$HOME/.config
export KUBEBUILDER_PLUGIN=$XDG_CONFIG_HOME/kustomize/plugin/kubebuilder.sigs.k8s.io/kubebuilderconfiggenbasicextension
mkdir -p $KUBEBUILDER_PLUGIN
cat > $KUBEBUILDER_PLUGIN/KubebuilderConfigGenBasicExtension <<EOF
#!/bin/bash 
KUSTOMIZE_FUNCTION=true basicextension
EOF
chmod +x $KUBEBUILDER_PLUGIN/KubebuilderConfigGenBasicExtension
```

## Modify the `kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

transformers:
# this is the original kubebuilder configuration generator
- |-
  apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
  kind: KubebuilderConfigGen
  metadata:
    name: example
  spec:
    # contains kubebuilder project go code for generating crds
    projectDirectory: ../../testdata/project/...

    # image containing controller-manager
    image: example/simple:latest
# this is the extension which modifies the output from the KubebuilderConfigGen
- |-
  apiVersion: kubebuilder.sigs.k8s.io
  kind: KubebuilderConfigGenBasicExtension
  metadata:
    name: example
  replicas: 5
```

## Run kustomize

```sh
kustomize build --enable-alpha-plugins
```
