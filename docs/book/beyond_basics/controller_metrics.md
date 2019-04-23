Monitoring
----------

Kubebuilder projects use [controller-runtime](https://sigs.k8s.io/controller-runtime)
to implement controllers and admission webhooks. `controller-runtime` instruments several key metrics
related to controllers and webhooks by default using [kubernetes instrumentation guidelines](https://github.com/kubernetes/community/blob/master/contributors/devel/instrumentation.md).
and makes them available via HTTP endpoint in [prometheus metric format](https://prometheus.io/docs/instrumenting/exposition_formats/).

Following metrics are instrumented by default:

 - Total number of reconcilation errors per controller
 - Length of reconcile queue per controller
 - Reconcilation latency
 - Usual resource metrics such as CPU, memory usage, file descriptor usage
 - Go runtime metrics such as number of Go routines, GC duration

{% panel style="info", title="Metrics support" %}
Please note that metrics support has been added in controller-runtime `0.1.8+`
release which is the default version for Kubebuilder `1.0.6+` releases. So if your
project was created using `1.0.5 or older` kubebuilder, then update the
controller-runtime dependencies to `0.1.8 or higher`.
{% endpanel %}

To quickly examine metrics in your development environment, you can run the
following:

```sh
# launch manager
$ make run

# in another terminal, access the metrics

$ curl http://localhost:8080/metrics
# HELP controller_runtime_reconcile_errors_total Total number of reconcile errors per controller
# TYPE controller_runtime_reconcile_errors_total counter
controller_runtime_reconcile_errors_total{controller="mysql-controller"} 10
# HELP controller_runtime_reconcile_queue_length Length of reconcile queue per controller
# TYPE controller_runtime_reconcile_queue_length gauge
controller_runtime_reconcile_queue_length{controller="mysql-controller"} 0
# HELP controller_runtime_reconcile_time_seconds Length of time per reconcile per controller
# TYPE controller_runtime_reconcile_time_seconds histogram
controller_runtime_reconcile_time_seconds_bucket{controller="mysql-controller",le="0.005"} 10
controller_runtime_reconcile_time_seconds_bucket{controller="mysql-controller",le="0.01"} 10
controller_runtime_reconcile_time_seconds_bucket{controller="mysql-controller",le="0.025"} 10
controller_runtime_reconcile_time_seconds_bucket{controller="mysql-controller",le="10"} 10
controller_runtime_reconcile_time_seconds_bucket{controller="mysql-controller",le="+Inf"} 10
controller_runtime_reconcile_time_seconds_sum{controller="mysql-controller"} 2.3416e-05
controller_runtime_reconcile_time_seconds_count{controller="mysql-controller"} 10
# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 7.69e-05
go_gc_duration_seconds{quantile="0.25"} 0.0001225
go_gc_duration_seconds{quantile="0.5"} 0.000124351
go_gc_duration_seconds{quantile="0.75"} 0.000236344
go_gc_duration_seconds{quantile="1"} 0.000262102
go_gc_duration_seconds_sum 0.000822197
go_gc_duration_seconds_count 5
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 39
# HELP go_info Information about the Go environment.
# TYPE go_info gauge
go_info{version="go1.9.4"} 1
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
.....
....

```

Is the metrics endpoint protected ?
-----------------------------------
Yes. By default, kubebuilder generated YAML manifests (under `config/` dir)
ensures that the access to metrics endpoint is authenticated and authorized using
an [auth proxy](https://github.com/brancz/kube-rbac-proxy) which is deployed as
sidecar container in the manager pod. You can read more details about the
auth proxy based approach [here](https://brancz.com/2018/02/27/using-kube-rbac-proxy-to-secure-kubernetes-workloads/).

If you want to disable the auth proxy, which is not recommended, you can follow
the instructions in the Kustomization file located in `config/kustomization.yaml`

If your project was created using `1.0.5 or older` kubebuilder, you need to modify
the following files as show in [PR #513](https://github.com/kubernetes-sigs/kubebuilder/pull/513/commits/a227e6457b581d4f1f1d79f16ca9b7baad8f38c0#diff-8e690fe6cdd7ce6beeb28f97e7423964).
- cmd/manager/main.go
- config/kustomization.yaml
- config/default/manager_auth_proxy_patch.yaml
- config/rbac/auth_proxy_role.yaml
- config/rbac/auth_proxy_role_binding.yaml
- config/rbac/auth_proxy_service.yaml

How do I configure Prometheus Server to access the metrics?
-----------------------------------------------------------

Kubebuilder generated manifests for manager have annotations such as
`prometheus.io/scrape`, `prometheus.io/path` on the metrics service so
that it can be easily discovered by the prometheus server deployed in your
kubernetes cluster.

Assuming auth is enabled, which is by default, you will have to add the
following to the job which is configured to scrap kubernetes service endpoints.

```yaml
tls_config:
    insecure_skip_verify: true

bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token

```
