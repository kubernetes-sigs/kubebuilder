# Metrics
# 指标

By default, controller-runtime builds a global prometheus registry and
publishes a collection of performance metrics for each controller.
默认情况下，controller-runtime 会构建一个全局 prometheus 注册，并且会把一堆每个控制器的性能指标推送过去。

## Protecting the Metrics 指标保护

These metrics are protected by [kube-auth-proxy](https://github.com/brancz/kube-rbac-proxy)
by default if using kubebuilder. Kubebuilder v2.2.0+ scaffold a clusterrole which
can be found at `config/rbac/auth_proxy_client_clusterrole.yaml`.
如果使用了 kubebuilder [kube-auth-proxy](https://github.com/brancz/kube-rbac-proxy) 默认会保护这些指标。Kubebuilder v2.2.0+ 创建一个集群角色，在 `config/rbac/auth_proxy_client_clusterrole.yaml` 文件中配置。

You will need to grant permissions to your Prometheus server so that it can
scrape the protected metrics. To achieve that, you can create a
`clusterRoleBinding` to bind the `clusterRole` to the service account that your
Prometheus server uses.
你需要给你所有的 Prometheus 服务授权，以便它可以拿到这些被保护的指标。按这样的方式来授权，创建一个 `clusterRoleBinding` 把 `clusterRole` 绑定到一个你的 Prometheus 服务使用的服务账户上。

You can run the following kubectl command to create it. If using kubebuilder
`<project-prefix>` is the `namePrefix` field in `config/default/kustomization.yaml`.
可以运行下面的 kubectl 命令来创建它。如果你使用 kubebuilder，在`config/default/kustomization.yaml` 文件中`namePrefix` 字段是表示 `<project-prefix>`。

```bash
kubectl create clusterrolebinding metrics --clusterrole=<project-prefix>-metrics-reader --serviceaccount=<namespace>:<service-account-name>
```

## Exporting Metrics for Prometheus
## 给 Prometheus 导出指标
Follow the steps below to export the metrics using the Prometheus Operator:
按照下面的步骤来用 Prometheus Operator 导出指标：

1. Install Prometheus and Prometheus Operator.
We recommend using [kube-prometheus](https://github.com/coreos/kube-prometheus#installing)
in production if you don't have your own monitoring system.
If you are just experimenting, you can only install Prometheus and Prometheus Operator.
2. Uncomment the line `- ../prometheus` in the `config/default/kustomization.yaml`.
It creates the `ServiceMonitor` resource which enables exporting the metrics.
1. 安装 Prometheus 和 Prometheus Operator。如果没有自己的监控系统，推荐使用 [kube-prometheus](https://github.com/coreos/kube-prometheus#installing)。
2. 在 `config/default/kustomization.yaml` 配置文件中取消 `- ../prometheus` 这一行的注释。它会创建可以导出指标的 `ServiceMonitor` 资源。

```yaml
# [PROMETHEUS] To enable prometheus monitor, uncomment all sections with 'PROMETHEUS'.
- ../prometheus
```

Note that, when you install your project in the cluster, it will create the
`ServiceMonitor` to export the metrics. To check the ServiceMonitor,
run `kubectl get ServiceMonitor -n <project>-system`. See an example:
注意，当你在集群中安装你的项目时，它会创建一个 `ServiceMonitor` 来导出指标。为了检查 ServiceMonitor，可以运行 `kubectl get ServiceMonitor -n <project>-system`。看下面的例子：

```
$ kubectl get ServiceMonitor -n monitor-system
NAME                                         AGE
monitor-controller-manager-metrics-monitor   2m8s
```

Also, notice that the metrics are exported by default through port `8443`. In this way,
you are able to check the Prometheus metrics in its dashboard. To verify it, search
for the metrics exported from the namespace where the project is running
`{namespace="<project>-system"}`. See an example:
同样，要注意默认情况下是通过 `8443` 端口导出指标的。这种情况下，你可以在自己的 dashboard 中检查 Prometheus metrics。要检查这些指标，在项目运行的 `{namespace="<project>-system"}` 命名空间下搜索导出的指标。看下面的例子：

<img width="1680" alt="Screenshot 2019-10-02 at 13 07 13" src="https://user-images.githubusercontent.com/7708031/66042888-a497da80-e515-11e9-9d77-d8a9fc1159a5.png">

## Publishing Additional Metrics
## 发布额外的指标：

If you wish to publish additional metrics from your controllers, this
can be easily achieved by using the global registry from
`conoller-runtime/pkg/metrics`.

One way to achieve this is to declare your collectors as global variables and then register them using `init()`.

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
reconcile loop, and those metrics will be available for prometheus or
other openmetrics systems to scrape.
