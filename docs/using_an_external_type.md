# Using an external Type


# Introduction

There are several different external types that may be referenced when writing a controller.
* A Custom Resource Definition (CRD) that is defined in the current project `kubebuilder create api`.
* A Core Kubernetes Resources eg. `kubebuilder create api --group apps --version v1 --kind Deployment`.
* A CRD that is created and installed in another project.
* A CR defined via an API Aggregation (AA). Aggregated APIs are subordinate APIServers that sit behind the primary API server, which acts as a proxy.

Currrently Kubebuilder handles the first two, CRDs and Core Resources, seamlessly.  External CRDs and CRs created via Aggregation must be scaffolded manually.

In order to use a Kubernetes Custom Resource that has been defined in another project
you will need to have several items of information.
* The Domain of the CR
* The Group under the Domain 
* The Go import path of the CR Type definition.

The Domain and Group variables have been discussed in other parts of the documentation.  The import path would be located in the project that installs the CR.

Example API Aggregation directory structure
```
github.com
    ├── example
        ├── myproject
            ├── apis
                ├── mygroup
                   ├── doc.go
                   ├── install
                   │   ├── install.go
                   ├── v1alpha1
                   │   ├── doc.go
                   │   ├── register.go
                   │   ├── types.go
                   │   ├── zz_generated.deepcopy.go
```

In the case above the import path would be `github.com/example/myproject/apis/mygroup/v1alpha1`

### Create a project

```
kubebuilder init --domain $APIDOMAIN --owner "MyCompany"
```

### Add a controller

be sure to answer no when it asks if you would like to create an api? [Y/n]
```
kubebuilder create api --group $APIGROUP --version $APIVERSION --kind MyKind

```

## Edit the Api files.

### Register your Types

Add the following file to the pkg/apis directory

file: pkg/apis/mytype_addtoscheme.go
```
package apis

import (
	mygroupv1alpha1 "github.com/username/myapirepo/apis/mygroup/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects 
	// to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, mygroupv1alpha1.AddToScheme)
}

```

## Edit the Controller files

### Use the correct import for your api

file: pkg/controllers/mytype_controller.go
```
import mygroupv1alpha1 "github.com/example/myproject/apis/mygroup/v1alpha1"

```

### Update dependencies

```
dep ensure --add
```

## Helpful Tips

### Locate your domain and group variables

The following kubectl commands may be useful

```
kubectl api-resources --verbs=list -o name

kubectl api-resources --verbs=list -o name | grep mydomain.com
```

