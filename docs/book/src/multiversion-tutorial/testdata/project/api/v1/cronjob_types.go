/*
Copyright 2025 The Kubernetes authors.

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
 */

package v1

/*
 */

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:docs-gen:collapse=Imports

// CronJobSpec defines the desired state of CronJob
type CronJobSpec struct {
	// schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	// +kubebuilder:validation:MinLength=0
	// +required
	Schedule string `json:"schedule"`

	// startingDeadlineSeconds defines in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	// +optional
	// +kubebuilder:validation:Minimum=0
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// concurrencyPolicy specifies how to treat concurrent executions of a Job.
	// Valid values are:
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	// +optional
	// +kubebuilder:default:=Allow
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// suspend tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// jobTemplate defines the job that will be created when executing a CronJob.
	// +required
	JobTemplate batchv1.JobTemplateSpec `json:"jobTemplate"`

	// successfulJobsHistoryLimit defines the number of successful finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	// +kubebuilder:validation:Minimum=0
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// failedJobsHistoryLimit defines the number of failed finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	// +kubebuilder:validation:Minimum=0
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`
}

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

// CronJobStatus defines the observed state of CronJob.
type CronJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// active defines a list of pointers to currently running jobs.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	Active []corev1.ObjectReference `json:"active,omitempty"`

	// lastScheduleTime defines when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the CronJob resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:docs-gen:collapse=old stuff

/*
 Since we'll have more than one version, we'll need to mark a storage version.
 This is the version that the Kubernetes API server uses to store our data.
 We'll chose the v1 version for our project.

 We'll use the [`+kubebuilder:storageversion`](/reference/markers/crd.md) to do this.

 Note that multiple versions may exist in storage if they were written before
 the storage version changes -- changing the storage version only affects how
 objects are created/updated after the change.
*/

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +versionName=v1
// +kubebuilder:storageversion
// CronJob is the Schema for the cronjobs API
type CronJob struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of CronJob
	// +required
	Spec CronJobSpec `json:"spec"`

	// status defines the observed state of CronJob
	// +optional
	Status CronJobStatus `json:"status,omitempty,omitzero"`
}

/*
 */

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

// +kubebuilder:docs-gen:collapse=old stuff
