/*
Copyright 2023 The Kubernetes Authors.

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

package multiversion

const importV2 = `import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)`

const importReplacement = `/*
 */
import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)`

// FIXME: We should just insert and replace what is need and not a block of code in this way
const cronjobSpecMore = `// startingDeadlineSeconds defines in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	// +optional
	// +kubebuilder:validation:Minimum=0
	StartingDeadlineSeconds *int64 ` + "`json:\"startingDeadlineSeconds,omitempty\"`" + `

	// concurrencyPolicy defines how to treat concurrent executions of a Job.
	// Valid values are:
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	// +optional
	// +kubebuilder:default:=Allow
	ConcurrencyPolicy ConcurrencyPolicy ` + "`json:\"concurrencyPolicy,omitempty\"`" + `

	// suspend tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool ` + "`json:\"suspend,omitempty\"`" + `

	// jobTemplate defines the job that will be created when executing a CronJob.
	// +required
	JobTemplate batchv1.JobTemplateSpec ` + "`json:\"jobTemplate\"`" + `

	// successfulJobsHistoryLimit defines the number of successful finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	// +kubebuilder:validation:Minimum=0
	SuccessfulJobsHistoryLimit *int32 ` + "`json:\"successfulJobsHistoryLimit,omitempty\"`" + `

	// failedJobsHistoryLimit defines the number of failed finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	// +kubebuilder:validation:Minimum=0
	FailedJobsHistoryLimit *int32 ` + "`json:\"failedJobsHistoryLimit,omitempty\"`" + `

	// +kubebuilder:docs-gen:collapse=The rest of Spec

}

/*
Next, we'll need to define a type to hold our schedule.
Based on our proposed YAML above, it'll have a field for
each corresponding Cron "field".
*/

// describes a Cron schedule.
type CronSchedule struct {
	// minute specifies the minutes during which the job executes.
	// +optional
	Minute *CronField ` + "`json:\"minute,omitempty\"`" + `
	// hour specifies the hour during which the job executes.
	// +optional
	Hour *CronField ` + "`json:\"hour,omitempty\"`" + `
	// dayOfMonth specifies the day of the month during which the job executes.
	// +optional
	DayOfMonth *CronField ` + "`json:\"dayOfMonth,omitempty\"`" + `
	// month specifies the month during which the job executes.
	// +optional
	Month *CronField ` + "`json:\"month,omitempty\"`" + `
	// dayOfWeek specifies the day of the week during which the job executes.
	// +optional
	DayOfWeek *CronField ` + "`json:\"dayOfWeek,omitempty\"`" + `
}

/*
Finally, we'll define a wrapper type to represent a field.
We could attach additional validation to this field,
but for now we'll just use it for documentation purposes.
*/

// represents a Cron field specifier.
type CronField string

/*
All the other types will stay the same as before.
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
`
