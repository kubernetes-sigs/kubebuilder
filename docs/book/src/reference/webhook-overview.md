# Webhook

Webhooks 是一种以阻塞方式发送的信息请求。实现 webhooks 的 web 应用程序将在特定事件发生时向其他应用程序发送 HTTP 请求。

在 kubernetes 中，有下面三种 webhook：[admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks)，
[authorization webhook](https://kubernetes.io/docs/reference/access-authn-authz/webhook/) 和 [CRD conversion webhook](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion)。

在 [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/webhook?tab=doc) 库中，我们支持 admission webhooks 和 CRD conversion webhooks。

Kubernetes 在 1.9 版本中（该特性进入 beta 版时）支持这些动态 admission webhooks。

Kubernetes 在 1.15 版本（该特性进入 beta 版时）支持 conversion webhook。
