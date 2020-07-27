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
我们非常简单地开始：我们导入`meta/v1` API 组，通常本身并不会暴露该组，而是包含所有 Kubernetes 种类共有的元数据。
*/
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
下一步，我们为种类的 Spe c和 Status 定义类型。Kubernetes 功能通过使期待的状态(`Spec`)和实际集群状态(其他对象的
`Status`)保持一致和外部状态，然后记录观察到的状态(`Status`)。
因此，每个 *functional* 对象包括 spec 和 status 。很少的类型，像 `ConfigMap` 不需要遵从这个模式，因为它们不编码期待的状态，
但是大部分类型需要做这一步。
*/
// 编辑这个文件！这是你拥有的脚手架！
// 注意: json 标签是必需的。为了能够序列化字段，任何你添加的新的字段一定有json标签。

// CronJobSpec 定义了 CronJob 期待的状态
type CronJobSpec struct {
	// 插入额外的 SPEC 字段 - 集群期待的状态
	// 重要：修改了这个文件之后运行"make"去重新生成代码
}

// CronJobStatus 定义了 CronJob 观察的的状态
type CronJobStatus struct {
	// 插入额外的 STATUS 字段 - 定义集群观察的状态
	// 重要：修改了这个文件之后运行"make"去重新生成代码
}

/*
下一步，我们定义与实际种类相对应的类型，`CronJob` 和 `CronJobList` 。
`CronJob` 是一个根类型, 它描述了 `CronJob` 种类。像所有 Kubernetes 对象，它包含
`TypeMeta` (描述了API版本和种类)，也包含其中拥有像名称,名称空间和标签的东西的 `ObjectMeta` 。

`CronJobList` 只是多个 `CronJob` 的容器。它是批量操作中使用的种类，像 LIST 。

通常情况下，我们从不修改任何一个 -- 所有修改都要到 Spec 或者 Status 。

那个小小的 `+kubebuilder:object:root` 注释被称为标记。我们将会看到更多的它们，但要知道它们充当额外的元数据，
告诉[controller-tools](https://github.com/kubernetes-sigs/controller-tools)(我们的代码和YAML生成器)额外的信息。
这个特定的标签告诉 `object `生成器这个类型表示一个种类。然后，`object` 生成器为我们生成
这个所有表示种类的类型一定要实现的[runtime.Object](https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime?tab=doc#Object)接口的实现。
*/
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// CronJob 是 cronjobs API 的 Schema
type CronJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronJobSpec   `json:"spec,omitempty"`
	Status CronJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// CronJobList 包含了一个 CronJob 的列表
type CronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronJob `json:"items"`
}

/*
最后，我们将这个 Go 类型添加到 API 组中。这允许我们将这个 API 组中的类型可以添加到任何[Scheme](https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime?tab=doc#Scheme)。
*/
func init() {
	SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}
