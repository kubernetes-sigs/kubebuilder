# 编写控制器测试示例

测试 Kubernetes 控制器是一个大的课题，kubebuilder 为您生成的样板测试文件相当少。

为了带您了解 Kubebuilder 生成的控制器的集成测试模式，我们将重新阅读一遍我们在第一篇教程中构建的 CronJob，并为它编写一个简单的测试。

基本的方法是，在生成的 `suite_test.go` 文件中，您将用 envtest 去创建一个本地 Kubernetes API 服务端，并实例化和运行你的控制器，然后编写附加的 `*_test.go` 文件并用 [Ginko](http://onsi.github.io/ginkgo) 去测试它。

如果您想修改您的 envtest 集群的配置方式，请查看 [编写和运行集成测试](/reference/testing/envtest.md) 和 [`envtest docs`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest?tab=doc) 章节。

## 测试环境配置

{{#literatego ../cronjob-tutorial/testdata/project/controllers/suite_test.go}}

## 测试控制器行为

{{#literatego ../cronjob-tutorial/testdata/project/controllers/cronjob_controller_test.go}}

上面状态更新的示例演示了一个带有下游对象的自定义 Kind 的通用测试策略。到此，希望您已经学到了下列用于测试控制器行为的方法：

* 配置你的控制器运行在 envtest 集群上
* 为创建测试对象编写测试示例
* 隔离对对象的更改，以测试特定的控制器行为

## 高级示例

还有更多使用 envtest 来严格测试控制器行为的例子。包括：

* Azure Databricks Operator: 仔细阅读 [`suite_test.go`](https://github.com/microsoft/azure-databricks-operator/blob/0f722a710fea06b86ecdccd9455336ca712bf775/controllers/suite_test.go) 以及那个目录下所有的 `*_test.go` 文件 [例如这个](https://github.com/microsoft/azure-databricks-operator/blob/0f722a710fea06b86ecdccd9455336ca712bf775/controllers/secretscope_controller_test.go)。
