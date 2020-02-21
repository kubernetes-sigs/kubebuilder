# Kind Cluster

This only cover the basics to use a kind cluster. You can find more details at
[kind documentation](https://kind.sigs.k8s.io/).

## Installation

You can follow [this](https://kind.sigs.k8s.io/#installation-and-usage) to
install `kind`.

## Create a Cluster

You can simply create a `kind` cluster by

```bash
kind create cluster
```

To customize your cluster, you can provide additional configuration.
For example, the following is a sample `kind` configuration.

```yaml
{{#include ../cronjob-tutorial/testdata/project/hack/kind-config.yaml}}
```

Using the configuration above, run the following command will give you a k8s
1.14.2 cluster with 1 master and 3 workers.

```bash
kind create cluster --config hack/kind-config.yaml --image=kindest/node:v1.14.2
```

You can use `--image` flag to specify the cluster version you want, e.g.
`--image=kindest/node:v1.13.6`, the supported version are listed
[here](https://hub.docker.com/r/kindest/node/tags)

## Cheetsheet

- [Load a local image into a kind cluster](https://kind.sigs.k8s.io/docs/user/quick-start/#loading-an-image-into-your-cluster).

```bash
kind load docker-image your-image-name:your-tag
```

- Point `kubectl` to the kind cluster
```bash
kind export kubeconfig
```

- Delete a kind cluster
```bash
kind delete cluster
```
