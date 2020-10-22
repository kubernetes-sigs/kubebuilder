/*
Copyright 2020 The Kubernetes authors.

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
首先，我们有一些*包*级别的标记的标记,表示存在这个包中的 Kubernetes 对象，并且这个包表示 `batch.tutorial.kubebuilder.io` 组。`object` 生成器使用前者，而后者是由 CRD 生成器来生成的，它会从这个包创建 CRD 的元数据。
*/

// 包 v1 包含了 batch v1 API 这个组的 API Schema 定义。
// +kubebuilder:object:generate=true
// +groupName=batch.tutorial.kubebuilder.io
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

/*
然后，我们有一些常见且常用的变量来帮助我们设置我们的 Scheme 。因为我们需要在这个包的 controller 中用到所有的类型，
用一个方便的方法给其他 `Scheme` 来添加所有的类型，是非常有用的(而且也是一种惯例)。SchemeBuilder 能够帮助我们轻松的实现这个事情。
*/

var (
	// GroupVersion 是用来注册这些对象的 group version。
	GroupVersion = schema.GroupVersion{Group: "batch.tutorial.kubebuilder.io", Version: "v1"}

	// SchemeBuilder 被用来给 GroupVersionKind scheme 添加 go 类型。
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme 将 group-version 中的类型添加到指定的 scheme 中。
	AddToScheme = SchemeBuilder.AddToScheme
)
