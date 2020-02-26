# Deployment and Testing

Before we can test out our conversion, we'll need to enable them conversion in our CRD:

Kubebuilder generates Kubernetes manifests under the `config` directory with webhook
bits disabled.  To enable them, we need to:

- Enable `patches/webhook_in_<kind>.yaml` and
  `patches/cainjection_in_<kind>.yaml` in
  `config/crd/kustomization.yaml` file.

- Enable `../certmanager` and `../webhook` directories under the
  `bases` section in `config/default/kustomization.yaml` file.

- Enable `manager_webhook_patch.yaml` under the `patches` section
  in `config/default/kustomization.yaml` file.

- Enable all the vars under the `CERTMANAGER` section in
  `config/default/kustomization.yaml` file.

Additionally, we'll need to set the `CRD_OPTIONS` variable to just
`"crd"`, removing the `trivialVersions` option (this ensures that we
actually [generate validation for each version][ref-multiver], instead of
telling Kubernetes that they're the same):

```makefile
CRD_OPTIONS ?= "crd"
```

Now we have all our code changes and manifests in place, so let's deploy it to
the cluster and test it out.

You'll need [cert-manager](../cronjob-tutorial/cert-manager.md) installed
(version `0.9.0+`) unless you've got some other certificate management
solution.  The Kubebuilder team has tested the instructions in this tutorial
with
[0.9.0-alpha.0](https://github.com/jetstack/cert-manager/releases/tag/v0.9.0-alpha.0)
release.

Once all our ducks are in a row with certificates, we can run `make
install deploy` (as normal) to deploy all the bits (CRD,
controller-manager deployment) onto the cluster.

## Testing

Once all of the bits are up an running on the cluster with conversion enabled, we can test out our
conversion by requesting different versions.

We'll make a v2 version based on our v1 version (put it under `config/samples`)

```yaml
{{#include ./testdata/project/config/samples/batch_v2_cronjob.yaml}}
```

Then, we can create it on the cluster: 

```shell
kubectl apply -f config/samples/batch_v2_cronjob.yaml
```

If we've done everything correctly, it should create successfully,
and we should be able to fetch it using both the v2 resource

```shell
kubectl get cronjobs.v2.batch.tutorial.kubebuilder.io -o yaml
```

```yaml
{{#include ./testdata/project/config/samples/batch_v2_cronjob.yaml}}
```

and the v1 resource

```shell
kubectl get cronjobs.v1.batch.tutorial.kubebuilder.io -o yaml
```
```yaml
{{#include ./testdata/project/config/samples/batch_v1_cronjob.yaml}}
```

Both should be filled out, and look equivalent to our v2 and v1 samples,
respectively.  Notice that each has a different API version.

Finally, if we wait a bit, we should notice that our CronJob continues to
reconcile, even though our controller is written against our v1 API version.

## Troubleshooting 

[steps for troubleshooting](/TODO.md)

[ref-multiver]: /reference/generating-crd.md#multiple-versions "Generating CRDs: Multiple Versions"
