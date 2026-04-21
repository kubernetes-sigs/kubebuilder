# Deployment and testing

Before testing out the conversion, enable them in the CRD:

Kubebuilder generates Kubernetes manifests under the `config` directory with webhook
bits disabled. To enable them:

- Enable `patches/webhook_in_<kind>.yaml` and
  `patches/cainjection_in_<kind>.yaml` in
  `config/crd/kustomization.yaml` file.

- Enable `../certmanager` and `../webhook` directories under the
  `bases` section in `config/default/kustomization.yaml` file.

- Enable all the vars under the `CERTMANAGER` section in
  `config/default/kustomization.yaml` file.

Additionally, if present in the Makefile, set the `CRD_OPTIONS` variable to just
`"crd"`, removing the `trivialVersions` option (this ensures that it
actually [generates validation for each version][ref-multiver], instead of
telling Kubernetes that they are the same):

```makefile
CRD_OPTIONS ?= "crd"
```

Now that all code changes and manifests are in place, deploy it to
the cluster and test it out.

You'll need [cert-manager](../cronjob-tutorial/cert-manager.md) installed
(version `0.9.0+`) unless you have got some other certificate management
solution.  The Kubebuilder team has tested the instructions in this tutorial
with
[0.9.0-alpha.0](https://github.com/cert-manager/cert-manager/releases/tag/v0.9.0-alpha.0)
release.

Once all ducks are in a row with certificates, run `make
install deploy` (as normal) to deploy all the bits (CRD,
controller-manager deployment) onto the cluster.

## Testing

Once all of the bits are up and running on the cluster with conversion enabled, test out the
conversion by requesting different versions.

Make a v2 version based on the v1 version (put it under `config/samples`)

```yaml
{{#include ./testdata/project/config/samples/batch_v2_cronjob.yaml}}
```

Then, create it on the cluster:

```shell
kubectl apply -f config/samples/batch_v2_cronjob.yaml
```

If you have done everything correctly, it should create successfully,
and you should be able to fetch it using both the v2 resource

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

Both should be filled out, and look equivalent to the v2 and v1 samples,
respectively.  Notice that each has a different API version.

Finally, if you wait a bit, you should notice that the CronJob continues to
reconcile, even though the controller is written against the v1 API version.

<aside class="note" role="note">

<p class="note-title">kubectl and Preferred Versions</p>

When accessing API types from Go code, you ask for a specific version
by using that version's Go type (e.g. `batchv2.CronJob`).

You might've noticed that the above invocations of kubectl looked
a little different from the usual approach -- namely, they specify
a *group-version-resource*, instead of just a resource.

When you write `kubectl get cronjob`, kubectl needs to figure out which
group-version-resource that maps to.  To do this, it uses the *discovery
API* to figure out the preferred version of the `cronjob` resource.  For
CRDs, this is more-or-less the latest stable version (see the [CRD
docs][CRD-version-pref] for specific details).

With the updates to CronJob, this means that `kubectl get cronjob` fetches
the `batch/v2` group-version.

To specify an exact version, use `kubectl get
resource.version.group`, as shown above.

***You should always use fully-qualified group-version-resource syntax in
scripts***.  `kubectl get resource` is for humans, self-aware robots, and
other sentient beings that can figure out new versions.  `kubectl get
resource.version.group` is for everything else.

</aside>

[ref-multiver]: /reference/generating-crd.md#multiple-versions "Generating CRDs: Multiple Versions"

[crd-version-pref]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#version-priority "Versions in CustomResourceDefinitions"
