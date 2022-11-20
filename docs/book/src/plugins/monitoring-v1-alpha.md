# Monitoring Plugin (`monitoring/v1-alpha`)

The Monitoring plugin is an optional plugin that can be used to scaffold Prometheus based monitoring, will provide best practices and tooling for monitoring requirements and help with standardizing the way monitoring is implemented in operators.

<aside class="note">
<h1>Examples</h1>

You can check its default scaffold by looking at the `project-v3-with-monitoring` projects under the [testdata][testdata] directory on the root directory of the Kubebuilder project.

</aside>

## When to use it ?

- If you are looking to observe the metrics exported by [controller metrics][controller-metrics] and collected by Prometheus via [Monitoring][monitoring].

## How to use it ?

### Prerequisites:

- Your project must be using [controller-runtime][controller-runtime] to expose the metrics via the [controller default metrics][controller-metrics] and they need to be collected by Prometheus.
- Access to [Prometheus][prometheus].
  - Prometheus should have an endpoint exposed. (For `prometheus-operator`, this is similar as: http://prometheus-k8s.monitoring.svc:9090 )
  - The endpoint is ready to/already become the datasource of your Monitoring. See [Add a data source](https://monitoring.com/docs/monitoring/latest/datasources/add-a-data-source/)
<aside class="note">

Check the [metrics][reference-metrics-doc] to know how to enable the metrics for your projects scaffold with Kubebuilder.

See that in the [config/prometheus][kustomize-plugin] you will find the ServiceMonitor to enable the metrics in the default endpoint `/metrics`.

</aside>

### Basic Usage

The monitoring plugin is attached to the `init` subcommand and the `edit` subcommand:

```sh
# Initialize a new project with monitoring plugin
kubebuilder init --plugins monitoring.kubebuilder.io/v1-alpha

# Enable monitoring plugin to an existing project
kubebuilder edit --plugins monitoring.kubebuilder.io/v1-alpha
```

The plugin will create a new directory and scaffold the GO files under it (i.e. `monitoring/metrics/metrics.go`).


[controller-metrics]: https://book.kubebuilder.io/reference/metrics-reference.html
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[monitoring]: https://monitoring.com/docs/monitoring/next/
[monitoring-docs]: https://monitoring.com/docs/monitoring/latest/dashboards/export-import/#import-dashboard
[kustomize-plugin]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/testdata/project-v3/config/prometheus/monitor.yaml
[kube-prometheus]: https://github.com/prometheus-operator/kube-prometheus
[plugin-implementation]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/optional/monitoring/alphav1
[prometheus]: https://prometheus.io/docs/introduction/overview/
[prom-operator]: https://prometheus-operator.dev/docs/prologue/introduction/
[reference-metrics-doc]: https://book.kubebuilder.io/reference/metrics.html#exporting-metrics-for-prometheus
[servicemonitor]: https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/user-guides/getting-started.md#related-resources
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata
