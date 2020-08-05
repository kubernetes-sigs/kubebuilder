# 结语

至此，我们已经实现了一个功能比较完备的 Cronjob controller 了，利用了 KubeBuilder 的大部分特性，而且用
 envtest 写了 controller 的测试。

如果你想要知道更多，可以看 [Multi-Version
Tutorial](/multiversion-tutorial/tutorial.md)，学习如何给项目添加新API。

另外，你可以自己尝试完成以下步骤--稍后我们会有一个教程。

- `kubectl get` [添加额外的列打印][printer-columns]

[printer-columns]: /reference/generating-crd.md#additional-printer-columns
