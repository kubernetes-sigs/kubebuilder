# 简要说明: 剩下文件的作用？

如果你在[`api/v1/`](https://sigs.k8s.io/kubebuilder/docs/book/src/cronjob-tutorial/testdata/project/api/v1)目录下看到了其他文件，
你可能会注意到除了`cronjob_types.go`这个文件外，还有两个文件：`groupversion_info.go` and `zz_generated.deepcopy.go`。


虽然这些文件都不需要编辑(前者保持原样，而后者是自动生成的)，但是如果知道这些文件的内容呢，那么将是非常有用的。

## `groupversion_info.go`

`groupversion_info.go` 包含了关于group-version的一些元数据:

{{#literatego ./testdata/project/api/v1/groupversion_info.go}}

## `zz_generated.deepcopy.go`

`zz_generated.deepcopy.go`包含了前述`runtime.Object`接口的自动实现，这些实现标记了代表 `Kinds` 的所有根类型。

`runtime.Object` 接口的核心是一个深拷贝方法，即`DeepCopyObject`。

controller-tools 中的 `object` 生成器也能够为每一个根类型以及其子类型生成另外两个易用的方法：`DeepCopy` and
`DeepCopyInto`。
