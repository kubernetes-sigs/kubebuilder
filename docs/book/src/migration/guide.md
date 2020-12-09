# 从 v1 版本迁移到 v2 版本


在继续后续操作之前要确保你了解[Kubebuilder v1 版本和 v2 版本之间的不同](./v1vsv2.md)。

请确保你根据[安装指导](/quick-start.md#installation)安装了迁移所需的组件。

迁移 v1 项目的推荐方法是创建一个新的 v2 项目，然后将 API 和 reconciliation 代码拷贝过来。
这种转换就像一个原生的 v2 项目。然后，在某些情况下，是可以进行就地升级的（比如，复用 v1 项目
的层级结构，升级 controller-runtime 和 controller-tools）。


让我们来看一个[v1 项目例子][v1-project]并且将其迁移至 Kubebuilder v2。最后，我们会有一些
东西看起来像[v2 项目例子][v2-project]。

## 准备


我们将需要明确 group，vresion，kind 和 domain 都是什么。

让我们看看我们目前的 v1 项目的结构:

```
pkg/
├── apis
│   ├── addtoscheme_batch_v1.go
│   ├── apis.go
│   └── batch
│       ├── group.go
│       └── v1
│           ├── cronjob_types.go
│           ├── cronjob_types_test.go
│           ├── doc.go
│           ├── register.go
│           ├── v1_suite_test.go
│           └── zz_generated.deepcopy.go
├── controller
└── webhook
```


我们所有的 API 信息都存放在 `pkg/apis/batch` 目录下，因此我们可以在那儿查找以
获取我们需要知道的东西。

在 `cronjob_types.go` 中，我们可以找到

```go
type CronJob struct {...}
```


在 `register.go` 中，我们可以找到

```go
SchemeGroupVersion = schema.GroupVersion{Group: "batch.tutorial.kubebuilder.io", Version: "v1"}
```


把这些集合起来，我们能够得到 kind 是 `CronJob`，group-version 是 `batch.tutorial.kubebuilder.io/v1`。

## 初始化一个 v2 项目


现在，我们需要初始化一个 v2 项目。然而，在此之前，如果在 `gopath` 中我们没有找到 go 模块，
那么我们需要初始化一个新的 go 模块。

```bash
go mod init tutorial.kubebuilder.io/project
```

接下来，我们可以用 kubebuilder 来完成项目的初始化：

```bash
kubebuilder init --domain tutorial.kubebuilder.io
```

## 迁移 APIs 和 Controllers


接下来，我们将重新生成 API 类型和 controllers。因为两者我们都需要，当向我们询问
我们想要生成哪些部分时，我们需要输入 yes 来同时生成 API 和 controller。

```bash
kubebuilder create api --group batch --version v1 --kind CronJob
```

如果你在使用多 group，迁移的时候就需要一些手动工作了。更多详细信息可以
查看[这个](/migration/multi-group.md)。

### 迁移 APIs


现在，让我们把 API 的定义部分从 `pkg/apis/batch/v1/cronjob_types.go` 拷贝
至 `api/v1/cronjob_types.go`。我们仅仅需要拷贝 `Spec` 和 `Status` 字段的实现部分。


我们可以用 `+kubebuilder:object:root=true` 来替代 `+k8s:deepcopy-gen:interfaces=...` 标记
（这个在[Kubebuilder 中废弃了](/reference/markers/object.md)）。


我们不再需要以下的标记了（他们不再被使用，是 KubeBuilder 一些老版本的遗留产物）。 

```go
// +genclient
// +k8s:openapi-gen=true
```


我们的 API 类型看起来应该像下面这样:

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// CronJob is the Schema for the cronjobs API
type CronJob struct {...}

// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob
type CronJobList struct {...}
```

### 迁移 Controllers


现在，让我们将 controller reconciler 的代码从 `pkg/controller/cronjob/cronjob_controller.go`
迁移至 `controllers/cronjob_controller.go`。


我们需要拷贝
- `ReconcileCronJob` 结构体中的字段到 `CronJobReconciler`。
- `Reconcile` 函数的内容。
- [rbac 相关标记](/reference/markers/rbac.md)到一个新文件。
- `func add(mgr manager.Manager, r reconcile.Reconciler) error` 下的代码
到 `func SetupWithManager`。

## 迁移 Webhooks

如果你还没有webhook，你可以跳过此章节。

### Core 类型和外部 CRDs 的 Webhooks 


如果你在使用 Kubernetes core 类型（比如 Pods），或者不属于你的一个外部 CRD 的 webhooks，
你可以参考[内置类型的 controller-runtime 例子][builtin-type-example]，然后做一些类似的事情。
Kubebuilder 不会为这种情形生成太多，但是你可以用 controller-runtime 中的一些库。

### 为我们的 CRDs 自动生成 Webhooks


现在让我们为我们的 CRD (CronJob) 自动生成 webhooks。我们需要运行以下命令并指定 `--defaulting`
 和 `--programmatic-validation` 参数（因为我们的测试项目使用 defaulting 和 validating webhooks）：

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation
```

取决于有多少 CRDs 需要 webhooks，我们或许需要为不同的 Group-Version-Kinds 多次执行上述的命令。


现在，我们需要为每一个 webhook 来拷贝逻辑。对于 validating webhooks，我们需要将
`pkg/default_server/cronjob/validating/cronjob_create_handler.go` 文件中 `func validatingCronJobFn`
内容拷贝至 `api/v1/cronjob_webhook.go` 文件中的 `func ValidateCreate`，对于 `update` 来说也是一样的。


类似的，我们将 `func mutatingCronJobFn` 拷贝至 `func Default`。

### Webhook 标记


当自动生成 webhooks 时，Kubebuilder v2 添加了以下标记：

```
// 这些是 v2 标记

// 这些是关于 mutating webhook 的
// +kubebuilder:webhook:path=/mutate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=true,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=mcronjob.kb.io

...

// 这些是关于 validating webhook 的
// +kubebuilder:webhook:path=/validate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=false,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=vcronjob.kb.io
```

默认的 verbs 是 `verbs=create;update`。我们需要确保 `verbs` 和我们所需要的是一致的。
比如，如果我们仅仅想验证 creation，那么我们就需要将 verbs 改成 `verbs=create`。

我们也需要确保 `failure-policy` 是不变的。

如下所示的标记将不再被使用（因为他们和自部署证书配置有关，这些在 v2 被移除了）：

```go
// v1 markers
// +kubebuilder:webhook:port=9876,cert-dir=/tmp/cert
// +kubebuilder:webhook:service=test-system:webhook-service,selector=app:webhook-server
// +kubebuilder:webhook:secret=test-system:webhook-server-secret
// +kubebuilder:webhook:mutating-webhook-config-name=test-mutating-webhook-cfg
// +kubebuilder:webhook:validating-webhook-config-name=test-validating-webhook-cfg
```

在 v1 中，一个单个的 webhook 标记可能会被拆分成多个段落。但是在 v2 中，每一个
 webhook 必须由一个单个的标记来表示。

## 其他

v1 中如果对 `main.go` 有任何手动更新，我们需要将修改同步至新的 `main.go` 中。我们还
需要确保所有需要的 schemes 已经被注册了。

如果在 `config` 目录下有一些额外的清单被添加进来，同样需要做同步。


如果需要的话在 Makefile 中修改镜像名字。

## 验证

最后，我们可以运行 `make` 和 `make docker-build` 来确保一些都运行正常。

[v2-project]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project
[builtin-type-example]: https://sigs.k8s.io/controller-runtime/examples/builtins
