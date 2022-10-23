# Deploying Admission Webhooks

## Kind Cluster

It is recommended to develop your webhook with a
[kind](../reference/kind.md) cluster for faster iteration.
Why?

- You can bring up a multi-node cluster locally within 1 minute.
- You can tear it down in seconds.
- You don't need to push your images to remote registry.

## cert-manager

You need to follow [this](./cert-manager.md) to install the cert-manager bundle.

## Build your image

Run the following command to build your image locally.

```bash
make docker-build docker-push IMG=<some-registry>/<project-name>:tag
```

You don't need to push the image to a remote container registry if you are using
a kind cluster. You can directly load your local image to your specified kind cluster:

```bash
kind load docker-image <your-image-name>:tag --name <your-kind-cluster-name>
```

## Deploy Webhooks

You need to enable the webhook and cert manager configuration through kustomize.
`config/default/kustomization.yaml` should now look like the following:

```yaml
{{#include ./testdata/project/config/default/kustomization.yaml}}
```

And `config/crd/kustomization.yaml` should now look like the following:

```yaml
{{#include ./testdata/project/config/crd/kustomization.yaml}}
```

Now you can deploy it to your cluster by

```bash
make deploy IMG=<some-registry>/<project-name>:tag
```

Wait a while till the webhook pod comes up and the certificates are provisioned.
It usually completes within 1 minute.

Now you can create a valid CronJob to test your webhooks. The creation should
successfully go through.

```bash
kubectl create -f config/samples/batch_v1_cronjob.yaml
```

You can also try to create an invalid CronJob (e.g. use an ill-formatted
schedule field). You should see a creation failure with a validation error.

<aside class="note warning">

<h1>The Bootstrapping Problem</h1>

If you are deploying a webhook for pods in the same cluster, be
careful about the bootstrapping problem, since the creation request of the
webhook pod would be sent to the webhook pod itself, which hasn't come up yet.

To make it work, you can either use [namespaceSelector] if your kubernetes
version is 1.9+ or use [objectSelector] if your kubernetes version is 1.15+ to
skip itself.

</aside>

[namespaceSelector]: https://github.com/kubernetes/api/blob/kubernetes-1.14.5/admissionregistration/v1beta1/types.go#L189-L233
[objectSelector]: https://github.com/kubernetes/api/blob/kubernetes-1.15.2/admissionregistration/v1beta1/types.go#L262-L274
