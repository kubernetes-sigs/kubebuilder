# Using kind for development purposes and CI

## Why use kind

- **Fast Setup:** Launch a multi-node Kubernetes cluster locally in under a minute.
- **Quick Teardown:** Dismantle the cluster in just a few seconds, streamlining your development workflow.
- **Local Image Usage:** Deploy your container images directly without the need to push to a remote registry.
- **Lightweight and Efficient:** Kind is a minimalistic Kubernetes distribution, making it perfect for local development and CI/CD pipelines.

This only cover the basics to use a kind cluster. You can find more details at
[kind documentation](https://kind.sigs.k8s.io/).

## Installation

Follow the [kind installation instructions](https://kind.sigs.k8s.io/#installation-and-usage) to install `kind`.

## Create a cluster

You can simply create a `kind` cluster by

```bash
kind create cluster
```

To customize your cluster, you can provide additional configuration.
For example, the following is a sample `kind` configuration.

```yaml
{{#include ./kind-config.yaml}}
```

Using the configuration above, the following command gives you a k8s
v1.17.2 cluster with 1 control-plane node and 3 worker nodes.

```bash
kind create cluster --config hack/kind-config.yaml --image=kindest/node:v1.17.2
```

You can use the `--image` flag to specify the cluster version you want, e.g.
`--image=kindest/node:v1.17.2`. See the [kind node images](https://hub.docker.com/r/kindest/node/tags) for supported versions.

## Load docker image into the cluster

When developing with a local kind cluster, loading docker images to the cluster
is a very useful feature. You can avoid using a container registry.

```bash
kind load docker-image your-image-name:your-tag
```

See [Load a local image into a kind cluster](https://kind.sigs.k8s.io/docs/user/quick-start/#loading-an-image-into-your-cluster) for more information.

## Delete a cluster

```bash
kind delete cluster
```
