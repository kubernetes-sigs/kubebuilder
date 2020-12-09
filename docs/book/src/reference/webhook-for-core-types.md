# 核心类型的准入 Webhook

为 CRD 构建准入 webhook 非常容易，这在 CronJob 教程中已经介绍过了。由于 kubebuilder 不支持核心类型的 webhook 自动生成，您必须使用 controller-runtime 的库来处理它。这里可以参考 controller-runtime 的一个 [示例](https://github.com/kubernetes-sigs/controller-runtime/tree/master/examples/builtins)。

建议使用 kubebuilder 初始化一个项目，然后按照下面的步骤为核心类型添加准入 webhook。

## 实现处理程序

你需要用自己的处理程序去实现 [admission.Handler](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/webhook/admission?tab=doc#Handler) 接口。

```go
type podAnnotator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (a *podAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	//在 pod 中修改字段

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
```

如果需要客户端，只需在结构构建时传入客户端。

如果你为你的处理程序添加了 `InjectDecoder` 方法，将会注入一个解码器。

```go
func (a *podAnnotator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
```

**注意**: 为了使得 controller-gen 能够为你生成 webhook 配置，你需要添加一些标记。例如，
`// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io`

## 更新 main.go

现在你需要在 webhook 服务端中注册你的处理程序。

```go
mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{Handler: &podAnnotator{Client: mgr.GetClient()}})
```

您需要确保这里的路径与标记中的路径相匹配。

## 部署

部署它就像为 CRD 部署 webhook 服务端一样。你需要

1) 提供服务证书
2) 部署服务端

你可以参考 [教程](/cronjob-tutorial/running.md)。
