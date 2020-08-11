# 参考

  - [生成 CRDs](generating-crd.md)
  - [使用 Finalizers](using-finalizers.md)
    Finalizers 是一种在资源被从 Kubernetes 集群删除之前，对资源执行的任何与资源相关的自定义逻辑的机制。
  - [Kind 集群](kind.md)
  - [什么是 webhook?](webhook-overview.md)
    Webhooks 是一种 HTTP 回调，在 k8s 中有三种 webhooks：1）准入 webhook
    2）CRD 转换 webhook 3）授权 webhook
    - [准入 webhook](admission-webhook.md)
      准入 webhooks 是对 mutating 和 validating 资源在准入 API server 之前的 HTTP 回调。
  - [Config/Code 生成标记](markers.md)

      - [CRD 生成](markers/crd.md)
      - [CRD 验证](markers/crd-validation.md)
      - [Webhook](markers/webhook.md)
      - [Object/DeepCopy](markers/object.md)
      - [RBAC](markers/rbac.md)

  - [controller-gen 命令行](controller-gen.md)
  - [补全](completion.md)
  - [制品](artifacts.md)
  - [编写 controller 测试](writing-tests.md)
  - [指标](metrics.md)
