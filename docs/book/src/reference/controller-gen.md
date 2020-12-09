# controller-gen CLI

KubeBuilder 使用了一个称为 [controller-gen](https://sigs.k8s.io/controller-tools/cmd/controller-gen)
用于生成通用代码和 Kubernetes YAML。 代码和配置的生成规则是被 Go 代码中的一些特殊[标记注释](/reference/markers.md)控制的。

controller-gen 由不同的“generators”(指定生成什么)和“输出规则”(指定如何以及在何处输出结果)。

两者都是通过指定的命令行参数配置的，更详细的说明见 [标记格式化](/reference/markers.md)。


例如，


```shell
controller-gen paths=./... crd:trivialVersions=true rbac:roleName=controller-perms output:crd:artifacts:config=config/crd/bases
```

生成的 CRD 和 RBAC YAML 文件默认存储在`config/crd/bases`目录。 
RBAC 规则默认输出到(`config/rbac`)。 主要考虑到当前目录结构中的每个包的关系。
(按照 go `...` 的通配符规则)。

## 生成器

每个不同的生成器都是通过 CLI 选项配置的。controller-gen 一次运行也可以指定多个生成器。

{{#markerdocs CLI: generators}}

## 输出规则

输出规则配置给定生成器如何输出其结果。 默认是一个全局 fallback 输出规则(指定为 `output:<rule>`)，
另外还有 per-generator 的规则(指定为`output:<generator>:<rule>`)，会覆盖掉 fallback 规则。

<aside class="note">

<h1>默认规则</h1>


如果没有手动指定 fallback 规则，默认的 per-generator 将被使用，生成的 YAML 将放到
`config/<generator>`相应目录，代码所在的位置不变。


对于每个生成器来说，默认的规则等价于`output:<generator>:artifacts:config=config/<generator>`。

指定 fallback 规则后，将使用该规则代替默认规则。

例如，如果你指定`crd rbac:roleName=controller-permsoutput:crd:stdout`，你将在标准输出中获得 CRD，在`config/rbac`目录得到 rbac 规则。 
如果你要添加全局规则，例如`crdrbac:roleName=controller-perms output:crd:stdout output:none`，CRD 会被重定向到终端输出，其他被重定向到 /dev/null，因为我们已经明确指定了 fallback 。

</aside>

为简便起见，每个生成器的输出规则(`output:<generator>:<rule>`)默认省略。 相当于这里列出的全局备用选项。

{{#markerdocs CLI: output rules (optionally as output:<generator>:...)}}

## 其他选项

{{#markerdocs CLI: generic}}
