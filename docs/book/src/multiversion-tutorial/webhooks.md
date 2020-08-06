# 设置 webhook

我们的 conversion 已经就位，所以接下来就是告诉 controller-runtime 关于我们的 conversion。

通常，我们通过运行

```shell
kubebuilder create webhook --group batch --version v1 --kind CronJob --conversion
```

来搭建起 webhook 设置。然而，当我们已经创建好默认和验证过的 webhook 时，我们就已经设置好 webhook。

## Webhook 设置...

{{#literatego ./testdata/project/api/v1/cronjob_webhook.go}}

## ...以及 `main.go`

同样地，我们的 main 文件也已就绪：

{{#literatego ./testdata/project/main.go}}

所有都已经设置准备好！接下来要做的只有测试我们的 webhook。
