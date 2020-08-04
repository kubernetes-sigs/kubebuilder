# 部署和测试

在测试版本转换之前，我们需要在 CRD 中启用转换：

Kubebuilder 在 `config` 目录下生成禁用 webhook bits 的 Kubernetes 清单。要启用它们，我们需要：

- 在 `config/crd/kustomization.yaml` 文件启用 `patches/webhook_in_<kind>.yaml` 和
  `patches/cainjection_in_<kind>.yaml`。

- 在 `config/default/kustomization.yaml` 文件的 `bases` 部分下启用 `../certmanager` 和 `../webhook` 目录。

- 在 `config/default/kustomization.yaml` 文件的 `patches` 部分下启用 `manager_webhook_patch.yaml`。

- 在 `config/default/kustomization.yaml` 文件的 `CERTMANAGER` 部分下启用所有变量。

此外，我们需要将 `CRD_OPTIONS` 变量设置为 `"crd"`，删除 `trivialVersions` 选项（这确保我们实际 
[为每个版本生成验证][ref-multiver]，而不是告诉 Kubernetes 它们是一样的）：

```makefile
CRD_OPTIONS ?= "crd"
```

现在，我们已经完成了所有的代码更改和清单，让我们将其部署到集群并对其进行测试。

你需要安装 [cert-manager](../cronjob-tutorial/cert-manager.md)（`0.9.0+` 版本），
除非你有其他证书管理解决方案。Kubebuilder 团队已在
[0.9.0-alpha.0](https://github.com/jetstack/cert-manager/releases/tag/v0.9.0-alpha.0)
版本中测试了本教程中的指令。

一旦所有的证书准备妥当后, 我们就可以运行 `make install deploy`（和平常一样）将所有的 bits（CRD,
controller-manager deployment）部署到集群上。

## 测试

一旦启用了转换的所有 bits 都在群集上运行，我们就可以通过请求不同的版本来测试转换。

我们将基于 v1 版本制作 v2 版本（将其放在 `config/samples` 下）

```yaml
{{#include ./testdata/project/config/samples/batch_v2_cronjob.yaml}}
```

然后，我们可以在集群中创建它：

```shell
kubectl apply -f config/samples/batch_v2_cronjob.yaml
```

如果我们正确地完成了所有操作，那么它应该能够成功地创建，并且我们能够使用 v2 资源来获取它

```shell
kubectl get cronjobs.v2.batch.tutorial.kubebuilder.io -o yaml
```

```yaml
{{#include ./testdata/project/config/samples/batch_v2_cronjob.yaml}}
```

v1 资源

```shell
kubectl get cronjobs.v1.batch.tutorial.kubebuilder.io -o yaml
```
```yaml
{{#include ./testdata/project/config/samples/batch_v1_cronjob.yaml}}
```

两者都应填写，并分别看起来等同于的 v2 和 v1 示例。注意，每个都有不同的 API 版本。

最后，如果稍等片刻，我们应该注意到，即使我们的控制器是根据 v1 API 版本编写的，我们的 CronJob 仍在继续协调。

<aside class="note">

<h1>kubectl 和首选版本</h1>

当我们从 Go 代码访问 API 类型时，我们会通过使用该版本的 Go 类型（例如 `batchv2.CronJob`）来请求特定版本。

你可能已经注意到，上面对 kubectl 的调用与我们通常所做的看起来有些不同 —— 即，它指定了一个 
*group-version-resource* 而不是一个资源。

当我们运行 `kubectl get cronjob` 时, kubectl 需要弄清楚映射到哪个
group-version-resource。为此，它使用 *discovery API* 来找出 `cronjob` 资源的首选版本。对于 CRD，
这或多或少是最新的稳定版本（具体细节请参阅 [CRD 文档][CRD-version-pref])。

随着我们对 CronJob 的更新, 意味着 `kubectl get cronjob` 将获取 `batch/v2` group-version。

如果我们想指定一个确切的版本，可以像上面一样使用 `kubectl get resource.version.group`。

***你应该始终在脚本中使用完全合格的 group-version-resource 语法***。
`kubectl get resource` 是为人类、有自我意识的机器人和其他能够理解新版本的众生设计的。
`kubectl get resource.version.group` 用于其他的一切。

</aside>

## 故障排除 

[故障排除的步骤](/TODO.md)

[ref-multiver]: /reference/generating-crd.md#multiple-versions "Generating CRDs: Multiple Versions"

[crd-version-pref]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#version-priority "Versions in CustomResourceDefinitions"
