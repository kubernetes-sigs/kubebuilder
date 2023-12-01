# Using an External Type

There are several different external types that may be referenced when writing a controller.
* Custom Resource Definitions (CRDs) that are defined in the current project (such as via `kubebuilder create api`).
* Core Kubernetes Resources (e.g. Deployments or Pods).
* CRDs that are created and installed in another project.
* A custom API defined via the aggregation layer, served by an extension API server for which the primary API server acts as a proxy.

Currently, kubebuilder handles the first two, CRDs and Core Resources, seamlessly. You must scaffold the latter two, External CRDs and APIs created via aggregation, manually.

In order to use a Kubernetes Custom Resource that has been defined in another project
you will need to have several items of information.
* The Domain of the CR
* The Group under the Domain 
* The Go import path of the CR Type definition
* The Custom Resource Type you want to depend on.

The Domain and Group variables have been discussed in other parts of the documentation.  The import path would be located in the project that installs the CR.
The Custom Resource Type is usually a Go Type of the same name as the CustomResourceDefinition in kubernetes, e.g. for a `Pod` there will be a type `Pod` in the `v1` group.
For Kubernetes Core Types, the domain can be omitted.
``
This document uses `my` and `their` prefixes as a naming convention for repos, groups, and types to clearly distinguish between your own project and the external one you are referencing.

In our example we will assume the following external API Type:

`github.com/theiruser/theirproject` is another kubebuilder project on whose CRD we want to depend and extend on.
Thus, it contains a `go.mod` in its repository root. The import path for the go types would be `github.com/theiruser/theirproject/api/theirgroup/v1alpha1`.

The Domain of the CR is `theirs.com`, the Group is `theirgroup` and the kind and go type would be `ExternalType`.

If there is an interest to have multiple Controllers running in different Groups (e.g. because one is an owned CRD and one is an external Type), please first
reconfigure the Project to use a multi-group layout as described in the [Multi-Group documentation](../migration/multi-group.md).

### Prerequisites

The following guide assumes that you have already created a project using `kubebuilder init` in a directory in the GOPATH. Please reference the [Getting Started Guide](../getting-started.md) for more information.

Note that if you did not pass `--domain` to `kubebuilder init` you will need to modify it for the individual api types as the default is `my.domain`, not `theirs.com`.
Similarly, if you intend to use your own domain, please configure your own domain with `kubebuilder init` and do not use `theirs.com for the domain.

### Add a controller for the external Type

Run the command `create api` to scaffold only the controller to manage the external type:

```shell
kubebuilder create api --group <theirgroup> --version v1alpha1 --kind <ExternalTypeKind> --controller --resource=false
```

Note that the `resource` argument is set to false, as we are not attempting to create our own CustomResourceDefinition,
but instead rely on an external one.

This will result in a `PROJECT` entry with the default domain of the `PROJECT` (`my.domain` if not specified in `kubebuilder init`).
For use of other domains, such as `theirs.com`, one will have to manually adjust the `PROJECT` file with the correct domain for the entry:

<aside class="note">
If you are looking to create Controllers to manage Kubernetes Core types (i.e. Deployments/Pods)y
you do not need to update the PROJECT file or register the Schema in the manager. All Core Types are registered by default. The Kubebuilder CLI will add the required values to the PROJECT file, but you still need to perform changes to the RBAC markers manually to ensure that the Rules will be generated accordingly.
</aside>

file: PROJECT
```
domain: my.domain
layout:
- go.kubebuilder.io/v4
projectName: testkube
repo: example.com
resources:
- controller: true
  domain: my.domain ## <- Replace the domain with theirs.com domain
  group: mygroup
  kind: ExternalType
  version: v1alpha1
version: "3"
```

At the same time, the generated RBAC manifests need to be adjusted:

file: internal/controller/externaltype_controller.go
```go
// ExternalTypeReconciler reconciles a ExternalType object
type ExternalTypeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// external types can be added like this
//+kubebuilder:rbac:groups=theirgroup.theirs.com,resources=externaltypes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=theirgroup.theirs.com,resources=externaltypes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=theirgroup.theirs.com,resources=externaltypes/finalizers,verbs=update
// core types can be added like this
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update
```

### Register your Types

<aside class="note">
Note that this is only valid for external types and not the kubernetes core types.
Core types such as pods or nodes are registered by default in the scheme.
</aside>

Edit the following lines to the main.go file to register the external types:

file: cmd/main.go
```go
package apis

import (
	theirgroupv1alpha1 "github.com/theiruser/theirproject/apis/theirgroup/v1alpha1"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(theirgroupv1alpha1.AddToScheme(scheme)) // this contains the external API types
	//+kubebuilder:scaffold:scheme
} 
```

## Edit the Controller `SetupWithManager` function

### Use the correct imports for your API and uncomment the controlled resource

file: internal/controllers/externaltype_controllers.go
```go
package controllers

import (
	theirgroupv1alpha1 "github.com/theiruser/theirproject/apis/theirgroup/v1alpha1"
)

//...

// SetupWithManager sets up the controller with the Manager.
func (r *ExternalTypeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&theirgroupv1alpha1.ExternalType{}).
		Complete(r)
}

```

Note that core resources may simply be imported by depending on the API's from upstream Kubernetes and do not need additional `AddToScheme` registrations:

file: internal/controllers/externaltype_controllers.go
```go
package controllers
// contains core resources like Deployment
import (
   v1 "k8s.io/api/apps/v1"
)


// SetupWithManager sets up the controller with the Manager.
func (r *ExternalTypeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		Complete(r)
}
```

### Update dependencies

```
go mod tidy
```

### Generate RBACs with updated Groups and Resources

```
make manifests
``` 

## Prepare for testing

### Register your resource in the Scheme

Edit the `CRDDirectoryPaths` in your test suite and add the correct `AddToScheme` entry during suite initialization:

file: internal/controllers/suite_test.go
```go
package controller

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
	theirgroupv1alpha1 "github.com/theiruser/theirproject/apis/theirgroup/v1alpha1"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}


var _ = BeforeSuite(func() {
	//...
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{
			// if you are using vendoring and rely on a kubebuilder based project, you can simply rely on the vendored config directory 
			filepath.Join("..", "..", "..", "vendor", "github.com", "theiruser", "theirproject", "config", "crds"),
			// otherwise you can simply download the CRD from any source and place it within the config/crd/bases directory,
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: false,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.28.3-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}
	
	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	//+kubebuilder:scaffold:scheme
    Expect(theirgroupv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())


})

```

### Verifying API Availability in the Cluster

Since we are now using external types, you will now have to rely on them being installed into the cluster.
If the APIs are not available at the time the manager starts, all informers listening to the non-available types
will fail, causing the manager to exit with an error similar to

```
failed to get informer from cache       {"error": "Timeout: failed waiting for *v1alpha1.ExternalType Informer to sync"}
```

This will signal that the API Server is not yet ready to serve the external types.

## Helpful Tips

### Locate your domain and group variables

The following kubectl commands may be useful

```shell
kubectl api-resources --verbs=list -o name
kubectl api-resources --verbs=list -o name | grep my.domain
```

