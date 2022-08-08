# Metrics

By default, controller-runtime builds a global prometheus registry and
publishes [a collection of performance metrics](/reference/metrics-reference.md) for each controller.

## Protecting the Metrics

These metrics are protected by [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy)
by default if using kubebuilder. Kubebuilder v2.2.0+ scaffold a clusterrole which
can be found at `config/rbac/auth_proxy_client_clusterrole.yaml`.

You will need to grant permissions to your Prometheus server so that it can
scrape the protected metrics. To achieve that, you can create a
`clusterRoleBinding` to bind the `clusterRole` to the service account that your
Prometheus server uses. If you are using [kube-prometheus](https://github.com/prometheus-operator/kube-prometheus),
this cluster binding already exists.

You can either run the following command, or apply the example yaml file provided below to create `clusterRoleBinding`.

If using kubebuilder
`<project-prefix>` is the `namePrefix` field in `config/default/kustomization.yaml`.

```bash
kubectl create clusterrolebinding metrics --clusterrole=<project-prefix>-metrics-reader --serviceaccount=<namespace>:<service-account-name>
```

You can also apply the following `ClusterRoleBinding`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus-k8s-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus-k8s-role
subjects:
  - kind: ServiceAccount
    name: <prometheus-service-account>
    namespace: <prometheus-service-account-namespace>
```

The `prometheus-k8s-role` referenced here should provide the necessary permissions to allow prometheus scrape metrics from operator pods.

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
<h2>Enabling metrics in Prometheus UI</h1>
  
In order to publish metrics and view them on the Prometheus UI, the Prometheus instance would have to be configured to select the Service Monitor instance based on its labels.

</aside>

Those metrics will be available for prometheus or
other openmetrics systems to scrape.

![Screen Shot 2021-06-14 at 10 15 59 AM](https://user-images.githubusercontent.com/37827279/121932262-8843cd80-ccf9-11eb-9c8e-98d0eda80169.png)
