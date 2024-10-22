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

const cronJobFieldsForDefaulting = `	// Default values for various CronJob fields
	DefaultConcurrencyPolicy          batchv2.ConcurrencyPolicy
	DefaultSuspend                    bool
	DefaultSuccessfulJobsHistoryLimit int32
	DefaultFailedJobsHistoryLimit     int32
`

const cronJobDefaultingLogic = `// Set default values
	d.applyDefaults(cronjob)
	return nil
`

const cronJobDefaultFunction = `
// applyDefaults applies default values to CronJob fields.
func (d *CronJobCustomDefaulter) applyDefaults(cronJob *batchv2.CronJob) {
	if cronJob.Spec.ConcurrencyPolicy == "" {
		cronJob.Spec.ConcurrencyPolicy = d.DefaultConcurrencyPolicy
	}
	if cronJob.Spec.Suspend == nil {
		cronJob.Spec.Suspend = new(bool)
		*cronJob.Spec.Suspend = d.DefaultSuspend
	}
	if cronJob.Spec.SuccessfulJobsHistoryLimit == nil {
		cronJob.Spec.SuccessfulJobsHistoryLimit = new(int32)
		*cronJob.Spec.SuccessfulJobsHistoryLimit = d.DefaultSuccessfulJobsHistoryLimit
	}
	if cronJob.Spec.FailedJobsHistoryLimit == nil {
		cronJob.Spec.FailedJobsHistoryLimit = new(int32)
		*cronJob.Spec.FailedJobsHistoryLimit = d.DefaultFailedJobsHistoryLimit
	}
}
`

const cronJobValidationFunction = `
// validateCronJob validates the fields of a CronJob object.
func validateCronJob(cronjob *batchv2.CronJob) error {
	var allErrs field.ErrorList
	if err := validateCronJobName(cronjob); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateCronJobSpec(cronjob); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(schema.GroupKind{Group: "batch.tutorial.kubebuilder.io", Kind: "CronJob"}, cronjob.Name, allErrs)
}

func validateCronJobName(cronjob *batchv2.CronJob) *field.Error {
	if len(cronjob.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		return field.Invalid(field.NewPath("metadata").Child("name"), cronjob.ObjectMeta.Name, "must be no more than 52 characters")
	}
	return nil
}

// validateCronJobSpec validates the schedule format of the custom CronSchedule type
func validateCronJobSpec(cronjob *batchv2.CronJob) *field.Error {
	// Build cron expression from the parts
	parts := []string{"*", "*", "*", "*", "*"} // default parts for minute, hour, day of month, month, day of week
	if cronjob.Spec.Schedule.Minute != nil {
		parts[0] = string(*cronjob.Spec.Schedule.Minute)  // Directly cast CronField (which is an alias of string) to string
	}
	if cronjob.Spec.Schedule.Hour != nil {
		parts[1] = string(*cronjob.Spec.Schedule.Hour)
	}
	if cronjob.Spec.Schedule.DayOfMonth != nil {
		parts[2] = string(*cronjob.Spec.Schedule.DayOfMonth)
	}
	if cronjob.Spec.Schedule.Month != nil {
		parts[3] = string(*cronjob.Spec.Schedule.Month)
	}
	if cronjob.Spec.Schedule.DayOfWeek != nil {
		parts[4] = string(*cronjob.Spec.Schedule.DayOfWeek)
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

const originalSetupManager = `// SetupCronJobWebhookWithManager registers the webhook for CronJob in the manager.
func SetupCronJobWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&batchv2.CronJob{}).
		WithValidator(&CronJobCustomValidator{}).
		WithDefaulter(&CronJobCustomDefaulter{}).
		Complete()
}`

const replaceSetupManager = `// SetupCronJobWebhookWithManager registers the webhook for CronJob in the manager.
func SetupCronJobWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&batchv2.CronJob{}).
		WithValidator(&CronJobCustomValidator{}).
		WithDefaulter(&CronJobCustomDefaulter{
			DefaultConcurrencyPolicy:          batchv2.AllowConcurrent,
			DefaultSuspend:                    false,
			DefaultSuccessfulJobsHistoryLimit: 3,
			DefaultFailedJobsHistoryLimit:     1,
		}).
		Complete()
}`
