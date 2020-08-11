# Kind 集群

这篇文章只涉及到使用一个 kind 集群的基础。你可以在 [kind 文档](https://kind.sigs.k8s.io/) 中找到更详细的介绍。

## 安装

你可以按照[这个文档](https://kind.sigs.k8s.io/#installation-and-usage)来安装 `kind`。

## 创建一个集群

你可以简单的通过下面的命令来创建一个 `kind` 集群。
```bash
kind create cluster
```

要定制你的集群，你可以提供额外的配置。比如，下面的例子是一个 `kind` 配置的例子。

```yaml
{{#include ../cronjob-tutorial/testdata/project/hack/kind-config.yaml}}
```

使用上面的配置来运行下面的命令会创建一个 k8s v1.17.2 的集群，包含了 1 个 master 节点和 3 个 worker 节点。

```bash
kind create cluster --config hack/kind-config.yaml --image=kindest/node:v1.17.2
```

你可以使用 `--image` 标记来指定你想创建集群的版本，比如：`--image=kindest/node:v1.17.2`，能支持的版本在[这里](https://hub.docker.com/r/kindest/node/tags)。

## 加载 Docker 镜像到集群

当使用一个本地 kind 集群进行开发时，加载 docker 镜像到集群中是一个非常有用的功能。可以让你避免使用容器仓库。

- [加载一个本地镜像到一个 kind 集群](https://kind.sigs.k8s.io/docs/user/quick-start/#loading-an-image-into-your-cluster).

```bash
kind load docker-image your-image-name:your-tag
```

## 删除一个集群

- 删除一个 kind 集群
```bash
kind delete cluster
```
