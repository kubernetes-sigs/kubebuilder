# 实现默认/验证 webhook

如果你想为你的 CRD 实现一个 [admission webhooks](../reference/admission-webhook.md)，
你需要做的一件事就是去实现`Defaulter` 和/或 `Validator` 接口。

Kubebuilder 会帮你处理剩下的事情，像下面这些：

1. 创建 webhook 服务端。
2. 确保服务端已添加到 manager 中。
3. 为你的 webhooks 创建处理函数。
4. 用路径在你的服务端中注册每个处理函数。

首先，让我们为我们的 CRD (CronJob) 创建一个 webhooks 的支架。我们将需要运行下面的命令并带上 `--defaulting` 和 `--programmatic-validation` 标志（因为我们的测试项目会用到默认和验证 webhooks)：

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation
```

这里会在你的 `main.go` 中搭建一个 webhook 函数的支架并用 manager 注册你的 webhook。

{{#literatego ./testdata/project/api/v1/cronjob_webhook.go}}
