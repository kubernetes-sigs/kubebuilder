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

package v2

/*
For imports, we'll need the controller-runtime
[`conversion`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc)
package, plus the API version for our hub type (v1), and finally some of the
standard packages.
*/

import (
	"fmt"
	"strings"

	"log"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
) // +kubebuilder:docs-gen:collapse=Imports

/*
Our "spoke" versions need to implement the
[`Convertible`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Convertible)
interface. Namely, they'll need `ConvertTo()` and `ConvertFrom()`
methods to convert to/from the hub version.
*/

/*
ConvertTo is expected to modify its argument to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/

// ConvertTo converts this CronJob (v2) to the Hub version (v1).
func (src *CronJob) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*batchv1.CronJob)
	log.Printf("ConvertTo: Converting CronJob from Spoke version v2 to Hub version v1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	sched := src.Spec.Schedule
	scheduleParts := []string{"*", "*", "*", "*", "*"}
	if sched.Minute != nil {
		scheduleParts[0] = string(*sched.Minute)
	}
	if sched.Hour != nil {
		scheduleParts[1] = string(*sched.Hour)
	}
	if sched.DayOfMonth != nil {
		scheduleParts[2] = string(*sched.DayOfMonth)
	}
	if sched.Month != nil {
		scheduleParts[3] = string(*sched.Month)
	}
	if sched.DayOfWeek != nil {
		scheduleParts[4] = string(*sched.DayOfWeek)
	}
	dst.Spec.Schedule = strings.Join(scheduleParts, " ")

	/*
		The rest of the conversion is pretty rote.
	*/
	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.StartingDeadlineSeconds = src.Spec.StartingDeadlineSeconds
	dst.Spec.ConcurrencyPolicy = batchv1.ConcurrencyPolicy(src.Spec.ConcurrencyPolicy)
	dst.Spec.Suspend = src.Spec.Suspend
	dst.Spec.JobTemplate = src.Spec.JobTemplate
	dst.Spec.SuccessfulJobsHistoryLimit = src.Spec.SuccessfulJobsHistoryLimit
	dst.Spec.FailedJobsHistoryLimit = src.Spec.FailedJobsHistoryLimit

	// Status
	dst.Status.Active = src.Status.Active
	dst.Status.LastScheduleTime = src.Status.LastScheduleTime

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

/*
ConvertFrom is expected to modify its receiver to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/

// ConvertFrom converts the Hub version (v1) to this CronJob (v2).
func (dst *CronJob) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*batchv1.CronJob)
	log.Printf("ConvertFrom: Converting CronJob from Hub version v1 to Spoke version v2;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	schedParts := strings.Split(src.Spec.Schedule, " ")
	if len(schedParts) != 5 {
		return fmt.Errorf("invalid schedule: not a standard 5-field schedule")
	}
	partIfNeeded := func(raw string) *CronField {
		if raw == "*" {
			return nil
		}
		part := CronField(raw)
		return &part
	}
	dst.Spec.Schedule.Minute = partIfNeeded(schedParts[0])
	dst.Spec.Schedule.Hour = partIfNeeded(schedParts[1])
	dst.Spec.Schedule.DayOfMonth = partIfNeeded(schedParts[2])
	dst.Spec.Schedule.Month = partIfNeeded(schedParts[3])
	dst.Spec.Schedule.DayOfWeek = partIfNeeded(schedParts[4])

	/*
		The rest of the conversion is pretty rote.
	*/
	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.StartingDeadlineSeconds = src.Spec.StartingDeadlineSeconds
	dst.Spec.ConcurrencyPolicy = ConcurrencyPolicy(src.Spec.ConcurrencyPolicy)
	dst.Spec.Suspend = src.Spec.Suspend
	dst.Spec.JobTemplate = src.Spec.JobTemplate
	dst.Spec.SuccessfulJobsHistoryLimit = src.Spec.SuccessfulJobsHistoryLimit
	dst.Spec.FailedJobsHistoryLimit = src.Spec.FailedJobsHistoryLimit

	// Status
	dst.Status.Active = src.Status.Active
	dst.Status.LastScheduleTime = src.Status.LastScheduleTime

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}
