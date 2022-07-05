# Grafana Plugin (`grafana/v1-alpha`)

The Grafana plugin is an optional plugin that can be used to scaffold Grafana Dashboards to allow you to check out the default metrics which are exported by projects using [controller-runtime][controller-runtime].

<aside class="note">
<h1>Examples</h1>

You can check its default scaffold by looking at the `project-v3-with-grafana` projects under the [testdata][testdata] directory on the root directory of the Kubebuilder project.

</aside>

## When to use it ?

- If you are looking to observe the metrics exported by [controller metrics][controller-metrics] and collected by Prometheus via [Grafana][grafana].

## How to use it ?

### Prerequisites:

- Your project must be using [controller-runtime][controller-runtime] to expose the metrics via the [controller default metrics][controller-metrics] and they need to be collected by Prometheus.
- Access to [Prometheus][prometheus].
  - Prometheus should have an endpoint exposed. (For `prometheus-operator`, this is similar as: http://prometheus-k8s.monitoring.svc:9090 )
  - The endpoint is ready to/already become the datasource of your Grafana. See [Add a data source](https://grafana.com/docs/grafana/latest/datasources/add-a-data-source/)
- Access to [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/installation/). Make sure you have:
  - [Dashboard edit permission](https://grafana.com/docs/grafana/next/administration/roles-and-permissions/#dashboard-permissions)
  - Prometheus Data source
    ![pre](https://user-images.githubusercontent.com/18136486/176119794-f6d69b0b-93f0-4f9e-a53c-daf9f77dadae.gif)

<aside class="note">

Check the [metrics][reference-metrics-doc] to know how to enable the metrics for your projects scaffold with Kubebuilder.

See that in the [config/prometheus][kustomize-plugin] you will find the ServiceMonitor to enable the metrics in the default endpoint `/metrics`.

</aside>

### Basic Usage

The Grafana plugin is attached to the `init` subcommand and the `edit` subcommand:

```sh
# Initialize a new project with grafana plugin
kubebuilder init --plugins grafana.kubebuilder.io/v1-alpha

# Enable grafana plugin to an existing project
kubebuilder edit --plugins grafana.kubebuilder.io/v1-alpha
```

The plugin will create a new directory and scaffold the JSON files under it (i.e. `grafana/controller-runtime-metrics.json`).

#### Show case:

See an example of how to use the plugin in your project:

![output](https://user-images.githubusercontent.com/18136486/175382307-9a6c3b8b-6cc7-4339-b221-2539d0fec042.gif)

#### Now, let's check how to use the Grafana dashboards

1. Copy the JSON file
2. Visit `<your-grafana-url>/dashboard/import` to [import a new dashboard](https://grafana.com/docs/grafana/latest/dashboards/export-import/#import-dashboard).
3. Paste the JSON content to `Import via panel json`, then press `Load` button
   <img width="644" alt="Screen Shot 2022-06-28 at 3 40 22 AM" src="https://user-images.githubusercontent.com/18136486/176121955-1c4aec9c-0ba4-4271-9767-e8d1726d9d9a.png">
4. Select the data source for Prometheus metrics
   <img width="633" alt="Screen Shot 2022-06-28 at 3 41 26 AM" src="https://user-images.githubusercontent.com/18136486/176122261-e3eab5b0-9fc4-45fc-a68c-d9ce1cfe96ee.png">
5. Once the json is imported in Grafana, the dashboard is ready.

### Grafana Dashboard

#### Controller Runtime Reconciliation total & errors

- Metrics:
  - controller_runtime_reconcile_total
  - controller_runtime_reconcile_errors_total
- Query:
  - sum(rate(controller_runtime_reconcile_total{job="$job"}[5m])) by (instance, pod)
  - sum(rate(controller_runtime_reconcile_errors_total{job="$job"}[5m])) by (instance, pod)
- Description:
  - Per-second rate of total reconciliation as measured over the last 5 minutes
  - Per-second rate of reconciliation errors as measured over the last 5 minutes
- Sample: <img width="1430" src="https://user-images.githubusercontent.com/18136486/176122555-f3493658-6c99-4ad6-a9b7-63d85620d370.png">

#### CPU & Memory Usage

- Metrics:
  - process_cpu_seconds_total
  - process_resident_memory_bytes
- Query:
  - rate(process_cpu_seconds_total{job="$job", namespace="$namespace", pod="$pod"}[5m]) \* 100
  - process_resident_memory_bytes{job="$job", namespace="$namespace", pod="$pod"}
- Description:
  - Per-second rate of CPU usage as measured over the last 5 minutes
  - Allocated Memory for the running controller
- Sample: <img width="1381" src="https://user-images.githubusercontent.com/18136486/177239808-7d94b17d-692c-4166-8875-6d9332e05bcb.png">

## Subcommands

The Grafana plugin implements the following subcommands:

- edit (`$ kubebuilder edit [OPTIONS]`)

- init (`$ kubebuilder init [OPTIONS]`)

## Affected files

The following scaffolds will be created or updated by this plugin:

- `grafana/*.json`

## Further resources

- Refer to a sample of `servicemonitor` provided by [kustomize plugin][kustomize-plugin]
- Check the [plugin implementation][plugin-implementation]
- [Grafana Docs][grafana-docs] of importing JSON file
- The usage of servicemonitor by [Prometheus Operator][servicemonitor]

[controller-metrics]: https://book.kubebuilder.io/reference/metrics-reference.html
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[grafana]: https://grafana.com/docs/grafana/next/
[grafana-docs]: https://grafana.com/docs/grafana/latest/dashboards/export-import/#import-dashboard
[kustomize-plugin]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/testdata/project-v3/config/prometheus/monitor.yaml
[kube-prometheus]: https://github.com/prometheus-operator/kube-prometheus
[plugin-implementation]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/optional/grafana/alphav1
[prometheus]: https://prometheus.io/docs/introduction/overview/
[prom-operator]: https://prometheus-operator.dev/docs/prologue/introduction/
[reference-metrics-doc]: https://book.kubebuilder.io/reference/metrics.html#exporting-metrics-for-prometheus
[servicemonitor]: https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/user-guides/getting-started.md#related-resources
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata
