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
因为我们现在在v2 包中，controller-gen 将自动假设这是对于 v2 版本的。
我们可以用[`+versionName`marker](/reference/markers/crd.md)去重写它。
*/
package v2

/*
 */
import (
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 编辑这个文件！这是你拥有的脚手架！
// 注意: json 标签是必需的。为了字段能够被序列化任何你添加的新的字段一定有 json 标签。

// +kubebuilder:docs-gen:collapse=Imports

/*
除了将 schedule 字段更改为一个新类型外，我们将基本上保持 spec 不变。
*/
// CronJobSpec 定义了 CronJob 期待的状态
type CronJobSpec struct {
	// Cron 格式的 schedule，详情请看https://en.wikipedia.org/wiki/Cron。
	Schedule CronSchedule `json:"schedule"`

	/*
	 */

	// +kubebuilder:validation:Minimum=0

	// 对于开始 job 以秒为单位的可选的并如果由于任何原因错失了调度的时间截止日期。未执行的
	// job 将被统计为失败的 job 。
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// 指定如何处理job的并发执行。
	// 有效的值是：
	// - "Allow" (默认)： 允许 CronJobs 并发执行；
	// - "Forbid"：禁止并发执行，如果之前运行的还没有完成，跳过下一次执行；
	// - "Replace"： 取消当前正在运行的 job 并用新的 job 替换它
	// +optional
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// 此标志告诉控制器暂停后续执行，它不会应用到已经开始执行的 job 。默认值是 false。
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// 指定当执行一个 CronJob 时将会被创建的 job 。
	JobTemplate batchv1beta1.JobTemplateSpec `json:"jobTemplate"`

	// +kubebuilder:validation:Minimum=0

	// 要保留的成功完成的 jobs 的数量。
	// 这是一个用来区分明确 0 值和未指定的指针。
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// +kubebuilder:validation:Minimum=0

	// 要保留的失败的 jobs 的数量。
	// 这是一个用来区分明确 0 值和未指定的指针。
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`

	// +kubebuilder:docs-gen:collapse=The rest of Spec
}

/*
接下来，我们定义一个类型存储我们的 schedule 。
基于我们上面提议的 YAML 格式，每个对应的 Cron “field” 都有一个字段。
*/

// 描述一个Cron schedule。
type CronSchedule struct {
	// 指定 job 执行的分钟数。
	// +optional
	Minute *CronField `json:"minute,omitempty"`
	// 指定 job 执行的小时数。
	// +optional
	Hour *CronField `json:"hour,omitempty"`
	// 指定 job 执行的月的天数。
	// +optional
	DayOfMonth *CronField `json:"dayOfMonth,omitempty"`
	// 指定 job 执行的月数。
	// +optional
	Month *CronField `json:"month,omitempty"`
	// 指定 job 执行的一周的天数。
	// +optional
	DayOfWeek *CronField `json:"dayOfWeek,omitempty"`
}

/*
最后，我们定义一个封装器类型来表示一个字段。
我们可以为这个字段附加一些额外的验证，但是现在我们只仅仅用它做文档的目的。
*/

// 表示一个 Cron 字段说明符。
type CronField string

/*
所有其他类型将保持与以前相同。
*/

// ConcurrencyPolicy 描述 job 将会被怎样处理。仅仅下面并发策略中的一种可以被指定。
// 如果没有指定下面策略的任何一种，那么默认的一个是 AllowConcurrent 。
// +kubebuilder:validation:Enum=Allow;Forbid;Replace
type ConcurrencyPolicy string

const (
	// AllowConcurrent 允许 CronJobs 并发执行.
	AllowConcurrent ConcurrencyPolicy = "Allow"

	// ForbidConcurrent 禁止并发执行, 如果之前运行的还没有完成，跳过下一次执行
	ForbidConcurrent ConcurrencyPolicy = "Forbid"

	// ReplaceConcurrent 取消当前正在运行的 job 并用新的 job 替换它。
	ReplaceConcurrent ConcurrencyPolicy = "Replace"
)

// CronJobStatus 定义了 CronJob 观察的的状态
type CronJobStatus struct {
	// 插入额外的 STATUS 字段 - 定义集群观察的状态
	// 重要：修改了这个文件之后运行"make"去重新生成代码

	// 一个存储当前正在运行 job 的指针列表。
	// +optional
	Active []corev1.ObjectReference `json:"active,omitempty"`

	// 当 job 最后一次成功被调度的信息。
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

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

func init() {
	SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}

// +kubebuilder:docs-gen:collapse=Other Types
