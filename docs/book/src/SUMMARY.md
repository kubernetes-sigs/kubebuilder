# Summary

[引言](./introduction.md)

[快速入门](./quick-start.md)

---

- [教程：构建 CronJob](cronjob-tutorial/cronjob-tutorial.md)

  - [基本项目中有什么？](./cronjob-tutorial/basic-project.md)
  - [每一个旅程都需要一个起点，每个程序都需要一个 main 入口](./cronjob-tutorial/empty-main.md)
  - [Groups、Versions 和 Kinds 之间的关系](./cronjob-tutorial/gvks.md)
  - [创建一个API](./cronjob-tutorial/new-api.md)
  - [设计一个API](./cronjob-tutorial/api-design.md)

      - [简要说明：剩下文件的作用？](./cronjob-tutorial/other-api-files.md)

  - [controller 中有什么？](./cronjob-tutorial/controller-overview.md)
  - [实现一个 controller](./cronjob-tutorial/controller-implementation.md)

    - [main 的修改？](./cronjob-tutorial/main-revisited.md)

  - [实现 defaulting/validating webhooks](./cronjob-tutorial/webhook-implementation.md)
  - [运行和部署 controller](./cronjob-tutorial/running.md)

    - [部署 cert manager](./cronjob-tutorial/cert-manager.md)
    - [部署 webhooks](./cronjob-tutorial/running-webhook.md)
  
  - [编写测试](./cronjob-tutorial/writing-tests.md)

  - [结语](./cronjob-tutorial/epilogue.md)

- [教程: Multi-Version API](./multiversion-tutorial/tutorial.md)

  - [Changing things up](./multiversion-tutorial/api-changes.md)
  - [Hubs, spokes, and other wheel metaphors](./multiversion-tutorial/conversion-concepts.md)
  - [实现 conversion](./multiversion-tutorial/conversion.md)

      - [配置 webhooks](./multiversion-tutorial/webhooks.md)

  - [Deployment 和 Testing](./multiversion-tutorial/deployment.md)

---

- [迁移](./migrations.md)

  - [Kubebuilder 从 v1 迁移到 v2 ](./migration/v1vsv2.md)

      - [迁移指南](./migration/guide.md)

  - [Single Group 到 Multi-Group](./migration/multi-group.md)

---

- [参考](./reference/reference.md)

  - [生成 CRDs](./reference/generating-crd.md)
  - [使用 Finalizers](./reference/using-finalizers.md)
  - [Kind 集群](reference/kind.md)
  - [webhook 是什么?](reference/webhook-overview.md)
    - [准入 webhook](reference/admission-webhook.md)
    - [核心类型的 Webhooks](reference/webhook-for-core-types.md)
  - [用于配置/代码生成的标记](./reference/markers.md)

      - [CRD 生成](./reference/markers/crd.md)
      - [CRD 验证](./reference/markers/crd-validation.md)
      - [CRD 处理](./reference/markers/crd-processing.md)
      - [Webhook](./reference/markers/webhook.md)
      - [Object/DeepCopy](./reference/markers/object.md)
      - [RBAC](./reference/markers/rbac.md)

  - [controller-gen 命令行界面](./reference/controller-gen.md)
  - [shell 自动补全](./reference/completion.md)
  - [制品包](./reference/artifacts.md)
  - [编写 controller 测试](./reference/writing-tests.md)

  - [在集成测试中使用 envtest](./reference/envtest.md)

  - [指标](./reference/metrics.md)

---

[附录: TODO 界面](./TODO.md)
