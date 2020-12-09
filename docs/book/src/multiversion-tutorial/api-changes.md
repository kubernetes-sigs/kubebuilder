# 修改

Kubernetes API 中一个相当常见的改变是获取一些非结构化的或者存储在一些特殊的字符串格式的数据，
并将其修改为结构化的数据。我们的 `schedule` 字段非常适合这个案例 -- 现在，在 `v1` 中，我们的 schedules 是这样的

```yaml
schedule: "*/1 * * * *"
```

这是一个非常典型的特殊字符串格式的例子（除非你是一个 Unix 系统管理员，否则非常难以理解它）。

让我们来使它更结构化一点。根据我们 [CronJob 代码][cronjob-sched-code]，我们支持"standard" Cron 格式。

在 Kubernetes 里，**所有版本都必须通过彼此进行安全的往返**。这意味着如果我们从版本 1 转换到版本 2，然后回退到版本 1，我们一定会失去一些信息。因此，我们对 API 所做的任何更改都必须与 v1 中所支持的内容兼容还需要确保我们添加到 v2 中的任何内容在 v1 中都得到支持。某些情况下，这意味着我们需要向 V1 中添加新的字段，但是在我们这个例子中，我们不会这么做，因为我们没有添加新的功能。

记住这些，让我们将上面的示例转换为稍微更结构化一点：

```yaml
schedule:
  minute: */1
```

现在，至少我们每个字段都有了标签，但是我们仍然可以为每个字段轻松地支持所有不同的语法。

对这个改变我们将需要一个新的 API 版本。我们称它为 v2:

```shell
kubebuilder create api --group batch --version v2 --kind CronJob
```

现在，让我们复制已经存在的类型，并做一些改变：

{{#literatego ./testdata/project/api/v2/cronjob_types.go}}

## 存储版本

{{#literatego ./testdata/project/api/v1/cronjob_types.go}}

现在我们已经准备好了类型，接下来需要设置转换。

[cronjob-sched-code]: /TODO.md
