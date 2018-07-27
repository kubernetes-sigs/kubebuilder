# Connect to an Existing Kind

### Locate your domain and group variables

The following kubectl commands may be useful

```
kubectl api-resources --verbs=list -o name

kubectl api-resources --verbs=list -o name | grep mydomain.com
```

### Create a project

```
kubebuilder init --domain $APIDOMAIN --owner "MyCompany"
```

### Add a controller

be sure to answer no when it asks if you would like to crate an api? [Y/n]
```
kubebuilder create api --group $APIGROUP --version $APIVERSION --kind MyKind

```

### Register your Types

Add the following file to the pkg/apis directory

file: pkg/apis/mytype_type.go
```
package apis

import (
	"github.com/username/myapirepo/apis/mygroup/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

func init() {
	// schemeGroupVersion is group version used to register these objects
	schemeGroupVersion := schema.GroupVersion{Group: "mygroup.mydomain.com", Version: "v1alpha1"}

	// schemeBuilder is used to add go types to the GroupVersionKind scheme
	schemeBuilder := &scheme.Builder{GroupVersion: schemeGroupVersion}
	schemeBuilder.Register(&v1alpha1.MyKind{},
		&v1alpha1.MyKindList{})

	// Register the types with the Scheme so the components can map objects 
	// to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, schemeBuilder.AddToScheme)
}

```

### Use the correct import for your api

```
import (
mygroupv1alpha1 "mydomain.com/myapiproject/apis/mygroup/v1alpha1"

...
```
