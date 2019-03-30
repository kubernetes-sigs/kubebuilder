
Copyright 2019 The Kubernetes authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

{% method %}

There are two types of files in our `api/v1` package: `doc.go` (this file)
and `*_types.go` files.

This file (`doc.go`) contains some common setup logic.  Namely, it sets up a
*SchemeBuilder* which will help us set up the type-Kind mappings to add to
our Scheme in main.go, and provides a nice constant representing the API Group
and Version of this package.

We have some marker comments on the package that tell our code generators in
controller-tools what API group this corresponds to, so that they can generate
interface definitions, CRD YAML, etc.

Our scaffolding tool will create all of this for us, and it basically never
has to change.

What we actually have to fill out is our API types, in `cronjob_types.go`.


```go


//go:generate go run ../../vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy -i ./... -h ../../hack/boilerplate.go.txt

// Package v1 contains API Schema definitions for the crew v1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=tutorial.kubebuilder.io
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	GroupVersion = schema.GroupVersion{Group: "tutorial.kubebuilder.io", Version: "v1"}
	schemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme = schemeBuilder.AddToScheme
)

```
{%% endmethod %%}


[Next](./page-2.md)
