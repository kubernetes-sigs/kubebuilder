# 部署认可的 Webhooks

## Kind Cluster

建议使用 [kind](../reference/kind.md) 集群来更快速的开发 webhook。
为什么呢？

- 你可以在 1 分钟内本地启动有多个节点的集群。
- 你可以在几秒中内关闭它。
- 你不需要把你的镜像推送到远程仓库。

## 证书管理

你要遵循[这个](./cert-manager.md)来安装证书管理器。

## 构建你的镜像

运行下面的命令来本地构建你的镜像。

```bash
make docker-build
```

如果你使用的是 kind 集群，那么你不需要把镜像推送到远程容器仓库。你可以直接加载你本地的镜像到你的 kind 集群：

```bash
kind load docker-image your-image-name:your-tag
```

## 部署 Webhooks

你需要通过启用 webhook 和证书管理配置。`config/default/kustomization.yaml` 应该看起来是这样子的：

```yaml
{{#include ./testdata/project/config/default/kustomization.yaml}}
```

现在你可以通过下面的命令把它部署到你的集群中了：

```bash
make deploy IMG=<some-registry>/<project-name>:tag
```

等一会儿，webhook 的 pod 启动并且也提供了证书认证。这个过程通常需要 1 分钟。

现在你可以创建一个有效的 CronJob 来测试你的 webhook。这个过程应该会顺利通过的。

```bash
kubectl create -f config/samples/batch_v1_cronjob.yaml
```

你也能试着创建一个无效的 CronJob（比如使用一个非法格式的调度字段）。你应该可以看到创建失败并且有验证错误信息。

<aside class="note warning">

<h1>启动问题</h1>

如果你在同一个集群为 pod 部署了一个 webhook，要留意以启动问题，因为创建 webhook pod 的请求可能会被发送到 webhook pod 它自己，而它自己还没有启动起来。

为了让它能正常工作，你可以使用选择器来跳过它自己，如果你的 kubernetes 版本是 1.9+ 的可以使用 [namespaceSelector]，如果你的 kubernetes 版本是 1.15+ 的使用 [objectSelector]。

</aside>

[namespaceSelector]: https://github.com/kubernetes/api/blob/kubernetes-1.14.5/admissionregistration/v1beta1/types.go#L189-L233
[objectSelector]: https://github.com/kubernetes/api/blob/kubernetes-1.15.2/admissionregistration/v1beta1/types.go#L262-L274
