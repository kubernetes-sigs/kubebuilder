**注意：** 着急的读者可以直接跳到这里 [Quick
Start](quick-start.md)。

**若在使用 Kubebuilder v1? 请查看 [legacy
documentation](https://book-v1.book.kubebuilder.io)**

## 文档适合哪些人看

#### Kubernetes 的使用者

Kubernetes 的使用者将通过学习其 APIs 是如何设计和实现的，从而更深入地了解 Kubernetes 。这本书将教读者如何开发自己的 Kubernetes APIs 以及如何设计核心 Kubernetes API 的原理。

包括：

- Kubernetes APIs 和 Resources 的构造
- APIs 版本控制语义
- 故障自愈
- 垃圾回收和 Finalizers
- 声明式与命令式 APIs
- 基于 Level-Based 与 Edge-Base APIs
- Resources 与 Subresources

#### Kubernetes API 开发者

API 扩展开发者将学习实现标准的 Kubernetes API 原理和概念，以及用于快速执行的简单工具和库。这本书涵盖了开发人员通常会遇到的陷阱和误区。

包括：

- 如何用一个 reconciliation 方法处理多个 events
- 如何配置定期执行 reconciliation 方法
- *即将到来的*
    - 何时使用 lister cache 与 live lookups
    - 垃圾回收与 Finalizers
    - 如何使用 Declarative 与 Webhook Validation
    - 如何实现 API 版本控制

## 贡献

如果您想为本书或代码做出贡献，请先阅读我们的[贡献](https://github.com/cloudnativeto/kubebuilder/tree/zh/docs/book)准则。

## 资源

* 代码仓库：[sigs.k8s.io/kubebuilder](https://sigs.k8s.io/kubebuilder)

* Slack 沟通群：[#kubebuilder](http://slack.k8s.io/#kubebuilder)

* 谷歌沟通群：
  [kubebuilder@googlegroups.com](https://groups.google.com/forum/#!forum/kubebuilder)  
* 云原生社区：
  [https://cloudnative.to](https://cloudnative.tor)  
