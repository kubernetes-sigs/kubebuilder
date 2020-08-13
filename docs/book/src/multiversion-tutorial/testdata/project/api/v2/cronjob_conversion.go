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

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"tutorial.kubebuilder.io/project/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
我们的 "spoke" 版本需要实现
[`Convertible`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Convertible)
接口。顾名思义，它需要实现 `ConvertTo` 从（其它版本）向 hub 版本转换，`ConvertFrom` 实现从 hub 版本转换到（其他版本）。
*/

/*
ConvertTo 期望修改其参数以包含转换后的对象。
大部分转换都是直接赋值，除了那些发生变化的 field。
*/
// ConvertTo 转换 CronJob 到 Hub 版本 (v1).
func (src *CronJob) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1.CronJob)

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
		剩下的转换都相当机械。
	*/
	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.StartingDeadlineSeconds = src.Spec.StartingDeadlineSeconds
	dst.Spec.ConcurrencyPolicy = v1.ConcurrencyPolicy(src.Spec.ConcurrencyPolicy)
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
ConvertFrom 期望修改其接收者以包含转换后的对象。
大部分转换都是直接赋值，除了那些发生变化的 field。
*/

// ConvertFrom 从 Hub 版本 (v1) 转换到这个版本。
func (dst *CronJob) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1.CronJob)

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
