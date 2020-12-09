# 实现转换

采用的转换模型已经就绪，就可以开始实现转换函数了。 我们将这些函数放置在 `cronjob_conversion.go` 文件中，`cronjob_conversion.go` 文件和 `cronjob_types.go` 文件同目录，以避免我们主要的类型文件和额外的方法产生混乱。

## Hub...

首先，我们需要实现 hub 接口。我们会选择 v1 版本作为 hub 的一个实现：

{{#literatego ./testdata/project/api/v1/cronjob_conversion.go}}

## ... 然后 Spokes

然后，我们需要实现我们的 spoke 接口，例如 v2 版本：

{{#literatego ./testdata/project/api/v2/cronjob_conversion.go}}

现在我们的转换方法已经就绪，我们要做的就是启动我们的 main 方法来运行 webhook。
