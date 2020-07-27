# 创建一个 API

搭建一个新的 Kind （你刚在 [上一章节](./gvks.md#kinds-and-resources) 中注意到的，是吗？) 和相应的控制器，我们可以用 `kubebuilder create api`:

```bash
kubebuilder create api --group batch --version v1 --kind CronJob
```

当第一次我们为每个组-版本调用这个命令的时候，它将会为新的组-版本创建一个目录。

在本案例中，创建了一个对应于`batch.tutorial.kubebuilder.io/v1`（记得我们在开始时 [`--domain`](cronjob-tutorial.md#scaffolding-out-our-project) 的设置吗？) 的 [`api/v1/`](https://sigs.k8s.io/kubebuilder/docs/book/src/cronjob-tutorial/testdata/project/api/v1) 目录。

它也为我们的`CronJob` Kind 添加了一个文件，`api/v1/cronjob_types.go`。每次当我们用不同的 kind 去调用这个命令，它将添加一个相应的新文件。

让我们来看看我们得到了哪些东西，然后我们就可以开始去填写了。

{{#literatego ./testdata/emptyapi.go}}

现在我们已经看到了基本的结构了，让我们开始去填写它吧！
