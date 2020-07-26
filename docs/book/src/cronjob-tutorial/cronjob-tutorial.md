# 教程：构建 CronJob

太多的教程一开始都是以一些非常复杂的设置开始，或者是实现一些玩具应用，让你了解了基本的知识，然后又在更复杂的东西上停滞不前。相反地，本教程将会带你了解 Kubebuilder 的（几乎）全部复杂功能，从简单的功能开始，到全部功能。

让我们假装厌倦了 Kubernetes 中的 CronJob 控制器的非 Kubebuilder 实现繁重的维护工作（当然，这有点小题大做），我们想用 KubeBuilder 来重写它。

**CronJob** 控制器的工作是定期在 Kubernetes 集群上运行一次性任务。它是以 **Job** 控制器为基础实现的，而 **Job** 控制器的任务是运行一次性的任务，确保它们完成。

与其试图一开始解决重写 Job 控制器的问题，我们先看看如何与外部类型进行交互。

<aside class="note">

<h1>渐进 vs 跳进</h1>

注意，本教程的大部分内容都是从书的源目录下的 Go 文件中生成的：[docs/book/src/cronjob-tutorial/testdata][tutorial-source]。完整的、可运行的项目在 [project][tutorial-project-source] 中，而中间文件则直接在 [testdata][tutorial-source] 目录。

[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata

[tutorial-project-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project

</aside>

## 创建项目

正如 [快速入门](../quick-start.md) 中所介绍的那样，我们需要创建一个新的项目。确保你已经 [安装 Kubebuilder](../quick-start.md#安装)，然后再创建一个新项目。

```bash
# 我们将使用 tutorial.kubebuilder.io 域，
# 所以所有的 API 组将是<group>.tutorial.kubebuilder.io.
kubebuilder init --domain tutorial.kubebuilder.io
```

现在我们已经创建了一个项目，让我们来看看 Kubebuilder 为我们初始化了哪些组件。....
