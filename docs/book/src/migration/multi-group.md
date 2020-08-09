# Single Group to Multi-Group

<aside class="note warning">

<h1>Note</h1>

kubebuilder在早期的v2版本(v2.0.0)中不支持 Multi-group 代码生成功能。

想要更新项目结构支持 Multi-Group，请运行命令`kubebuilder edit --multigroup = true`。 
更新到 Multi-group 结构后，将在新结构中生成新的种类，但是需要一些其他手动操作才能将旧的API组移至新结构中。

</aside>

尽管默认情况下KubeBuilder v2不会在同一存储库中搭建与多个API组兼容的项目结构，但可以修改默认项目结构以支持它。

让我们迁移下[CronJob示例] [cronjob-tutorial]。

通常，我们使用API组的前缀作为目录名称。 查看 `api/v1/groupversion_info.go`，我们可以看到：

```go
// +groupName=batch.tutorial.kubebuilder.io
package v1
```

为了让api结构更清晰，我们将 `api` 重名名为 `apis`，并将现有的 API 移动到新的子目录 batch 中：

```bash
mkdir apis/batch
mv api/* apis/batch
# 确保所有文件都成功移动后，删除旧目录 `api`
rm -rf api/ 
```


将API移至新目录后，控制器也需要做相同的处理：

```bash
mkdir controllers/batch
mv controllers/* controllers/batch/
```

接下来，我们将需要更新所有对旧软件包名称的引用。
对于 CronJob，我们需要更新 `main.go` 和 `controllers/batch/cronjob_controller.go`

如果你已经在项目中添加了其他文件，那么也需要更新这些文件中的引用。

最后，我们将运行在项目中启用Multi-group模式的命令：

```
kubebuilder edit --multigroup=true
```

执行 `kubebuilder edit --multigroup=true` 命令后，kubebuilder 将会在 `PROJECT` 中新增一行，标记该项目是一个 multi-group 项目:
                                                      
```yaml
version: "2"
domain: tutorial.kubebuilder.io
repo: tutorial.kubebuilder.io/project
multigroup: true
```

请注意，此选项表示这是一个Multi-group项目。

这样，如果项目不是新项目，并且已经实现了以前的API，则将在以前的结构中。
请注意，在 `multi-group` 项目中，Kind API 的文件被生成在 `apis/<group>/<version>`，而不是在 `api/<version>`。
另外，请注意控制器将在 `controllers/<group>` 目录下创建，而不是在 `controllers`。 
这就是为什么我们在前面的步骤中使用脚本移动之前生成的API。
请记住，之后要更新参考。

[CronJob教程] [cronjob-tutorial]更详细地解释了每个更改（在KubeBuilder为single-group项目生成这些更改的上下文中）。

[multi-group-issue]: https://github.com/kubernetes-sigs/kubebuilder/issues/923 "KubeBuilder Issue #923"
[cronjob-tutorial]: /cronjob-tutorial/cronjob-tutorial.md "Tutorial: Building CronJob"
