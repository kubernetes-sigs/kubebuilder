/*
Copyright 2019 The Kubernetes Authors.

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

/*
Every Go type that represents a Kubernetes Kind has to have some common set
of fields.  Those common fields live in the meta/v1 package, so we'll need to
import that.  We'll also want some helper types from the core Kubernetes API
(commonly just called v1).
*/
package v1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
Here we'll actually register our types with the scheme.  Most Kinds in Kubernetes
come in pairs: the actual Kind, and the list of that Kind (returned from list operations).

Since we've set up our SchemeBuilder to be for the tutorial.kubebuilder.io/v1 API group,
we can just register the types.  The SchemeBuilder will automatically infer that these
types correspond to kinds of the same name (i.e. "CronJob" and "CronJobList").
*/
func init() {
	schemeBuilder.Register(&CronJob{}, &CronJobList{})
}

/*
Every type that corresponds to a Kubernetes Kind has some comment set of
fields. These fields fall into two categories: metadata about the type and API
group/version (which we call `TypeMeta`), and metadata about the object itself
(like name, annotations, etc), which is slightly different for normal types and
list types (`ObjectMeta` and `ListMeta`, respectively).  These fields look
*identical* across all Kubernetes objects.

Our scaffolding tool will create the basic skeleton for both of these types.

Within our Go code, we mark every such type with an interface:
`runtime.Object`.  That interface has practical uses -- it lets us deep-copy
objects, and manipulate their type metadata.  However, it also serves as
a marker that some Go type actually corresponds to something that can be
serialized on the wire.  We autogenerate the deep-copy for performance reasons.
*/

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronJob represents the configuration of a single cron job.
type CronJob struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" "`

/*
*Most* Kubernetes objects have the same basic structure, beyond their metadata:
a spec and a status. The spec contains information about how different parts of
the system *should* behave.  For example, our CronJob spec will contain
information about the schedule in conforms to, and the jobs it should create

The status, on the other hand, contains information about the last observed
state of the system. No information should be stored in the status that can't
be retreived from looking at the system -- if the status suddenly gets wiped,
your controller shouldn't care.
*/

	// Specification of the desired behavior of a cron job, including the schedule.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec CronJobSpec `json:"spec,omitempty" "`

	// Current status of a cron job.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Status CronJobStatus `json:"status,omitempty" "`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

/*
The list type is fairly simple, and looks the same across all object: it has metadata,
as well as an items field containing the Kubernetes objects that it refers to.
*/
// CronJobList is a collection of cron jobs.
type CronJobList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" "`

	// items is the list of CronJobs.
	Items []CronJob `json:"items" "`
}

/*
For our spec, we'll need to store all the information that we need to run and manage our jobs:

- A schedule
- A deadline for starting the jobs
- What to do when multiple jobs would run at once
- A way to stop our controller
- The actual job to run
- Limits on the number of old jobs to keep around.

Each field has some information about how to serialize and deserialize it to/from JSON/YAML.
Mostly, this is just the name of the field (all Kubernetes JSON fields have a lower-case first letter,
and are `camelCase`).  However, if a field is optional, we mark it with a `// +optional` comment,
and put `omitempty` in its JSON struct tag.

We can also add information here about validation, and how to pretty-print with `kubectl get` (TODO).
*/
// CronJobSpec describes how the job execution will look like and when it will actually run.
type CronJobSpec struct {

	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule" "`

	// Optional deadline in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty" "`

	// Specifies how to treat concurrent executions of a Job.
	// Valid values are:
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	// +optional
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty" "`

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty" "`

	// Specifies the job that will be created when executing a CronJob.
	JobTemplate JobTemplateSpec `json:"jobTemplate" "`

	// The number of successful finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty" "`

	// The number of failed finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty" "`
}

// ConcurrencyPolicy describes how the job will be handled.
// Only one of the following concurrent policies may be specified.
// If none of the following policies is specified, the default one
// is AllowConcurrent.
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
Similarly, for our status, we'll want to note down the last time we ran a job,
and which jobs we think are currently running.  Status fields are almost always
optional, because they won't be populated until after our object is first
created (the first time it gets reconciled).
*/
// CronJobStatus represents the current state of a cron job.
type CronJobStatus struct {
	// A list of pointers to currently running jobs.
	// +optional
	Active []v1.ObjectReference `json:"active,omitempty" "`

	// Information when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty" "`
}

/*
Finally, we'll define a type to hold the template for creating our Jobs.  It
references the actual Job's spec from batch/v1, plus some additional metadata
to inject into the Job.
*/

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobTemplate describes a template for creating copies of a predefined pod.
type JobTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" "`

	// Defines jobs that will be created from this template.
	// https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Template JobTemplateSpec `json:"template,omitempty" "`
}

// JobTemplateSpec describes the data a Job should have when created from a template
type JobTemplateSpec struct {
	// Standard object's metadata of the jobs created from this template.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" "`

	// Specification of the desired behavior of the job.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec JobSpec `json:"spec,omitempty" "`
}

/*
Now that we've defined our API types, we can move on to the code that makes our
CronJobs tick: the reconciler.

+goto /controllers/cronjob_controller.go
*/
