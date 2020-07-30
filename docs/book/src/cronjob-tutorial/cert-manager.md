# 部署 cert manager

我们建议使用 [cert manager](https://github.com/jetstack/cert-manager) 为 webhook 服务器提供证书。只要其他解决方案将证书放在期望的位置，也将会起作用。

你可以按照 [cert manager 文档](https://docs.cert-manager.io/en/latest/getting-started/install/kubernetes.html) 进行安装。

Cert manager 还有一个叫做 CA 注入器的组件，该组件负责将 CA 捆绑注入到 Mutating|ValidatingWebhookConfiguration 中。

为此，你需要在 Mutating|ValidatingWebhookConfiguration 对象中使用带有 key 为 `cert-manager.io/inject-ca-from` 的注释。
注释的值应指向现有的证书 CR 实例，格式为 `<certificate-namespace>/<certificate-name>`。

这是我们用于注释 Mutating|ValidatingWebhookConfiguration 对象的 [kustomize](https://github.com/kubernetes-sigs/kustomize) patch。
```yaml
{{#include ./testdata/project/config/default/webhookcainjection_patch.yaml}}
```
