# GVK 介绍

在我们开始讲 API 之前，我们应该先介绍一下 Kubernetes 中 API 相关的术语。

当我们在 Kubernetes 中谈论 API 时，我们经常会使用 4 个术语：**groups** 、**versions**、**kinds** 和 **resources**。

## 组和版本

Kubernetes 中的 **API 组**简单来说就是相关功能的集合。每个组都有一个或多个**版本**，顾名思义，它允许我们随着时间的推移改变 **API** 的职责。

## 类型和资源

每个 API 组-版本包含一个或多个 API 类型，我们称之为 **Kinds**。虽然一个 Kind 可以在不同版本之间改变表单内容，但每个表单必须能够以某种方式存储其他表单的所有数据（我们可以将数据存储在字段中，或者在注释中）。 这意味着，使用旧的 API 版本不会导致新的数据丢失或损坏。更多 API 信息，请参阅 [Kubernetes API 指南](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md)。

你也会偶尔听到提到 **resources**。 resources（资源） 只是 API 中的一个 Kind 的使用方式。通常情况下，Kind 和 resources 之间有一个一对一的映射。 例如，`pods` 资源对应于 `Pod` 种类。但是有时，同一类型可能由多个资源返回。例如，`Scale` Kind 是由所有 `scale` 子资源返回的，如 `deployments/scale` 或 `replicasets/scale`。这就是允许 Kubernetes HorizontalPodAutoscaler(HPA) 与不同资源交互的原因。然而，使用 CRD，每个 Kind 都将对应一个 resources。

注意：resources 总是用小写，按照惯例是 Kind 的小写形式。

> GVK = Group Version Kind
> GVR = Group Version Resources

## 那么，这些术语如何对应到 Golang 中的实现呢？

当我们在一个特定的群组版本 (Group-Version) 中提到一个 Kind 时，我们会把它称为 **GroupVersionKind**，简称 GVK。与 资源 (resources) 和 GVR 一样，我们很快就会看到，每个 GVK 对应 Golang 代码中的到对应生成代码中的 Go type。

现在我们理解了这些术语，我们就可以**真正**地创建我们的 API！

## Scheme 是什么？

我们之前看到的 `Scheme` 是一种追踪 Go Type 的方法，它对应于给定的 GVK（不要被它吓倒 [godocs](https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#Scheme)）。

例如，假设我们将 `"tutorial.kubebuilder.io/api/v1".CronJob{}` 类型放置在 `batch.tutorial.kubebuilder.io/v1` API 组中（也就是说它有 `CronJob` Kind)。

然后，我们便可以在 API server 给定的 json 数据构造一个新的 `&CronJob{}`。

```json
{
    "kind": "CronJob",
    "apiVersion": "batch.tutorial.kubebuilder.io/v1",
    ...
}
```

或当我们在一次变更中去更新或提交 `&CronJob{}` 时，查找正确的组版本。
