# Running and deploying the controller

To test out the controller, we can run it locally against the cluster.
Before we do so, though, we'll need to install our CRDs, as per the [quick
start](/quick-start.md).  This will automatically update the YAML
manifests using controller-tools, if needed:

```bash
make install
```

Now that we've installed our CRDs, we can run the controller against our
cluster.  This will use whatever credentials that we connect to the
cluster with, so we don't need to worry about RBAC just yet.

<aside class="note"> 

<h1>Running webhooks locally</h1>

If you want to run the webhooks locally, you'll have to generate
certificates for serving the webhooks, and place them in the right
directory (`/tmp/k8s-webhook-server/serving-certs/tls.{crt,key}`, by
default).

If you're not running a local API server, you'll also need to figure out
how to proxy traffic from the remote cluster to your local webhook server.
For this reason, we generally reccomended disabling webhooks when doing
your local code-run-test cycle, as we do below.

</aside>

In a separate terminal, run

```bash
make run ENABLE_WEBHOOKS=false
```

You should see logs from the controller about starting up, but it won't do
anything just yet.

At this point, we need a CronJob to test with.  Let's write a sample to
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

Now that we know it's working, we can run it in the cluster. Stop the
`make run` invocation, and run

```bash
make docker-build docker-push IMG=<some-registry>/<project-name>:tag
make deploy IMG=<some-registry>/<project-name>:tag
```

If we list cronjobs again like we did before, we should see the controller
functioning again!
