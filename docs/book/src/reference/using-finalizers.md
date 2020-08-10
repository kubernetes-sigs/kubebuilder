# 使用 Finalizers

`Finalizers` 允许控制器实现异步预删除挂钩。假设你为 API 类型的每个对象创建了一个外部资源（例如存储桶），并且想要从 Kubernetes 中删除对象同时删除关联的外部资源，则可以使用 finalizers 来实现。

您可以在[Kubernetes参考文档中](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#finalizers)阅读有关 finalizers 的更多信息。以下部分演示了如何在控制器的 `Reconcile` 方法中注册和触发预删除挂钩。

要注意的关键点是 finalizers 使对象上的“删除”成为设置删除时间戳的“更新”。对象上存在删除时间戳记表明该对象正在被删除。否则，在没有 finalizers 的情况下，删除将显示为协调，缓存中缺少该对象。

注意：
- 如果未删除对象并且未注册 finalizers ，则添加 finalizers 并在 Kubernetes 中更新对象。
- 如果要删除对象，但 finalizers 列表中仍存在 finalizers ，请执行预删除逻辑并移除 finalizers 并更新对象。
- 确保预删除逻辑是幂等的。

{{#literatego ../cronjob-tutorial/testdata/finalizer_example.go}}

