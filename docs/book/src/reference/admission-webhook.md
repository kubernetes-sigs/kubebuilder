# Admission Webhooks

Admission webhook 是 HTTP 的回调，它可以接受 adminssion 请求，处理它们并且返回 adminssion 响应。

Kubernetes 提供了下面几种类型的 admission webhook：

- **变更 Adminssion Webhook**
这种类型的 webhook 会在对象创建或是更新且没有存储前改变操作对象，然后才存储。它可以用于资源请求中的默认字段，比如在 Deployment 中没有被用户制定的字段。它可以用于注入 sidecar 容器。

- **验证 Admission Webhook**
这种类型的 webhook 会在对象创建或是更新且没有存储前验证操作对象，然后才存储。它可以有比纯基于 schema 验证更加复杂的验证。比如：交叉字段验证和 pod 镜像白名单。

默认情况下 apiserver 自己没有对 webhook 进行认证。然而，如果你想认证客户端，你可以配置 apiserver 使用基本授权，持有 token，或者证书对 webhook 进行认证。
详细的步骤可以查看[这里](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers)。
