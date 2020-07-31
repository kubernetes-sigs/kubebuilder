# 生成 CRD

KubeBuilder 使用一个叫做 [`controller-gen`][controller-tools] 的工具来生成工具代码和 Kubernetes 的 YAML 对象，比如 CustomResourceDefinitions。

为了实现这种方式，它使用一种特殊的 “评论标记”（以 `// +` 开头）来表示这里要插入字段，类型和包相关的信息。如果是 CRD，那么这些信息通常是从以 `_types.go` 结尾的文件中产生的。更多关于标记的信息，可以看[标记相关文档][marker-ref]。

KubeBuilder 提供了提供了一个 `make` 的命令来运行 controller-gen 并生成 CRD：`make manifests`。
 
当运行 `make manifests` 的时候，在 `config/crd/bases` 目录下可以看到生成的 CRD。`make manifests` 可以生成许多其它的文件 -- 更多详情请查看[标记相关文档][marker-ref]。

## 验证

CRD 支持使用 [OpenAPI v3 schema][openapi-schema] 在 `validation` 段中进行[声明式验证 validation][kube-validation]。

通常，[验证标记](./markers/crd-validation.md)可能会关联到字段或者类型。如果你定义了复杂的验证，或者如果你需要重复使用验证，亦或者你需要验证切片元素，那么通常你最好定义一个新的类型来描述你的验证。

例如：

```go
type ToySpec struct {
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:MaxItems=500
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=true
	Knights []string `json:"knights,omitempty"`

	Alias   Alias   `json:"alias,omitempty"`
	Rank    Rank    `json:"rank"`
}

// +kubebuilder:validation:Enum=Lion;Wolf;Dragon
type Alias string

// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=3
// +kubebuilder:validation:ExclusiveMaximum=false
type Rank int32

```

## 打印其其它信息列

从 Kubernetes 1.11 开始，`kubectl get` 可以询问 Kubernetes 服务要展示哪些列。对于 CRD 来说，可以用 `kubectl get` 来提供展示有用的特定类型的信息，类似于为内置类型提供的信息。

你 CRD 的 [additionalPrinterColumns 字段][kube-additional-printer-columns]控制了要展示的信息，它是通过在给 CRD 的 Go 类型上标注 [`+kubebuilder:printcolumn`][crd-markers] 标签来控制要展示的信息。

比如下面的验证例子，我们为 knights, rank, 和 alias 字段添加了要展示的信息字段。

```go
// +kubebuilder:printcolumn:name="Alias",type=string,JSONPath=`.spec.alias`
// +kubebuilder:printcolumn:name="Rank",type=integer,JSONPath=`.spec.rank`
// +kubebuilder:printcolumn:name="Bravely Run Away",type=boolean,JSONPath=`.spec.knights[?(@ == "Sir Robin")]`,description="when danger rears its ugly head, he bravely turned his tail and fled",priority=10
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Toy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToySpec   `json:"spec,omitempty"`
	Status ToyStatus `json:"status,omitempty"`
}

```

## 子资源

在 Kubernetes 1.13 中 CRD 可以选择实现 `/status` 和 `/scale` 这类[子资源][kube-subresources]。

通常推荐你在所有资源上实现 `/status` 子资源的时候，要有一个状态字段。

两个子资源都有对应的[标签][crd markers]。

### 状态

通过 `+kubebuilder:subresource:status` 设置子资源的状态。当时启用状态时，更细主资源不会修改它的状态。类似的，更新子资源状态也只是修改了状态字段。

例如：

```go
// +kubebuilder:subresource:status
type Toy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToySpec   `json:"spec,omitempty"`
	Status ToyStatus `json:"status,omitempty"`
}
```

### 扩展

通过设置 `+kubebuilder:subresource:scale` 来启用扩展子资源。当启用的时候用户可以对你的资源使用 `kubectl scale` 命令。如果 `selectorpath` 参数被指定为字符串形式的标签选择器，HorizontalPodAutoscaler 将可以自动扩容你的资源。

例如：

```go
type CustomSetSpec struct {
	Replicas *int32 `json:"replicas"`
}

type CustomSetStatus struct {
	Replicas int32 `json:"replicas"`
    Selector string `json:"selector"` // this must be the string form of the selector
}


// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
type CustomSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToySpec   `json:"spec,omitempty"`
	Status ToyStatus `json:"status,omitempty"`
}
```

## 多版本

Kubernetes 1.13，你可以有在你的 CRD 的同一个 Kind 中定义多个版本，并且使用一个 webhook 对它们进行互转。

更多这部分相关的信息，请看[多版本教程](/multiversion-tutorial/tutorial.md)。

默认情况下，KubeBuilder 会禁止为你的 CRD 的不同版本产生不同的验证，这都是为了兼容老版本的 Kubernetes。

如果需要，你要通过修改 makefile 中的命令：把 `CRD_OPTIONS ?= "crd:trivialVersions=true` 修改为 `CRD_OPTIONS ?= crd`。

这样，你就可以使用  `+kubebuilder:storageversion` [标签][crd-markers] 来告知 [GVK](/cronjob-tutorial/gvks.md "Group-Version-Kind") 这个字段应该被 API 服务来存储数据。

## 背后执行

KubeBuilder 会制定规则来运行 `controller-gen`。如果 `controller-gen` 不在 `go get` 用来下载 Go 模块的路径下的时候，这些规则会自动的安装 `controller-gen`。

如果你想它到底做了什么，你也可以直接运行 `controller-gen`。

每一个 `controller-gen` “生成器” 都由 `controller-gen` 的一个参数选项控制，和标签的语法一样。比如，要生成带有 "trivial versions" 的 CRD（无版本转换的 webhook），我们可以执行 `controller-gen crd:trivialVersions=true paths=./api/...`。

`controller-gen` 也支持不同的输出“规则”，以此来控制如何及输出到哪里。注意 `manifests` 生成规则（是只生成 CRD 的简短写法）：

```makefile
# Generate manifests for CRDs
manifests: controller-gen
	$(CONTROLLER_GEN) crd:trivialVersions=true paths="./..." output:crd:artifacts:config=config/crd/bases
```

它使用了 `output:crd:artifacts`  输出规则来表示 CRD 关联的配置（非代码）应该用 `config/crd/bases`，而不是用 `config/crd`。

运行如下命令可以看到 `controller-gen` 的所有支持参数：

```shell
$ controller-gen -h
```

或者，这样查看更多详情：

```shell
$ controller-gen -hhh
```

[marker-ref]: ./markers.md "Markers for Config/Code Generation"

[kube-validation]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#validation "Custom Resource Definitions: Validation"

[openapi-schema]: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#schemaObject "OpenAPI v3"

[kube-additional-printer-colums]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#additional-printer-columns "Custom Resource Definitions: Additional Printer Columns"

[kube-subresources]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#status-subresource "Custom Resource Definitions: Status Subresource"

[crd-markers]: ./markers/crd.md "CRD Generation"

[controller-tools]: https://sigs.k8s.io/controller-tools "Controller Tools"
