# 教程：多版本 API

大多数项目都是从一个 alpha API 开始的，这个 API 会随着发布版本的不同而变化。
然后，最终，大多数项目将会转向更稳定的版本。一旦你的 API 足够的稳定，你就不能够对它做破坏性的修改。
这就是 API 版本发挥作用的地方。

让我们对 `CronJob` API spec 做一些改变，确保我们的 CronJob 项目支持所有不同的版本。

如果你还没有准备好，确保你已经阅读过了基础的 [CronJob 教程](/cronjob-tutorial/cronjob-tutorial.md)。

<aside class="note">

<h1>跟随 vs 跳跃</h1>

注意本教程的大部分内容是由形成一个可运行的项目的 literate Go 文件生成的，并且放在了本书的下面源目录下 [docs/book/src/multiversion-tutorial/testdata/project][tutorial-source]。

[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/multiversion-tutorial/testdata/project

</aside>

<aside class="note warning">

<h1>最小 kubernetes 版本来了！</h1>

CRD 转换在 Kubernetes 1.13 版本（也就是说它是默认关闭的，需要通过一个 [feature gate][kube-feature-gates] 去打开）中作为一个 alpha 特性被引入支持，
在 Kubernetes 1.15 版本（也就是说它是默认打开的）中变成了 beta 版。

如果你的 Kubernetes 版本是 1.13-1.14，确保开启了特性开关。
如果你的 Kubernetes 版本是 1.12 或者更低，你将需要一个新的集群去使用转换。
查看 [Kind 说明](/reference/kind.md) 去了解如果设置一个多功能的集群。

</aside>

接下来，让我们弄清楚我们想要做哪些更改。

[kube-feature-gates]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/ "Kubernetes Feature Gates"
