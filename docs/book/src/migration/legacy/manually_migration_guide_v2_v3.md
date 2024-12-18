# Migration from v2 to v3 by updating the files manually

Make sure you understand the [differences between Kubebuilder v2 and v3][migration-v2vsv3]
before continuing

Please ensure you have followed the [installation guide](/quick-start.md#installation)
to install the required components.

The following guide describes the manual steps required to upgrade your config version and start using the plugin-enabled version.

This way is more complex, susceptible to errors, and success cannot be assured. Also, by following these steps you will not get the improvements and bug fixes in the default generated project files.

Usually you will only try to do it manually if you customized your project and deviated too much from the proposed scaffold. Before continuing, ensure that you understand the note about [project customizations][project-customizations]. Note that you might need to spend more effort to do this process manually than organize your project customizations to follow up the proposed layout and keep your project maintainable and upgradable with less effort in the future.

The recommended upgrade approach is to follow the [Migration Guide v2 to V3][migration-guide-v2-to-v3] instead.

## Migration from project config version "2" to "3"

Migrating between project configuration versions involves additions, removals, and/or changes
to fields in your project's `PROJECT` file, which is created by running the `init` command.

The `PROJECT` file now has a new layout. It stores more information about what resources are in use, to better enable plugins to make useful decisions when scaffolding.

Furthermore, the `PROJECT` file itself is now versioned. The `version` field corresponds to the version of the `PROJECT` file itself, while the `layout` field indicates the scaffolding and the primary plugin version in use.

### Steps to migrate

The following steps describe the manual changes required to bring the project configuration file (`PROJECT`). These change will add the information that Kubebuilder would add when generating the file. This file can be found in the root directory.

#### Add the `projectName`

The project name is the name of the project directory in lowercase:

```yaml
...
projectName: example
...
```

#### Add the `layout`

The default plugin layout which is equivalent to the previous version is `go.kubebuilder.io/v2`:

```yaml
...
layout:
- go.kubebuilder.io/v2
...
```

#### Update the `version`

The `version` field represents the version of project's layout. Update this to `"3"`:

```yaml
...
version: "3"
...
```

#### Add the resource data

The attribute `resources` represents the list of resources scaffolded in your project.

You will need to add the following data for each resource added to the project.

##### Add the Kubernetes API version by adding `resources[entry].api.crdVersion: v1beta1`:

```yaml
...
resources:
- api:
    ...
    crdVersion: v1beta1
  domain: my.domain
  group: webapp
  kind: Guestbook
  ...
```

##### Add the scope used do scaffold the CRDs by adding `resources[entry].api.namespaced: true` unless they were cluster-scoped:

```yaml
...
resources:
- api:
    ...
    namespaced: true
  group: webapp
  kind: Guestbook
  ...
```

##### If you have a controller scaffolded for the API then, add `resources[entry].controller: true`:

```yaml
...
resources:
- api:
    ...
  controller: true
  group: webapp
  kind: Guestbook
```

##### Add the resource domain such as `resources[entry].domain: testproject.org` which usually will be the project domain unless the API scaffold is a core type and/or an external type:

```yaml
...
resources:
- api:
    ...
  domain: testproject.org
  group: webapp
  kind: Guestbook
```

<aside class="note">
<h1>Supportability</h1>

Kubebuilder only supports core types and the APIs scaffolded in the project by default unless you manually change the files you will be unable to work with external-types.

  For core types, the domain value will be `k8s.io` or empty.

  However, for an external-type you might leave this attribute empty. We cannot suggest what would be the best approach in this case until it become officially supported by the tool. For further information check the issue [#1999][issue-1999].

</aside>

Note that you will only need to add the `domain` if your project has a scaffold for a core type API which the `Domain` value is not empty in Kubernetes API group qualified scheme definition. (For example, see [here](https://github.com/kubernetes/api/blob/v0.19.7/apps/v1/register.go#L26) that for Kinds from the API `apps` it has not a domain when see [here](https://github.com/kubernetes/api/blob/v0.19.7/authentication/v1/register.go#L26) that for Kinds from the API `authentication` its domain is `k8s.io` )

 Check the following the list to know the core types supported and its domain:

| Core Type | Domain |
|----------|:-------------:|
| admission | "k8s.io" |
| admissionregistration | "k8s.io" |
| apps | empty |
| auditregistration | "k8s.io" |
| apiextensions | "k8s.io" |
| authentication | "k8s.io" |
| authorization | "k8s.io" |
| autoscaling | empty |
| batch | empty |
| certificates | "k8s.io" |
| coordination | "k8s.io" |
| core | empty |
| events | "k8s.io" |
| extensions | empty |
| imagepolicy | "k8s.io" |
| networking | "k8s.io" |
| node | "k8s.io" |
| metrics | "k8s.io" |
| policy | empty |
| rbac.authorization | "k8s.io" |
| scheduling | "k8s.io" |
| setting | "k8s.io" |
| storage | "k8s.io" |

Following an example where a controller was scaffold for the core type Kind Deployment via the command `create api --group apps --version v1 --kind Deployment --controller=true --resource=false --make=false`:

```yaml
- controller: true
  group: apps
  kind: Deployment
  path: k8s.io/api/apps/v1
  version: v1
```

##### Add the `resources[entry].path` with the import path for the api:

<aside class="note">
<h1>Path</h1>

If you did not scaffold an API but only generate a controller for the API(GKV) informed then, you do not need to add the path. Note, that it usually happens when you add a controller for an external or core type.

Kubebuilder only supports core types and the APIs scaffolded in the project by default unless you manually change the files you will be unable to work with external-types.

The path will always be the import path used in your Go files to use the API.

</aside>

```yaml
...
resources:
- api:
    ...
  ...
  group: webapp
  kind: Guestbook
  path: example/api/v1
```

##### If your project is using webhooks then, add `resources[entry].webhooks.[type]: true` for each type generated and then, add `resources[entry].webhooks.webhookVersion: v1beta1`:

<aside class="note">
<h1>Webhooks</h1>

The valid types are: `defaulting`, `validation` and `conversion`. Use the webhook type used to scaffold the project.

The Kubernetes API version used to do the webhooks scaffolds in `Kubebuilder v2` is `v1beta1`. Then, you will add the `webhookVersion: v1beta1` for all cases.

</aside>

```yaml
resources:
- api:
    ...
  ...
  group: webapp
  kind: Guestbook
  webhooks:
    defaulting: true
    validation: true
    webhookVersion: v1beta1
```

#### Check your PROJECT file

Now ensure that your `PROJECT` file has the same information when the manifests are generated via Kubebuilder V3 CLI.

For the QuickStart example, the `PROJECT` file manually updated to use `go.kubebuilder.io/v2` would look like:

```yaml
domain: my.domain
layout:
- go.kubebuilder.io/v2
projectName: example
repo: example
resources:
- api:
    crdVersion: v1
    namespaced: true
  controller: true
  domain: my.domain
  group: webapp
  kind: Guestbook
  path: example/api/v1
  version: v1
version: "3"
```

You can check the differences between the previous layout(`version 2`) and the current format(`version 3`) with the `go.kubebuilder.io/v2` by comparing an example scenario which involves more than one API and webhook, see:

**Example (Project version 2)**

```yaml
domain: testproject.org
repo: sigs.k8s.io/kubebuilder/example
resources:
- group: crew
  kind: Captain
  version: v1
- group: crew
  kind: FirstMate
  version: v1
- group: crew
  kind: Admiral
  version: v1
version: "2"
```

**Example (Project version 3)**

```yaml
domain: testproject.org
layout:
- go.kubebuilder.io/v2
projectName: example
repo: sigs.k8s.io/kubebuilder/example
resources:
- api:
    crdVersion: v1
    namespaced: true
  controller: true
  domain: testproject.org
  group: crew
  kind: Captain
  path: example/api/v1
  version: v1
  webhooks:
    defaulting: true
    validation: true
    webhookVersion: v1
- api:
    crdVersion: v1
    namespaced: true
  controller: true
  domain: testproject.org
  group: crew
  kind: FirstMate
  path: example/api/v1
  version: v1
  webhooks:
    conversion: true
    webhookVersion: v1
- api:
    crdVersion: v1
  controller: true
  domain: testproject.org
  group: crew
  kind: Admiral
  path: example/api/v1
  plural: admirales
  version: v1
  webhooks:
    defaulting: true
    webhookVersion: v1
version: "3"
```

### Verification

In the steps above, you updated only the `PROJECT` file which represents the project configuration. This configuration is useful only for the CLI tool. It should not affect how your project behaves.

There is no option to verify that you properly updated the configuration file. The best way to ensure the configuration file has the correct `V3+` fields is to initialize a project with the same API(s), controller(s), and webhook(s) in order to compare generated configuration with the manually changed configuration.

If you made mistakes in the above process, you will likely face issues using the CLI.


## Update your project to use go/v3 plugin

Migrating between project [plugins][plugins-doc] involves additions, removals, and/or changes
to files created by any plugin-supported command, e.g. `init` and `create`. A plugin supports
one or more project config versions; make sure you upgrade your project's
config version to the latest supported by your target plugin version before upgrading plugin versions.

The following steps describe the manual changes required to modify the project's layout enabling your project to use the `go/v3` plugin. These steps will not help you address all the bug fixes of the already generated scaffolds.

<aside class="note warning">
<h1> Deprecated APIs </h1>

The following steps will not migrate the API versions which are deprecated `apiextensions.k8s.io/v1beta1`, `admissionregistration.k8s.io/v1beta1`, `cert-manager.io/v1alpha2`.

</aside>

### Steps to migrate

#### Update your plugin version into the PROJECT file

Before updating the `layout`, please ensure you have followed the above steps to upgrade your Project version to `3`. Once you have upgraded the project version, update the `layout` to the new plugin version ` go.kubebuilder.io/v3` as follows:

```yaml
domain: my.domain
layout:
- go.kubebuilder.io/v3
...
```

#### Upgrade the Go version and its dependencies:

Ensure that your `go.mod` is using Go version `1.15` and the following dependency versions:

```go
module example

go 1.18

require (
    github.com/onsi/ginkgo/v2 v2.1.4
    github.com/onsi/gomega v1.19.0
    k8s.io/api v0.24.0
    k8s.io/apimachinery v0.24.0
    k8s.io/client-go v0.24.0
    sigs.k8s.io/controller-runtime v0.12.1
)

```

#### Update the golang image

In the Dockerfile, replace:

```
# Build the manager binary
FROM golang:1.13 as builder
```

With:
```
# Build the manager binary
FROM golang:1.16 as builder
```

####  Update your Makefile

##### To allow controller-gen to scaffold the nw Kubernetes APIs

To allow `controller-gen` and the scaffolding tool to use the new API versions, replace:

```
CRD_OPTIONS ?= "crd:trivialVersions=true"
```

With:

```
CRD_OPTIONS ?= "crd"
```

##### To allow automatic downloads

To allow downloading the newer versions of the Kubernetes binaries required by Envtest into the `testbin/` directory of your project instead of the global setup, replace:

```makefile
# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out
```

With:

```makefile
# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out
```

<aside class="note">
<h1>Envtest binaries</h1>

The Kubernetes binaries that are required for the Envtest were upgraded from `1.16.4` to `1.22.1`.
You can still install them globally by following [these installation instructions][doc-envtest].

</aside>

##### To upgrade `controller-gen` and `kustomize` dependencies versions used

To upgrade the `controller-gen` and `kustomize` version used to generate the manifests replace:

```
# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
```

With:

```
##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.9.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
```

And then, to make your project use the `kustomize` version defined in the Makefile, replace all usage of `kustomize` with `$(KUSTOMIZE)`

<aside class="note">
<h1>Makefile</h1>

You can check all changes applied to the Makefile by looking in the samples projects generated in the `testdata` directory of the Kubebuilder repository or by just by creating a new project with the Kubebuilder CLI.

</aside>

#### Update your controllers

<aside class="note warning">
<h1>Controller-runtime version updated has breaking changes</h1>

Check [sigs.k8s.io/controller-runtime release docs from 0.7.0+ version][controller-releases] for breaking changes.

</aside>

Replace:

```go
func (r *<MyKind>Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
    ctx := context.Background()
    log := r.Log.WithValues("cronjob", req.NamespacedName)
```

With:

```go
func (r *<MyKind>Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := r.Log.WithValues("cronjob", req.NamespacedName)
```

#### Update your controller and webhook test suite

<aside class="note warning">
<h1>Ginkgo V2 version update has breaking changes</h1>

Check [Ginkgo V2 Migration Guide](https://onsi.github.io/ginkgo/MIGRATING_TO_V2) for breaking changes.

</aside>

Replace:

```go
	. "github.com/onsi/ginkgo"
```

With:

```go
	. "github.com/onsi/ginkgo/v2"
```

Also, adjust your test suite.

For Controller Suite:

```go
	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
```

With:

```go
	RunSpecs(t, "Controller Suite")
```

For Webhook Suite:

```go
	RunSpecsWithDefaultAndCustomReporters(t,
		"Webhook Suite",
		[]Reporter{printer.NewlineReporter{}})
```

With:

```go
	RunSpecs(t, "Webhook Suite")
```

Last but not least, remove the timeout variable from the `BeforeSuite` blocks:

Replace:

```go
var _ = BeforeSuite(func(done Done) {
	....
}, 60)
```

With


```go
var _ = BeforeSuite(func(done Done) {
	....
})
```



#### Change Logger to use flag options

In the `main.go` file replace:

```go
flag.Parse()

ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
```

With:

```go
opts := zap.Options{
	Development: true,
}
opts.BindFlags(flag.CommandLine)
flag.Parse()

ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
```

#### Rename the manager flags

The manager flags `--metrics-addr` and `enable-leader-election` were renamed to `--metrics-bind-address` and `--leader-elect` to be more aligned with core Kubernetes Components. More info: [#1839][issue-1893].

In your `main.go` file replace:


```go
func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
```

With:

```go
func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
```

And then, rename the flags in the `config/default/manager_auth_proxy_patch.yaml` and `config/default/manager.yaml`:

```yaml
- name: manager
args:
- "--health-probe-bind-address=:8081"
- "--metrics-bind-address=127.0.0.1:8080"
- "--leader-elect"
```

#### Verification

Finally, we can run `make` and `make docker-build` to ensure things are working
fine.

## Change your project to remove the Kubernetes deprecated API versions usage

<aside class="note">
<h1>Before continuing</h1>

Make sure you understand [Versions in CustomResourceDefinitions][custom-resource-definition-versioning].

</aside>


The following steps describe a workflow to upgrade your project to remove the deprecated Kubernetes APIs: `apiextensions.k8s.io/v1beta1`, `admissionregistration.k8s.io/v1beta1`, `cert-manager.io/v1alpha2`.

The Kubebuilder CLI tool does not support scaffolded resources for both Kubernetes API versions such as; an API/CRD with `apiextensions.k8s.io/v1beta1` and another one with `apiextensions.k8s.io/v1`.

<aside class="note">
<h1>Cert Manager API</h1>

If you scaffold a webhook using the Kubernetes API `admissionregistration.k8s.io/v1` then, by default, it will use the API `cert-manager.io/v1` in the manifests.

</aside>

The first step is to update your `PROJECT` file by replacing the `api.crdVersion:v1beta` and `webhooks.WebhookVersion:v1beta` with `api.crdVersion:v1` and `webhooks.WebhookVersion:v1` which would look like:

```yaml
domain: my.domain
layout: go.kubebuilder.io/v3
projectName: example
repo: example
resources:
- api:
    crdVersion: v1
    namespaced: true
  group: webapp
  kind: Guestbook
  version: v1
  webhooks:
    defaulting: true
    webhookVersion: v1
version: "3"
```

You can try to re-create the APIS(CRDs) and Webhooks manifests by using the `--force` flag.

<aside class="note warning">
<h1>Before re-create</h1>

Note, however, that the tool will re-scaffold the files which means that you will lose their content.

Before executing the commands ensure that you have the files content stored in another place. An easy option is to use `git` to compare your local change with the previous version to recover the contents.

</aside>

Now, re-create the APIS(CRDs) and Webhooks manifests by running the  `kubebuilder create api` and `kubebuilder create webhook` for the same group, kind and versions with the flag `--force`, respectively.


[migration-guide-v2-to-v3]: migration_guide_v2tov3.md
[envtest]: https://book.kubebuilder.io/reference/testing/envtest.html
[controller-releases]: https://github.com/kubernetes-sigs/controller-runtime/releases
[issue-1893]: https://github.com/kubernetes-sigs/kubebuilder/issues/1839
[plugins-doc]: /reference/cli-plugins.md
[migration-v2vsv3]: v2vsv3.md
[custom-resource-definition-versioning]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/
[issue-1999]: https://github.com/kubernetes-sigs/kubebuilder/issues/1999
[project-customizations]: v2vsv3.md#project-customizations
[doc-envtest]:/reference/envtest.md
