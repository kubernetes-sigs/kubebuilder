# Hubs, spokes, 和其他的 wheel metaphors

由于我们现在有两个不同的版本，用户可以请求任意一个版本，我们必须定义一种在版本之间进行转换的方法。对于CRD，这是通过使用 Webhook 完成的，类似我们[在基础中定义webhooks教程]（/ cronjob-tutorial / webhook-implementation.md）的默认设置和验证一样。像以前一样，控制器运行时将帮助我们将所有细节都连接在一起，而我们只需实现本身的转换即可。

在执行此操作之前，我们需要了解控制器运行时如何处理版本的。即：

## 任意两个版本间转换的不足之处

定义转换的一种简单方法可能是定义转换函数如何可以在我们的每个版本之间进行转换。然后，只要我们需要进行转换的时候，我们只需要查找适当的函数，然后调用它就可以执行转换。当我们只有两个版本时，这可以正常工作，但是如果我们有4个版本的时候，或者更多的时候该怎么办？那将会有很多转换功能。

相反，控制器运行时会根据 “hub 和 spoke” 模型-我们将一个版本标记为“hub”，而所有其他版本只需定义为与 hub 之间的来源即可：

<!-- include these inline so we can style an match variables -->
<div class="diagrams">
{{#include ./complete-graph-8.svg}}
<div>becomes</div>
{{#include ./hub-spoke-graph.svg}}
</div>

如果我们必须在两个 non-hub 之间进行转换，则我们首先要进行转换到这个 hub 对应的版本，然后再转换到我们所需的版本：

<div class="diagrams">
{{#include ./conversion-diagram.svg}}
</div>

这样就减少了我们所需定义转换函数的数量，其实就是在模仿 Kubernetes 内部实际的工作方式。

## 与Webhooks有什么关系？

当API客户端（例如kubectl或您的控制器）请求特定的版本的资源，Kubernetes API服务器需要返回该版本的结果。但是，该版本可能不匹配API服务器实际存储的版本。

在这种情况下，API服务器需要知道如何在所需的版本和存储的版本之间进行转换。由于转换不是CRD内置的，于是Kubernetes API服务器通过调用Webhook来执行转换。对于KubeBuilder，跟我们上面讨论一样，Webhook通过控制器运行时来执行hub-and-spoke的转换。

现在我们有了向下转换的模型，我们就可以实现转换操作了。
