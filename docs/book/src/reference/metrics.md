# 指标

默认情况下，controller-runtime 会构建一个全局 prometheus 注册，并且会为每个控制器发布一堆性能指标。

## 指标保护

如果使用了 kubebuilder [kube-auth-proxy](https://github.com/brancz/kube-rbac-proxy) 默认会保护这些指标。Kubebuilder v2.2.0+ 会创建一个集群角色，它在 `config/rbac/auth_proxy_client_clusterrole.yaml` 文件中配置。

你需要给你所有的 Prometheus 服务授权，以便它可以拿到这些被保护的指标。为实现授权，你可以创建一个 `clusterRoleBinding` 把 `clusterRole` 绑定到一个你x的 Prometheus 服务使用的账户上。

可以运行下面的 kubectl 命令来创建它。如果你使用 kubebuilder，在`config/default/kustomization.yaml` 文件中 `namePrefix` 字段是 `<project-prefix>`。

```bash
kubectl create clusterrolebinding metrics --clusterrole=<project-prefix>-metrics-reader --serviceaccount=<namespace>:<service-account-name>
```

## 给 Prometheus 导出指标
按照下面的步骤来用 Prometheus Operator 导出指标：

1. 安装 Prometheus 和 Prometheus Operator。如果没有自己的监控系统，推荐使用 [kube-prometheus](https://github.com/coreos/kube-prometheus#installing)。如果你经验丰富，那么可以只安装 Prometheus 和 Prometheus Operator。
2. 在 `config/default/kustomization.yaml` 配置文件中取消 `- ../prometheus` 这一行的注释。它会创建可以导出指标的 `ServiceMonitor` 资源。

```yaml
# [PROMETHEUS] To enable prometheus monitor, uncomment all sections with 'PROMETHEUS'.
- ../prometheus
```

注意，当你在集群中安装你的项目时，它会创建一个 `ServiceMonitor` 来导出指标。为了检查 ServiceMonitor，可以运行 `kubectl get ServiceMonitor -n <project>-system`。看下面的例子：

```
$ kubectl get ServiceMonitor -n monitor-system
NAME                                         AGE
monitor-controller-manager-metrics-monitor   2m8s
```

同样，要注意默认情况下是通过 `8443` 端口导出指标的。这种情况下，你可以在自己的 dashboard 中检查 Prometheus metrics。要检查这些指标，在项目运行的 `{namespace="<project>-system"}` 命名空间下搜索导出的指标。看下面的例子：

<img width="1680" alt="Screenshot 2019-10-02 at 13 07 13" src="https://user-images.githubusercontent.com/7708031/66042888-a497da80-e515-11e9-9d77-d8a9fc1159a5.png">

## 发布额外的指标：

如果你想从你的控制器发布额外的指标，可以通过在 `conoller-runtime/pkg/metrics` 中使用全局注册的方式来轻松做到。

实现发布额外指标的一种方式是把收集器声明为全局变量，并且使用 `init()` 来注册。

例如：

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

然后就可以从你的接收循环部分中记录这些指标到收集器中了，并且这些指标就可以被 prometheus 或者其它开放指标系统来抓取了。
