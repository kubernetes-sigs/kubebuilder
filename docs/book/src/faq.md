
# FAQ

## How does the value informed via the domain flag (i.e. `kubebuilder init --domain example.com`) when we init a project?

After creating a project, usually you will want to extend the Kubernetes APIs and define new APIs which will be owned by your project. Therefore, the domain value is tracked in the [PROJECT][project-file-def] file which defines the config of your project and will be used as a domain to create the endpoints of your API(s). Please, ensure that you understand the [Groups and Versions and Kinds, oh my!][gvk].

The domain is for the group suffix, to explicitly show the resource group category.
For example, if set `--domain=example.com`:
```
kubebuilder init --domain example.com --repo xxx --plugins=go/v4
kubebuilder create api --group mygroup --version v1beta1 --kind Mykind
```
Then the result resource group will be `mygroup.example.com`.

> If domain field not set, the default value is `my.domain`.

## I'd like to customize my project to use [klog][klog] instead of the [zap][zap] provided by controller-runtime. How to use `klog` or other loggers as the project logger?

In the `main.go` you can replace:
```go
    opts := zap.Options{
    Development: true,
    }
    opts.BindFlags(flag.CommandLine)
    flag.Parse()
    
    ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
```
with:
```go
    flag.Parse()
	ctrl.SetLogger(klog.NewKlogr())
```

## After `make run`, I see errors like "unable to find leader election namespace: not running in-cluster..."

You can enable the leader election. However, if you are testing the project locally using the `make run`
target which will run the manager outside of the cluster then, you might also need to set the
namespace the leader election resource will be created, as follows:
```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      metricsAddr,
		Port:                    9443,
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "14be1926.testproject.org",
		LeaderElectionNamespace: "<project-name>-system",
```

If you are running the project on the cluster with `make deploy` target
then, you might not want to add this option. So, you might want to customize this behaviour using
environment variables to only add this option for development purposes, such as:

```go
    leaderElectionNS := ""
	if os.Getenv("ENABLE_LEADER_ELECATION_NAMESPACE") != "false" {
		leaderElectionNS = "<project-name>-system"
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      metricsAddr,
		Port:                    9443,
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionNamespace: leaderElectionNS,
		LeaderElectionID:        "14be1926.testproject.org",
		...
```

## I am facing the error "open /var/run/secrets/kubernetes.io/serviceaccount/token: permission denied" when I deploy my project against Kubernetes old versions. How to sort it out?

If you are facing the error:
```
1.6656687258729894e+09  ERROR   controller-runtime.client.config        unable to get kubeconfig        {"error": "open /var/run/secrets/kubernetes.io/serviceaccount/token: permission denied"}
sigs.k8s.io/controller-runtime/pkg/client/config.GetConfigOrDie
        /go/pkg/mod/sigs.k8s.io/controller-runtime@v0.13.0/pkg/client/config/config.go:153
main.main
        /workspace/main.go:68
runtime.main
        /usr/local/go/src/runtime/proc.go:250
```
when you are running the project against a Kubernetes old version (maybe <= 1.21) , it might be caused by the [issue][permission-issue] , the reason is the mounted token file set to `0600`, see [solution][permission-PR] here. Then, the workaround is:

Add `fsGroup` in the manager.yaml
```yaml
securityContext:
        runAsNonRoot: true
        fsGroup: 65532 # add this fsGroup to make the token file readable
```
However, note that this problem is fixed and will not occur if you deploy the project in high versions (maybe >= 1.22).

[gvk]: ./cronjob-tutorial/gvks.md
[project-file-def]: ./reference/project-config.md
[klog]: https://github.com/kubernetes/klog
[zap]: https://github.com/uber-go/zap
[permission-issue]: https://github.com/kubernetes/kubernetes/issues/82573
[permission-PR]: https://github.com/kubernetes/kubernetes/pull/89193
