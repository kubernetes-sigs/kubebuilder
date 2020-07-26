# 一个 kubebuilder 项目有哪些组件？

在脚手架生成新项目时，Kubebebuilder 为我们提供了一些基本的模板。

## 创建基础组件

首先是基本的项目文件初始化，为项目构建做好准备。

<details> <summary>`go.mod`: 我们的项目的 Go mod 配置文件，记录依赖库信息。</summary>

```go
{{#include ./testdata/project/go.mod}}
```
</details>

<details><summary>`Makefile`: 用于控制器构建和部署的 Makefile 文件</summary>

```makefile
{{#include ./testdata/project/Makefile}}
```
</details>

<details><summary>`PROJECT`: 用于生成组件的 Kubebuilder 元数据</summary>

```yaml
{{#include ./testdata/project/PROJECT}}
```
</details>

## 启动配置

我们还可以在 [`config/`](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project/config) 目录下获得启动配置。现在，它只包含了在集群上启动控制器所需的 [Kustomize](https://sigs.k8s.io/kustomize) YAML 定义，但一旦我们开始编写控制器，它还将包含我们的 CustomResourceDefinitions(CRD) 、RBAC 配置和 WebhookConfigurations 。

[`config/default`](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project/config/default) 在标准配置中包含 [Kustomize base](https://github.com/kubernetes-sigs/kubebuilder/blob/master/docs/book/src/cronjob-tutorial/testdata/project/config/default/kustomization.yaml) ，它用于启动控制器。

其他每个目录都包含一个不同的配置，重构为自己的基础。

- [`config/manager`](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project/config/manager): 在集群中以 pod 的形式启动控制器

- [`config/rbac`](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project/config/rbac): 在自己的账户下运行控制器所需的权限

## 入口函数

最后，当然也是最重要的一点，生成项目的入口函数：`main.go`。接下来我们看看它。.....
