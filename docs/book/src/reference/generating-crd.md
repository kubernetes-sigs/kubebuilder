# 生成 CRDs

KubeBuilder 用工具 [`controller-gen`][controller-tools] 来生成功能代码和描述 Kubernetes 
对象的 YAML 文件，比如 CustomResourceDefinitions。

为此，它使用了特殊的“标记注释”(以 `// +` 为开头的注释)来表明与字段，类型和包有关的额外信息。
对于 CRD 来讲，他们通常是从你的 `_types.go` 生成的。关于标记的更多详细信息，可以查看 [marker
reference docs][marker-ref]。

KubeBuilder 提供了一个能够运行 controller-gen 并生成 CRDs 的 `make` 命令: `make manifests`。

当你运行 `make manifests` 命令时，你应该能在 `config/crd/bases` 目录下看到生成的CRDs。
`make manifests` 命令还能够生成其他的制品--更多详情可以看 [marker reference docs][marker-ref] 。

## 验证

在验证章节，CRDs 支持用 [OpenAPI
v3 schema][openapi-schema] 来支持验证 [declarative validation][kube-validation]。

通常， [validation markers](./markers/crd-validation.md)  或许是和字段或者类型相关联的。
如果你定义了复杂的验证，或者如果你需要重新验证，抑或你需要验证分片元素，最好的办法是定义一个
新的类型来描述你的验证。

比如:

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

## 额外的列信息打印

从 Kubernetes 1.11 版本开始，`kubectl get` 可以告诉服务端如何进行行信息的显示。
对于 CRDs 来讲， `kubectl get` 可以被用来提供一些有用的，特定类型的信息，类似于
内置类型的信息展示。


对于你的CRD，显示的信息可以用 [additionalPrinterColumns field][kube-additional-printer-columns] 
来进行控制，这些 CRD 是被 [`+kubebuilder:printcolumn`][crd-markers] 之类的 Go 类型标记来控制的。

例如，在随后的例子中，我们添加字段来显示 knights，rank 和 alias 字段的信息：

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

截止 Kubernetes 1.13 版本，可以选择实现 CRDs 的 `/status` 和 `/scale`

在所有具有 status 字段的资源中，我们推荐你使用 `/status` 子资源。

所有的子资源都有一个相对应的 [marker][crd-markers]。

### 状态

status 子资源可以通过 `+kubebuilder:subresource:status` 来进行启用。启用后，
主资源的更新不会使子资源的状态发生变化。类似的，子资源的更新除了带来 status 
字段的更新外，不会引起其他任何变化。

比如:

```go
// +kubebuilder:subresource:status
type Toy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToySpec   `json:"spec,omitempty"`
	Status ToyStatus `json:"status,omitempty"`
}
```

### 伸缩

子资源的伸缩可以通过 `+kubebuilder:subresource:scale` 来启用。启用后，用户
可以使用 `kubectl scale` 来对你的资源进行扩容或者缩容。如果 `selectorpath` 参数
是指向一个  label selector 的字符格式， 那么 HorizontalPodAutoscaler 就会对你的资源
进行自动扩容或者缩容。

比如:

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

从 Kubernetes 1.13 开始，在你的 CRD 中定义的 Kind 可以有多个版本，可以用一个 
webhook 来完成彼此的切换。

关于这一点，更多的信息可以查看 [multiversion
tutorial](/multiversion-tutorial/tutorial.md)。


默认的， KubeBuilder 禁用了为用户自定义 CRD 中的 Kind  的不同版本生成不同的验证，
这样做是为了能兼容较老版本的 Kubernetes 。


你可以通过把你 makefile 中的 `CRD_OPTIONS ?= "crd:trivialVersions=true`
改成 `CRD_OPTIONS ?= crd` 来启用这个功能。


然后，你就可以用 `+kubebuilder:storageversion` [marker][crd-markers] 来表明
 [GVK](/cronjob-tutorial/gvks.md "Group-Version-Kind") 应该被用来存储 API server
 的数据。

## 写在最后

KubeBuilder scaffolds out 使用一些规则来运行 `controller-gen`。如果在使用 Go 
模块的 `go get` 所输出的路径中，没有找到 controller-gen ，那么这些规则将自动
安装 controller-gen 。

如果你想知道到底发生了什么，你还可以直接运行 `controller-gen` 。

每一个 controller-gen “生成器” 都有一个 controller-gen 选项来控制，使用语法
和 markers 是一样的。例如，为了用 "trivial versions" (没有版本转换 webhooks) 选项
来生成 CRDs, 我们使用 `controller-gen crd:trivialVersions=true paths=./api/...` 。

controller-gen 也支持不同的输出“规则”来控制输出如何运作以及在哪儿运作。
注意 `manifests` 的 make 规则(仅仅用来生成 CRDs ):

```makefile
# 生成 CRD 清单
manifests: controller-gen
	$(CONTROLLER_GEN) crd:trivialVersions=true paths="./..." output:crd:artifacts:config=config/crd/bases
```

它使用 `output:crd:artifacts` 输出规则来表明与 CRD 相关的配置(非代码级别) 应该在 `config/crd/bases`
目录下，而不是 `config/crd` 目录下。

如果要查看 `controller-gen` 的其他使用选项，可以运行

```shell
$ controller-gen -h
```

或者，可以执行以下命令，获取更多详细信息:

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
