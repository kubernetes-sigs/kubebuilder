
# FAQ

<aside class="note">
<h1> Controller-Runtime FAQ </h1>

Kubebuilder is developed on top of the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
and [controller-tools](https://github.com/kubernetes-sigs/controller-tools) libraries. We recommend you also check
the [Controller-Runtime FAQ page](https://github.com/kubernetes-sigs/controller-runtime/blob/main/FAQ.md).
</aside>


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
	if os.Getenv("ENABLE_LEADER_ELECTION_NAMESPACE") != "false" {
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

## The error `Too long: must have at most 262144 bytes` is faced when I run `make install` to apply the CRD manifests. How to solve it? Why this error is faced?

When attempting to run `make install` to apply the CRD manifests, the error `Too long: must have at most 262144 bytes may be encountered.` This error arises due to a size limit enforced by the Kubernetes API. Note that the `make install` target will apply the CRD manifest under `config/crd` using `kubectl apply -f -`. Therefore, when the apply command is used, the API annotates the object with the `last-applied-configuration` which contains the entire previous configuration. If this configuration is too large, it will exceed the allowed byte size. ([More info][k8s-obj-creation])

In ideal approach might use client-side apply might seem like the perfect solution since with the entire object configuration doesn't have to be stored as an annotation (last-applied-configuration) on the server. However, it's worth noting that as of now, it isn't supported by controller-gen or kubebuilder. For more on this, refer to: [Controller-tool-discussion][controller-tool-pr].

Therefore, you have a few options to workround this scenario such as:

**By removing the descriptions from CRDs:**

Your CRDs are generated using [controller-gen][controller-gen]. By using the option `maxDescLen=0` to remove the description, you may reduce the size, potentially resolving the issue. To do it you can update the Makefile as the following example and then, call the target `make manifest` to regenerate your CRDs without description, see:

```shell

 .PHONY: manifests
 manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
     # Note that the option maxDescLen=0 was added in the default scaffold in order to sort out the issue
     # Too long: must have at most 262144 bytes. By using kubectl apply to create / update resources an annotation
     # is created by K8s API to store the latest version of the resource ( kubectl.kubernetes.io/last-applied-configuration).
     # However, it has a size limit and if the CRD is too big with so many long descriptions as this one it will cause the failure.
 	$(CONTROLLER_GEN) rbac:roleName=manager-role crd:maxDescLen=0 webhook paths="./..." output:crd:artifacts:config=config/crd/bases
```
**By re-design your APIs:**

You can review the design of your APIs and see if it has not more specs than should be by hurting single responsibility principle for example. So that you might to re-design them.

## How can I validate and parse fields in CRDs effectively?

To enhance user experience, it is recommended to use [OpenAPI v3 schema](https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.0.md#schemaObject) validation when writing your CRDs. However, this approach can sometimes require an additional parsing step.
For example, consider this code
```go
type StructName struct {
	// +kubebuilder:validation:Format=date-time
	TimeField string `json:"timeField,omitempty"`
}
```

### What happens in this scenario?

- Users will receive an error notification from the Kubernetes API if they attempt to create a CRD with an invalid timeField value.
- On the developer side, the string value needs to be parsed manually before use.

### Is there a better approach?

To provide both a better user experience and a streamlined developer experience, it is advisable to use predefined types like [`metav1.Time`](https://pkg.go.dev/k8s.io/apimachinery@v0.31.1/pkg/apis/meta/v1#Time)
For example, consider this code
```go
type StructName struct {
	TimeField metav1.Time `json:"timeField,omitempty"`
}
```

### What happens in this scenario?

- Users still receive error notifications from the Kubernetes API for invalid `timeField` values.
- Developers can directly use the parsed TimeField in their code without additional parsing, reducing errors and improving efficiency.



[k8s-obj-creation]: https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/#how-to-create-objects
[gvk]: ./cronjob-tutorial/gvks.md
[project-file-def]: ./reference/project-config.md
[klog]: https://github.com/kubernetes/klog
[zap]: https://github.com/uber-go/zap
[permission-issue]: https://github.com/kubernetes/kubernetes/issues/82573
[permission-PR]: https://github.com/kubernetes/kubernetes/pull/89193
[controller-gen]: ./reference/controller-gen.html
[controller-tool-pr]: https://github.com/kubernetes-sigs/controller-tools/pull/536