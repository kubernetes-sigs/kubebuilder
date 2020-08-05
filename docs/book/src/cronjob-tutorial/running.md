# 运行和部署 controller

要测试 controller，我们可以在集群本地运行它。不过，在开始之前，我们需要按照 [快速入门](/quick-start.md) 安装 CRD。如果需要，将使用 controller-tools 自动更新 YAML 清单：

```bash
make install
```

现在我们已经安装了 CRD，在集群上运行 controller 了。这将使用与集群连接所用的任何凭据，因此我们现在不必担心 RBAC。

<aside class="note"> 

<h1>本地运行 webhooks</h1>

如果您想在本地运行 webhooks，必须为 webhooks 服务生成证书，并将它们放在正确的目录中（默认 `/tmp/k8s-webhook-server/serving-certs/tls.{crt,key}`）。

如果您未运行本地 API server，则还需要弄清楚如何将流量从远程群集代理到本地 Webhook 服务器。因此，我们通常建议在执行本地 code-run-test 时禁用 webhooks，如下所示。

</aside>

在单独的终端中运行

```bash
make run ENABLE_WEBHOOKS=false
```

您应该会看到 controller 关于启动的日志，但它还没有做任何事情。

此时，我们需要一个 CronJob 进行测试。让我们写一个样例到 `config/samples/batch_v1_cronjob.yaml`，并使用:

```yaml
{{#include ./testdata/project/config/samples/batch_v1_cronjob.yaml}}
```

```bash
kubectl create -f config/samples/batch_v1_cronjob.yaml
```

此时，您应该看到一系列的活动。如果看到变更，则应该看到您的 cronjob 正在运行，并且正在更新状态：

```bash
kubectl get cronjob.batch.tutorial.kubebuilder.io -o yaml
kubectl get job
```

现在我们知道它正在工作，我们可以在集群中运行它。停止 `make run` 调用，然后运行

```bash
make docker-build docker-push IMG=<some-registry>/<project-name>:tag
make deploy IMG=<some-registry>/<project-name>:tag
```

如果像以前一样再次列出 cronjobs，我们应该会看到控制器再次运行！
