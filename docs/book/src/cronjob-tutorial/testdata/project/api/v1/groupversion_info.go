/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// +kubebuilder:docs-gen:collapse=Apache License

/* 
首先，我们又一些*包级别*的标记用来表示这个包中的Kubernetes对象，这个包代表`batch.tutorial.kubebuilder.io`这个group。
`object` 生成器利用前者，后者被CRD生成器用来从这个包中构建出CRDs的元数据。
*/

// 包v1 包含了batch v1 API 这个group的API Schema 定义。
// +kubebuilder:object:generate=true
// +groupName=batch.tutorial.kubebuilder.io
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

/*
然后，我们有一些常见且常用的变量来帮助我们设置我们的Scheme。因为我们需要在这个包的controller中用到所有的类型，
用一个方便的方法给其他 `Scheme` 来添加所有的类型，是非常有用的(而且也是一种惯例)。SchemeBuilder能够帮助我们轻松的实现这个事情。
*/

var (
	// GroupVersion 是用来注册这些对象的group version。
	GroupVersion = schema.GroupVersion{Group: "batch.tutorial.kubebuilder.io", Version: "v1"}

	// SchemeBuilder 被用来给 GroupVersionKind scheme 添加go类型。
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme 将group-version中的类型添加到指定的scheme中。
	AddToScheme = SchemeBuilder.AddToScheme
)
