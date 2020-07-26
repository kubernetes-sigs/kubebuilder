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

package v1

import (
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:docs-gen:collapse=Imports

/*
首先，让我们来看看 spec。正如我们之前讨论过的，spec 代表所期望的状态，所以控制器的任何 "输入" 都会在这里。

通常来说，CronJob 由以下几部分组成：

- 一个时间表（ CronJob 中的 cron ）
- 要运行的 Job 模板（ CronJob 中的 Job ）

当然 CronJob 还需要一些额外的东西，使得它更加易用

- 一个已经启动的 Job 的超时时间（如果该 Job 执行超时，那么我们会将在下次调度的时候重新执行该 Job）。
- 如果多个 Job 同时运行，该怎么办（我们要等待吗？还是停止旧的 Job ？）
- 暂停 CronJob 运行的方法，以防出现问题。
- 对旧 Job 历史的限制

请记住，由于我们从不读取自己的状态，我们需要有一些其他的方法来跟踪一个 Job 是否已经运行。我们可以使用至少一个旧的 Job 来做这件事。

我们将使用几个标记（`// +comment`）来指定额外的元数据。在生成 CRD 清单时，[controller-tools](https://github.com/kubernetes-sigs/controller-tools) 将使用这些数据。我们稍后将看到，controller-tools 也将使用 GoDoc 来生成字段的描述。
*/

// CronJobSpec defines the desired state of CronJob
type CronJobSpec struct {
	// +kubebuilder:validation:MinLength=0

	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`

	// +kubebuilder:validation:Minimum=0

	// Optional deadline in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// Specifies how to treat concurrent executions of a Job.
	// Valid values are:
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	// +optional
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// Specifies the job that will be created when executing a CronJob.
	JobTemplate batchv1beta1.JobTemplateSpec `json:"jobTemplate"`

	// +kubebuilder:validation:Minimum=0

	// The number of successful finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// +kubebuilder:validation:Minimum=0

	// The number of failed finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`
}

/*
我们定义了一个自定义类型来保存我们的并发策略。实际上，它的底层类型是 string，但该类型给出了额外的文档，并允许我们在类型上附加验证，而不是在字段上验证，使验证逻辑更容易复用。
*/

// ConcurrencyPolicy describes how the job will be handled.
// Only one of the following concurrent policies may be specified.
// If none of the following policies is specified, the default one
// is AllowConcurrent.
// +kubebuilder:validation:Enum=Allow;Forbid;Replace
type ConcurrencyPolicy string

const (
	// AllowConcurrent allows CronJobs to run concurrently.
	AllowConcurrent ConcurrencyPolicy = "Allow"

	// ForbidConcurrent forbids concurrent runs, skipping next run if previous
	// hasn't finished yet.
	ForbidConcurrent ConcurrencyPolicy = "Forbid"

	// ReplaceConcurrent cancels currently running job and replaces it with a new one.
	ReplaceConcurrent ConcurrencyPolicy = "Replace"
)

/*
接下来，让我们设计一下我们的 status，它表示实际看到的状态。它包含了我们希望用户或其他控制器能够轻松获得的任何信息。

我们将保存一个正在运行的 Jobs，以及我们最后一次成功运行 Job 的时间。注意，我们使用 `metav1.Time` 而不是 `time.Time` 来保证序列化的兼容性以及稳定性，如上所述。
*/

// CronJobStatus defines the observed state of CronJob
type CronJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// A list of pointers to currently running jobs.
	// +optional
	Active []corev1.ObjectReference `json:"active,omitempty"`

	// Information when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

/*
最后，CronJob 和 CronJobList 直接使用模板生成的即可。如前所述，我们不需要改变这个，除了标记我们想要一个有状态子资源，这样我们的行为就像内置的 kubernetes 类型。
*/

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CronJob is the Schema for the cronjobs API
type CronJob struct {
	/*
	 */
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronJobSpec   `json:"spec,omitempty"`
	Status CronJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob
type CronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}

// +kubebuilder:docs-gen:collapse=Root Object Definitions
