# Running and deploying the controller

### Optional
If opting to make any changes to the API definitions, then before proceeding,
generate the manifests like CRs or CRDs with
```bash
make manifests
```

To test out the controller, run it locally against the cluster.
Before doing so, install the CRDs, as per the [quick
start](/quick-start.md).  This will automatically update the YAML
manifests using controller-tools, if needed:

```bash
make install
```

<aside class="note" role="note">

<p class="note-title">Too long annotations error</p>

If you encounter errors when applying the CRDs, due to `metadata.annotations` exceeding the
262144 bytes limit, please refer to the specific entry in the [FAQ section](/faq#the-error-too-long-must-have-at-most-262144-bytes-is-faced-when-i-run-make-install-to-apply-the-crd-manifests-how-to-solve-it-why-this-error-is-faced).

</aside>

Now that you've installed the CRDs, run the controller against the
cluster.  This uses whatever credentials you use to connect to the
cluster, so you don't need to worry about RBAC just yet.

<aside class="note" role="note">

<p class="note-title">Running webhooks locally</p>

If you want to run the webhooks locally, you'll have to generate
certificates for serving the webhooks, and place them in the right
directory (`/tmp/k8s-webhook-server/serving-certs/tls.{crt,key}`, by
default).

If you're not running a local API server, you'll also need to figure out
how to proxy traffic from the remote cluster to your local webhook server.
For this reason, disable webhooks when doing
your local code-run-test cycle, as shown below.

</aside>

In a separate terminal, run

```bash
export ENABLE_WEBHOOKS=false
make run
```

You should see logs from the controller about starting up, but it won't do
anything just yet.

At this point, you need a CronJob to test with.  Write a sample to
`config/samples/batch_v1_cronjob.yaml`, and use that:

```yaml
{{#include ./testdata/project/config/samples/batch_v1_cronjob.yaml}}
```

```bash
kubectl create -f config/samples/batch_v1_cronjob.yaml
```

At this point, you should see a flurry of activity.  If you watch the
changes, you should see your cronjob running, and updating status:

```bash
kubectl get cronjob.batch.tutorial.kubebuilder.io -o yaml
kubectl get job
```

Now that you know it's working, run it in the cluster. Stop the
`make run` invocation, and run

```bash
make docker-build docker-push IMG=<some-registry>/<project-name>:tag
make deploy IMG=<some-registry>/<project-name>:tag
```

<aside class="note" role="note">
<p class="note-title">Registry Permission</p>

This image ought to be published in the personal registry you specified. And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don't work.

Consider incorporating Kind into your workflow for a faster, more efficient local development and CI experience.
Note that, if you're using a Kind cluster, there's no need to push your image to a remote container registry.
You can directly load your local image into your specified Kind cluster:

```bash
kind load docker-image <your-image-name>:tag --name <your-kind-cluster-name>
```

To know more, see: [Using Kind For Development Purposes and CI](./../reference/kind.md)

<p class="note-title">RBAC errors</p>

If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin. See [Prerequisites for using Kubernetes RBAC on GKE cluster v1.11.x and older][pre-rbc-gke] which may be your case.

</aside>

If you list cronjobs again like before, you should see the controller
functioning again!

[pre-rbc-gke]: https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control#iam-rolebinding-bootstrap