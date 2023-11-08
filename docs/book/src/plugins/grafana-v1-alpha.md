# Grafana Plugin (`grafana/v1-alpha`)

The Grafana plugin is an optional plugin that can be used to scaffold Grafana Dashboards to allow you to check out the default metrics which are exported by projects using [controller-runtime][controller-runtime].

<aside class="note">
<h1>Examples</h1>

You can check its default scaffold by looking at the `project-v3-with-metrics` projects under the [testdata][testdata] directory on the root directory of the Kubebuilder project.

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
   <img width="644" src="https://user-images.githubusercontent.com/18136486/176121955-1c4aec9c-0ba4-4271-9767-e8d1726d9d9a.png">
4. Select the data source for Prometheus metrics
   <img width="633" src="https://user-images.githubusercontent.com/18136486/176122261-e3eab5b0-9fc4-45fc-a68c-d9ce1cfe96ee.png">
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
- Sample: <img width="912" src="https://user-images.githubusercontent.com/18136486/176122555-f3493658-6c99-4ad6-a9b7-63d85620d370.png">

#### Controller CPU & Memory Usage

- Metrics:
  - process_cpu_seconds_total
  - process_resident_memory_bytes
- Query:
  - rate(process_cpu_seconds_total{job="$job", namespace="$namespace", pod="$pod"}[5m]) \* 100
  - process_resident_memory_bytes{job="$job", namespace="$namespace", pod="$pod"}
- Description:
  - Per-second rate of CPU usage as measured over the last 5 minutes
  - Allocated Memory for the running controller
- Sample: <img width="912" src="https://user-images.githubusercontent.com/18136486/177239808-7d94b17d-692c-4166-8875-6d9332e05bcb.png">

#### Seconds of P50/90/99 Items Stay in Work Queue

- Metrics
  - workqueue_queue_duration_seconds_bucket
- Query:
  - histogram_quantile(0.50, sum(rate(workqueue_queue_duration_seconds_bucket{job="$job", namespace="$namespace"}[5m])) by (instance, name, le))
- Description
  - Seconds an item stays in workqueue before being requested.
- Sample: <img width="912" src="https://user-images.githubusercontent.com/18136486/180359126-452b2a0f-a511-4ae3-844f-231d13cd27f8.png">

#### Seconds of P50/90/99 Items Processed in Work Queue

- Metrics
  - workqueue_work_duration_seconds_bucket
- Query:
  - histogram_quantile(0.50, sum(rate(workqueue_work_duration_seconds_bucket{job="$job", namespace="$namespace"}[5m])) by (instance, name, le))
- Description
  - Seconds of processing an item from workqueue takes.
- Sample: <img width="912" src="https://user-images.githubusercontent.com/18136486/180359617-b7a59552-1e40-44f9-999f-4feb2584b2dd.png">

#### Add Rate in Work Queue

- Metrics
  - workqueue_adds_total
- Query:
  - sum(rate(workqueue_adds_total{job="$job", namespace="$namespace"}[5m])) by (instance, name)
- Description
  - Per-second rate of items added to work queue
- Sample: <img width="912" src="https://user-images.githubusercontent.com/18136486/180360073-698b6f77-a2c4-4a95-8313-fd8745ad472f.png">

#### Retries Rate in Work Queue

- Metrics
  - workqueue_retries_total
- Query:
  - sum(rate(workqueue_retries_total{job="$job", namespace="$namespace"}[5m])) by (instance, name)
- Description
  - Per-second rate of retries handled by workqueue
- Sample: <img width="912" src="https://user-images.githubusercontent.com/18136486/180360101-411c81e9-d54e-4b21-bbb0-e3f94fcf48cb.png">

#### Number of Workers in Use

- Metrics
  - controller_runtime_active_workers
- Query:
  - controller_runtime_active_workers{job="$job", namespace="$namespace"}
- Description
  - The number of active controller workers
- Sample: <img width="912" src="https://github.com/kubernetes-sigs/kubebuilder/assets/18136486/288db1b5-e2d8-48ea-9aae-30de7eeca277">

#### WorkQueue Depth

- Metrics
  - workqueue_depth
- Query:
  - workqueue_depth{job="$job", namespace="$namespace"}
- Description
  - Current depth of workqueue
- Sample: <img width="912" src="https://github.com/kubernetes-sigs/kubebuilder/assets/18136486/34f14df4-0428-460e-9658-01dd3d34aade">

#### Unfinished Seconds

- Metrics
  - workqueue_unfinished_work_seconds
- Query:
  - rate(workqueue_unfinished_work_seconds{job="$job", namespace="$namespace"}[5m])
- Description
  - How many seconds of work has done that is in progress and hasn't been observed by work_duration.
- Sample: <img width="912" src="https://github.com/kubernetes-sigs/kubebuilder/assets/18136486/081727c0-9531-4f7a-9649-87723ebc773f">

### Visualize Custom Metrics

The Grafana plugin supports scaffolding manifests for custom metrics.

#### Generate Config Template

When the plugin is triggered for the first time, `grafana/custom-metrics/config.yaml` is generated.

```yaml
---
customMetrics:
#  - metric: # Raw custom metric (required)
#    type:   # Metric type: counter/gauge/histogram (required)
#    expr:   # Prom_ql for the metric (optional)
#    unit:   # Unit of measurement, examples: s,none,bytes,percent,etc. (optional)
```

#### Add Custom Metrics to Config

You can enter multiple custom metrics in the file. For each element, you need to specify the `metric` and its `type`.
The Grafana plugin can automatically generate `expr` for visualization.
Alternatively, you can provide `expr` and the plugin will use the specified one directly.

```yaml
---
customMetrics:
  - metric: memcached_operator_reconcile_total # Raw custom metric (required)
    type: counter # Metric type: counter/gauge/histogram (required)
    unit: none
  - metric: memcached_operator_reconcile_time_seconds_bucket
    type: histogram
```

#### Scaffold Manifest

Once `config.yaml` is configured, you can run `kubebuilder edit --plugins grafana.kubebuilder.io/v1-alpha` again.
This time, the plugin will generate `grafana/custom-metrics/custom-metrics-dashboard.json`, which can be imported to Grafana UI.

#### Show case:

See an example of how to visualize your custom metrics:

![output2](https://user-images.githubusercontent.com/18136486/186933170-d2e0de71-e079-4d1b-906a-99a549d66ebf.gif)

## Subcommands

The Grafana plugin implements the following subcommands:

- edit (`$ kubebuilder edit [OPTIONS]`)

- init (`$ kubebuilder init [OPTIONS]`)

## Affected files

The following scaffolds will be created or updated by this plugin:

- `grafana/*.json`

## Further resources

- Check out [video to show how it works](https://youtu.be/-w_JjcV8jXc)
- Checkout the [video to show how the custom metrics feature works](https://youtu.be/x_0FHta2HXc)
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
