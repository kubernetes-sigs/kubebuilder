# Config/Code 生成标记

Kubebuilder 利用一个叫做[controller-gen](/reference/controller-gen.md)的工具来生成公共的代码和 Kubernetes YAML 文件。
这些代码和配置的生成是由 Go 代码中特殊存在的“标记注释”来控制的。


标记都是以加号开头的单行注释，后面跟着一个标记名称，而跟随的关于标记的特定配置则是可选的。

```go
// +kubebuilder:validation:Optional
// +kubebuilder:validation:MaxItems=2
// +kubebuilder:printcolumn:JSONPath=".status.replicas",name=Replicas,type=string
```

关于不同类型的代码和 YAML 生成可以查看每一小节来获取详细信息。

## 在 KubeBuilder 中生成代码 & 制品

Kubebuilder 项目有两个 `make` 命令用到了 controller-gen：

- `make manifests` 用来生成 Kubernetes 对象的 YAML 文件，像[CustomResourceDefinitions](./markers/crd.md)，[WebhookConfigurations](./markers/webhook.md) 和 [RBAC
  roles](./markers/rbac.md)。

- `make generate` 用来生成代码，像[runtime.Object/DeepCopy
  implementations](./markers/object.md)。


查看[生成 CRDs]来获取综合描述。

## 标记语法

准确的语法在[godocs for
controller-tools](https://pkg.go.dev/sigs.k8s.io/controller-tools/pkg/markers?tab=doc)有描述。

通常，标记可以是：

- **Empty** (`+kubebuilder:validation:Optional`)：空标记像命令行中的布尔标记位-- 仅仅是指定他们来开启某些行为。

- **Anonymous** (`+kubebuilder:validation:MaxItems=2`)：匿名标记使用单个值作为参数。

- **Multi-option**
  (`+kubebuilder:printcolumn:JSONPath=".status.replicas",name=Replicas,type=string`)：多选项标记使用一个或多个命名参数。第一个参数与名称之间用冒号隔开，而后面的参数使用逗号隔开。参数的顺序没有关系。有些参数是可选的。

Marker arguments may be strings, ints, bools, slices, or maps thereof.
Strings, ints, and bools follow their Go syntax:

标记的参数可以是字符，整数，布尔，切片，或者 map 类型。
字符，整数，和布尔都应该符合 Go 语法：

```go
// +kubebuilder:validation:ExclusiveMaximum=false
// +kubebuilder:validation:Format="date-time"
// +kubebuilder:validation:Maximum=42
```

为了方便，在简单的例子中字符的引号可以被忽略，尽管这种做法在任何时候都是不被鼓励使用的，即便是单个字符：

```go
// +kubebuilder:validation:Type=string
```

切片可以用大括号和逗号分隔来指定。

```go
// +kubebuilder:webhooks:Enum={"crackers, Gromit, we forgot the crackers!","not even wensleydale?"}
```

或者，在简单的例子中，用分号来隔开。

```go
// +kubebuilder:validation:Enum=Wallace;Gromit;Chicken
```

Maps 是用字符类型的键和任意类型的值（有效地`map[string]interface{}`）来指定的。一个 map 是由大括号（`{}`）包围起来的，每一个键和每一个值是用冒号（`:`）隔开的，每一个键值对是由逗号隔开的。

```go
// +kubebuilder:validation:Default={magic: {numero: 42, stringified: forty-two}}
```
