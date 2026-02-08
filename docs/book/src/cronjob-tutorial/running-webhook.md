# Deploying Admission Webhooks

## cert-manager

You need to follow [this](./cert-manager.md) to install the cert-manager bundle.

## Build your image

Run the following command to build your image locally.

```bash
make docker-build docker-push IMG=<some-registry>/<project-name>:tag
```

<aside class="note">
<h4> Using Kind </h4>

Consider incorporating Kind into your workflow for a faster, more efficient local development and CI experience.
Note that, if you're using a Kind cluster, there's no need to push your image to a remote container registry.
You can directly load your local image into your specified Kind cluster:

```bash
kind load docker-image <your-image-name>:tag --name <your-kind-cluster-name>
```

To know more, see: [Using Kind For Development Purposes and CI](./../reference/kind.md)

</aside>


## Deploy Webhooks

You need to enable the webhook and cert manager configuration through kustomize.
`config/default/kustomization.yaml` should have the following webhook-related sections uncommented:

**Resources** - Add the webhook and cert-manager resources:
```yaml
{{#include ./testdata/project/config/default/kustomization.yaml:webhook-resources}}
```

**Patches** - Add the webhook manager patch:
```yaml
{{#include ./testdata/project/config/default/kustomization.yaml:webhook-patch}}
```

**Replacements** - Add the webhook certificate replacements:
```yaml
{{#include ./testdata/project/config/default/kustomization.yaml:webhook-replacements}}
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

<aside class="warning">
<h3>The Bootstrapping Problem</h3>

When you deploy a webhook into the same cluster that it will validate, you can run into a *bootstrapping issue*:
the webhook may try to validate the creation of its own Pod before it’s actually running.
This can block the webhook from ever starting.

To avoid this, make sure the webhook **ignores its own resources**.
You can do this in one of two ways:

- **[namespaceSelector]** – label the namespace where the webhook runs and configure the webhook to skip it.
- **[objectSelector]** – label the webhook’s own Pods or Deployments and exclude those objects directly.

See the complete step-by-step guide: **[Webhook Bootstrap Problem](../reference/webhook-bootstrap-problem.md)**

</aside>

[namespaceSelector]: https://github.com/kubernetes/api/blob/kubernetes-1.14.5/admissionregistration/v1beta1/types.go#L189-L233
[objectSelector]: https://github.com/kubernetes/api/blob/kubernetes-1.15.2/admissionregistration/v1beta1/types.go#L262-L274
