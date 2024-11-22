# Metrics

By default, controller-runtime builds a global prometheus registry and
publishes [a collection of performance metrics](/reference/metrics-reference.md) for each controller.

<aside class="note warning">
<h1>IMPORTANT: If you are using `kube-rbac-proxy`</h1>

Please stop using the image `gcr.io/kubebuilder/kube-rbac-proxy` as soon as possible.
Your projects will be affected and may fail to work if the image cannot be pulled.

**Images provided under `gcr.io/kubebuilder/` will be unavailable from early 2025.**

- **Projects initialized with Kubebuilder versions `v3.14` or lower** utilize [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) to protect the metrics endpoint.
In this case, you might want to upgrade your project to the latest release or ensure that you have applied the same or similar code changes.

- **However, projects initialized with Kubebuilder versions `v4.1.0` or higher** have similar protection using `authn/authz`
enabled by default via Controller-Runtime's feature [WithAuthenticationAndAuthorization](https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization).

If you want to continue using [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) then you MUST change
your project to use the image from another source.

> For further information, see: [kubebuilder/discussions/3907](https://github.com/kubernetes-sigs/kubebuilder/discussions/3907)

</aside>

## Enabling the Metrics

First, you will need enable the Metrics by uncommenting the following line
in the file `config/default/kustomization.yaml`, see:

```yaml
# [METRICS] To enable the controller manager metrics service, uncomment the following line.
#- metrics_service.yaml
```

```yaml
# [METRICS] The following patch will enable the metrics endpoint. Ensure that you also protect this endpoint.
# More info: https://book.kubebuilder.io/reference/metrics
# If you want to expose the metric endpoint of your controller-manager uncomment the following line.
#- path: manager_metrics_patch.yaml
#  target:
#    kind: Deployment
```

Note that projects are scaffolded by default passing the flag `--metrics-bind-address=0`
to the manager to ensure that metrics are disabled. See the [controller-runtime
implementation](https://github.com/kubernetes-sigs/controller-runtime/blob/834905b07c7b5a78e86d21d764f7c2fdaa9602e0/pkg/metrics/server/server.go#L119-L122)
where the server creation will be skipped in this case.

## Protecting the Metrics

Unprotected metrics endpoints can expose valuable data to unauthorized users,
such as system performance, application behavior, and potentially confidential
operational metrics. This exposure can lead to security vulnerabilities
where an attacker could gain insights into the system's operation
and exploit weaknesses.

### By using Network Policy

NetworkPolicy acts as a basic firewall for pods within a Kubernetes cluster, controlling traffic
flow at the IP address or port level. However, it doesn't handle authentication (authn), authorization (authz),
or encryption directly like [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) solution.

### By exposing the metrics endpoint using HTTPS and CertManager

Integrating `cert-manager` with your metrics service can secure the endpoint via TLS encryption.

To modify your project setup to expose metrics using HTTPS with
the help of cert-manager, you'll need to change the configuration of both
the `Service` under `config/default/metrics_service.yaml` and
the `ServiceMonitor` under `config/prometheus/monitor.yaml` to use a secure HTTPS port
and ensure the necessary certificate is applied.

### By using Controller-Runtime new feature

Also, you might want to check the new feature added in Controller-Runtime via
the [pr](https://github.com/kubernetes-sigs/controller-runtime/pull/2407) which can handle authentication (`authn`),
authorization (`authz`) similar to [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) has been doing.

<aside class="note">
<h1>Changes required</h1>

After the [issue](https://github.com/kubernetes-sigs/controller-runtime/issues/2781) opened
for controller-runtime enhance this new feature to address the concerns raised we plan add an option
to use it in the default scaffold combined with cert-manager. For further information, please check the
[proposal](../../../../designs/discontinue_usage_of_kube_rbac_proxy.md).

</aside>

## Exporting Metrics for Prometheus

Follow the steps below to export the metrics using the Prometheus Operator:

1. Install Prometheus and Prometheus Operator.
   We recommend using [kube-prometheus](https://github.com/coreos/kube-prometheus#installing)
   in production if you don't have your own monitoring system.
   If you are just experimenting, you can only install Prometheus and Prometheus Operator.

2. Uncomment the line `- ../prometheus` in the `config/default/kustomization.yaml`.
   It creates the `ServiceMonitor` resource which enables exporting the metrics.

```yaml
# [PROMETHEUS] To enable prometheus monitor, uncomment all sections with 'PROMETHEUS'.
- ../prometheus
```

Note that, when you install your project in the cluster, it will create the
`ServiceMonitor` to export the metrics. To check the ServiceMonitor,
run `kubectl get ServiceMonitor -n <project>-system`. See an example:

```
$ kubectl get ServiceMonitor -n monitor-system
NAME                                         AGE
monitor-controller-manager-metrics-monitor   2m8s
```

<aside class="warning">
<h2>If you are using Prometheus Operator ensure that you have the required
permissions</h2>

If you are using Prometheus Operator, be aware that, by default, its RBAC
rules are only enabled for the `default` and `kube-system namespaces`. See its
guide to know [how to configure kube-prometheus to monitor other namespaces using the `.jsonnet` file](https://github.com/prometheus-operator/kube-prometheus/blob/main/docs/monitoring-other-namespaces.md).

Alternatively, you can give the Prometheus Operator permissions to monitor other namespaces using RBAC. See the Prometheus Operator
[Enable RBAC rules for Prometheus pods](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/user-guides/getting-started.md#enable-rbac-rules-for-prometheus-pods)
documentation to know how to enable the permissions on the namespace where the
`ServiceMonitor` and manager exist.
</aside>

Also, notice that the metrics are exported by default through port `8443`. In this way,
you are able to check the Prometheus metrics in its dashboard. To verify it, search
for the metrics exported from the namespace where the project is running
`{namespace="<project>-system"}`. See an example:

<img width="1680" alt="Screenshot 2019-10-02 at 13 07 13" src="https://user-images.githubusercontent.com/7708031/66042888-a497da80-e515-11e9-9d77-d8a9fc1159a5.png">

## Consuming the Metrics from other Pods.

Then, see an example to create a Pod using Curl to reach out the metrics:

```sh
kubectl run curl --restart=Never -n <namespace-name> --image=curlimages/curl:7.78.0 -- /bin/sh -c "curl -v http://<my-project>-controller-manager-metrics-service.<my-project-system>.svc.cluster.local:8080/metrics"
```

## Publishing Additional Metrics

If you wish to publish additional metrics from your controllers, this
can be easily achieved by using the global registry from
`controller-runtime/pkg/metrics`.

One way to achieve this is to declare your collectors as global variables and then register them using `init()` in the controller's package.

For example:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
    goobers = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "goobers_total",
            Help: "Number of goobers proccessed",
        },
    )
    gooberFailures = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "goober_failures_total",
            Help: "Number of failed goobers",
        },
    )
)

func init() {
    // Register custom metrics with the global prometheus registry
    metrics.Registry.MustRegister(goobers, gooberFailures)
}
```

You may then record metrics to those collectors from any part of your
reconcile loop. These metrics can be evaluated from anywhere in the operator code.

<aside class="note">
<h1>Enabling metrics in Prometheus UI</h1>

In order to publish metrics and view them on the Prometheus UI, the Prometheus instance would have to be configured to select the Service Monitor instance based on its labels.

</aside>

Those metrics will be available for prometheus or
other openmetrics systems to scrape.

![Screen Shot 2021-06-14 at 10 15 59 AM](https://user-images.githubusercontent.com/37827279/121932262-8843cd80-ccf9-11eb-9c8e-98d0eda80169.png)
