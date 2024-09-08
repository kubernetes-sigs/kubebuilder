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

const cronJobFieldsForDefaulting = `
// Default values for various CronJob fields
DefaultConcurrencyPolicy          ConcurrencyPolicy
DefaultSuspend                    bool
DefaultSuccessfulJobsHistoryLimit int32
DefaultFailedJobsHistoryLimit     int32
`

const cronJobDefaultingLogic = `
// Set default values
cronjob.Default()
`

const cronJobDefaultFunction = `
func (r *CronJob) Default() {
	if r.Spec.ConcurrencyPolicy == "" {
		r.Spec.ConcurrencyPolicy = AllowConcurrent
	}
	if r.Spec.Suspend == nil {
		r.Spec.Suspend = new(bool)
	}
	if r.Spec.SuccessfulJobsHistoryLimit == nil {
		r.Spec.SuccessfulJobsHistoryLimit = new(int32)
		*r.Spec.SuccessfulJobsHistoryLimit = 3
	}
	if r.Spec.FailedJobsHistoryLimit == nil {
		r.Spec.FailedJobsHistoryLimit = new(int32)
		*r.Spec.FailedJobsHistoryLimit = 1
	}
}
`

const cronJobValidationFunction = `
func (r *CronJob) validateCronJob() error {
	var allErrs field.ErrorList
	if err := r.validateCronJobName(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := r.validateCronJobSpec(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "batch.tutorial.kubebuilder.io", Kind: "CronJob"},
		r.Name, allErrs)
}

func (r *CronJob) validateCronJobName() *field.Error {
	if len(r.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		return field.Invalid(field.NewPath("metadata").Child("name"), r.Name, "must be no more than 52 characters")
	}
	return nil
}

// validateCronJobSpec validates the schedule format of the custom CronSchedule type
func (r *CronJob) validateCronJobSpec() *field.Error {
	// Build cron expression from the parts
	parts := []string{"*", "*", "*", "*", "*"} // default parts for minute, hour, day of month, month, day of week
	if r.Spec.Schedule.Minute != nil {
		parts[0] = string(*r.Spec.Schedule.Minute)  // Directly cast CronField (which is an alias of string) to string
	}
	if r.Spec.Schedule.Hour != nil {
		parts[1] = string(*r.Spec.Schedule.Hour)
	}
	if r.Spec.Schedule.DayOfMonth != nil {
		parts[2] = string(*r.Spec.Schedule.DayOfMonth)
	}
	if r.Spec.Schedule.Month != nil {
		parts[3] = string(*r.Spec.Schedule.Month)
	}
	if r.Spec.Schedule.DayOfWeek != nil {
		parts[4] = string(*r.Spec.Schedule.DayOfWeek)
	}

	// Join parts to form the full cron expression
	cronExpression := strings.Join(parts, " ")

	return validateScheduleFormat(
		cronExpression,
		field.NewPath("spec").Child("schedule"))
}

func validateScheduleFormat(schedule string, fldPath *field.Path) *field.Error {
	if _, err := cron.ParseStandard(schedule); err != nil {
		return field.Invalid(fldPath, schedule, "invalid cron schedule format: "+err.Error())
	}
	return nil
}
`
