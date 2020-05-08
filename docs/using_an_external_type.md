# Using an External Type


# Introduction

There are several different external types that may be referenced when writing a controller.
* A Custom Resource Definition (CRD) that is defined in the current project `kubebuilder create api`.
* A Core Kubernetes Resources eg. `kubebuilder create api --group apps --version v1 --kind Deployment`.
* A CRD that is created and installed in another project.
* A CR defined via an API Aggregation (AA). Aggregated APIs are subordinate APIServers that sit behind the primary API server, which acts as a proxy.

Currently Kubebuilder handles the first two, CRDs and Core Resources, seamlessly.  External CRDs and CRs created via Aggregation must be scaffolded manually.

In order to use a Kubernetes Custom Resource that has been defined in another project
you will need to have several items of information.
* The Domain of the CR
* The Group under the Domain 
* The Go import path of the CR Type definition.

The Domain and Group variables have been discussed in other parts of the documentation.  The import path would be located in the project that installs the CR.

This document uses `my` and `their` prefixes as a naming convention for repos, groups, and types to clearly distinguish between your own project and the external one you are referencing.

Example external API Aggregation directory structure
```
github.com
    ├── theiruser
        ├── theirproject
            ├── apis
                ├── theirgroup
                   ├── doc.go
                   ├── install
                   │   ├── install.go
                   ├── v1alpha1
                   │   ├── doc.go
                   │   ├── register.go
                   │   ├── types.go
                   │   ├── zz_generated.deepcopy.go
```

In the case above the import path would be `github.com/theiruser/theirproject/apis/theirgroup/v1alpha1`

### Create a project

```
kubebuilder init --domain $APIDOMAIN --owner "MyCompany"
```

### Add a controller

be sure to answer no when it asks if you would like to create an api? [Y/n]
```
kubebuilder create api --group mygroup --version $APIVERSION --kind MyKind

```

## Edit the API files.

### Register your Types

Edit the following file to the pkg/apis directory to append their `AddToScheme` to your `AddToSchemes`:

file: pkg/apis/mytype_addtoscheme.go
```
package apis

import (
	mygroupv1alpha1 "github.com/myuser/myrepo/apis/mygroup/v1alpha1"
	theirgroupv1alpha1 "github.com/theiruser/theirproject/apis/theirgroup/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects 
	// to GroupVersionKinds and back
	AddToSchemes = append(
	  AddToSchemes, 
	  mygroupv1alpha1.SchemeBuilder.AddToScheme,
	  theirgroupv1alpha1.SchemeBuilder.AddToScheme,
	)
}

```

## Edit the Controller files

### Use the correct imports for your API

file: pkg/controllers/mytype_controller.go
```
import (
	mygroupv1alpha1 "github.com/myuser/myrepo/apis/mygroup/v1alpha1"
	theirgroupv1alpha1 "github.com/theiruser/theirproject/apis/theirgroup/v1alpha1"
)
```

### Update dependencies

```
dep ensure --add
```

## Prepare for testing

#### Register your resource

Edit the `CRDDirectoryPaths` in your test suite by appending the path to their CRDs:

file pkg/controllers/my_kind_controller_suite_test.go
```
var cfg *rest.Config

func TestMain(m *testing.M) {
	// Get a config to talk to the apiserver
	t := &envtest.Environment{
		Config:             cfg,
		CRDDirectoryPaths:  []string{
		  filepath.Join("..", "..", "..", "config", "crds"),
		  filepath.Join("..", "..", "..", "vendor", "github.com", "theiruser", "theirproject", "config", "crds"),
        },
		UseExistingCluster: true,
	}

	apis.AddToScheme(scheme.Scheme)

	var err error
	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}

	code := m.Run()
	t.Stop()
	os.Exit(code)
}

```

## Helpful Tips

### Locate your domain and group variables

The following kubectl commands may be useful

```
kubectl api-resources --verbs=list -o name

kubectl api-resources --verbs=list -o name | grep mydomain.com
```

