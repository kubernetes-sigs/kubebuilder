/*
Copyright 2024 The Kubernetes authors.

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

package v2

import (
	"context"
	"fmt"
	"github.com/robfig/cron"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var cronjoblog = logf.Log.WithName("cronjob-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks.
func (r *CronJob) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&CronJobCustomValidator{}).
		WithDefaulter(&CronJobCustomDefaulter{
			DefaultConcurrencyPolicy:          AllowConcurrent,
			DefaultSuspend:                    false,
			DefaultSuccessfulJobsHistoryLimit: 3,
			DefaultFailedJobsHistoryLimit:     1,
		}).
		Complete()
}

// +kubebuilder:object:generate=false
// CronJobCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind CronJob when those are created or updated.
type CronJobCustomDefaulter struct {
	// Default values for various CronJob fields
	DefaultConcurrencyPolicy          ConcurrencyPolicy
	DefaultSuspend                    bool
	DefaultSuccessfulJobsHistoryLimit int32
	DefaultFailedJobsHistoryLimit     int32
}

var _ webhook.CustomDefaulter = &CronJobCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind CronJob.
func (d *CronJobCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	cronjob, ok := obj.(*CronJob)
	if !ok {
		return fmt.Errorf("expected a CronJob object but got %T", obj)
	}
	cronjoblog.Info("Defaulting for CronJob", "name", cronjob.GetName())

	// Set default values
	cronjob.Default()
	return nil
}

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

// +kubebuilder:object:generate=false
// CronJobCustomValidator struct is responsible for validating the CronJob resource
// when it is created, updated, or deleted.
type CronJobCustomValidator struct{}

var _ webhook.CustomValidator = &CronJobCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type CronJob.
func (v *CronJobCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cronjob, ok := obj.(*CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob object but got %T", obj)
	}
	cronjoblog.Info("Validation for CronJob upon creation", "name", cronjob.GetName())
	return nil, cronjob.validateCronJob()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type CronJob.
func (v *CronJobCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	cronjob, ok := newObj.(*CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob object but got %T", newObj)
	}
	cronjoblog.Info("Validation for CronJob upon update", "name", cronjob.GetName())
	return nil, cronjob.validateCronJob()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type CronJob.
func (v *CronJobCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cronjob, ok := obj.(*CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob object but got %T", obj)
	}
	cronjoblog.Info("Validation for CronJob upon deletion", "name", cronjob.GetName())
	return nil, nil
}

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
		parts[0] = string(*r.Spec.Schedule.Minute) // Directly cast CronField (which is an alias of string) to string
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
