# 控制器简介

控制器是 Kubernetes 的核心，也是任何 operator 的核心。 

控制器的工作是确保对于任何给定的对象，世界的实际状态（包括集群状态，以及潜在的外部状态，如 Kubelet 的运行容器或云提供商的负载均衡器）与对象中的期望状态相匹配。每个控制器专注于一个根Kind，但可能会与其他Kind交互。

我们把这个过程称为 **reconciling**。

在 controller-runtime 中，为特定种类实现 reconciling 的逻辑被称为[* Reconciler *](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile)。 Reconciler 接受一个对象的名称，并返回我们是否需要再次尝试（例如在错误或周期性控制器的情况下，如 HorizontalPodAutoscaler）。

{{#literatego ./testdata/emptycontroller.go}}

现在我们已经了解了 Reconcile 的基本结构，我们来补充一下 `CronJob`s 的逻辑。
