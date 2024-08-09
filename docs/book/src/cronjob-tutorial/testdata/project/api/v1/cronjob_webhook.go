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
// +kubebuilder:docs-gen:collapse=Apache License

package v1

import (
	"context"
	"fmt"
	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:docs-gen:collapse=Go imports

/*
Next, we'll setup a logger for the webhooks.
*/

var cronjoblog = logf.Log.WithName("cronjob-resource")

/*
Then, we set up the webhook with the manager.
*/

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *CronJob) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&CronJobCustomValidator{}).
		WithDefaulter(&CronJobCustomDefaulter{}).
		Complete()
}

/*
Notice that we use kubebuilder markers to generate webhook manifests.
This marker is responsible for generating a mutating webhook manifest.

The meaning of each marker can be found [here](/reference/markers/webhook.md).
*/

// +kubebuilder:webhook:path=/mutate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=true,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=mcronjob.kb.io,sideEffects=None,admissionReviewVersions=v1

/*
We use the `webhook.Defaulter` interface to set defaults to our CRD.
A webhook will automatically be served that calls this defaulting.

The `Default` method is expected to mutate the receiver, setting the defaults.
*/

type CronJobCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &CronJobCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (d *CronJobCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	cronjoblog.Info("CustomDefaulter for Admiral")
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "CronJob" {
		return fmt.Errorf("expected Kind Admiral got %q", req.Kind.Kind)
	}
	castedObj, ok := obj.(*CronJob)
	if !ok {
		return fmt.Errorf("expected an CronJob object but got %T", obj)
	}
	cronjoblog.Info("default", "name", castedObj.GetName())

	if castedObj.Spec.ConcurrencyPolicy == "" {
		castedObj.Spec.ConcurrencyPolicy = AllowConcurrent
	}
	if castedObj.Spec.Suspend == nil {
		castedObj.Spec.Suspend = new(bool)
	}
	if castedObj.Spec.SuccessfulJobsHistoryLimit == nil {
		castedObj.Spec.SuccessfulJobsHistoryLimit = new(int32)
		*castedObj.Spec.SuccessfulJobsHistoryLimit = 3
	}
	if castedObj.Spec.FailedJobsHistoryLimit == nil {
		castedObj.Spec.FailedJobsHistoryLimit = new(int32)
		*castedObj.Spec.FailedJobsHistoryLimit = 1
	}

	return nil
}

/*
This marker is responsible for generating a validating webhook manifest.
*/

// +kubebuilder:webhook:verbs=create;update;delete,path=/validate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=false,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,versions=v1,name=vcronjob.kb.io,sideEffects=None,admissionReviewVersions=v1

/*
We can validate our CRD beyond what's possible with declarative
validation. Generally, declarative validation should be sufficient, but
sometimes more advanced use cases call for complex validation.

For instance, we'll see below that we use this to validate a well-formed cron
schedule without making up a long regular expression.

If `webhook.Validator` interface is implemented, a webhook will automatically be
served that calls the validation.

The `ValidateCreate`, `ValidateUpdate` and `ValidateDelete` methods are expected
to validate its receiver upon creation, update and deletion respectively.
We separate out ValidateCreate from ValidateUpdate to allow behavior like making
certain fields immutable, so that they can only be set on creation.
ValidateDelete is also separated from ValidateUpdate to allow different
validation behavior on deletion.
Here, however, we just use the same shared validation for `ValidateCreate` and
`ValidateUpdate`. And we do nothing in `ValidateDelete`, since we don't need to
validate anything on deletion.
*/

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.

type CronJobCustomValidator struct{}

var _ webhook.CustomValidator = &CronJobCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *CronJobCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cronjoblog.Info("Creation Validation for CronJob")

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "CronJob" {
		return nil, fmt.Errorf("expected Kind CronJob got %q", req.Kind.Kind)
	}
	castedObj, ok := obj.(*CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob object but got %T", obj)
	}
	cronjoblog.Info("default", "name", castedObj.GetName())

	return nil, v.validateCronJob(castedObj)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *CronJobCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	cronjoblog.Info("Update Validation for CronJob")
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "CronJob" {
		return nil, fmt.Errorf("expected Kind CronJob got %q", req.Kind.Kind)
	}
	castedObj, ok := newObj.(*CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob object but got %T", newObj)
	}
	cronjoblog.Info("default", "name", castedObj.GetName())

	return nil, v.validateCronJob(castedObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (v *CronJobCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cronjoblog.Info("Deletion Validation for CronJob")
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "CronJob" {
		return nil, fmt.Errorf("expected Kind CronJob got %q", req.Kind.Kind)
	}
	castedObj, ok := obj.(*CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob object but got %T", obj)
	}
	cronjoblog.Info("default", "name", castedObj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

/*
We validate the name and the spec of the CronJob.
*/

func (v *CronJobCustomValidator) validateCronJob(castedObj *CronJob) error {
	var allErrs field.ErrorList
	if err := v.validateCronJobName(castedObj); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := v.validateCronJobSpec(castedObj); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "batch.tutorial.kubebuilder.io", Kind: "CronJob"},
		castedObj.Name, allErrs)
}

/*
Some fields are declaratively validated by OpenAPI schema.
You can find kubebuilder validation markers (prefixed
with `// +kubebuilder:validation`) in the
[Designing an API](api-design.md) section.
You can find all of the kubebuilder supported markers for
declaring validation by running `controller-gen crd -w`,
or [here](/reference/markers/crd-validation.md).
*/

func (v *CronJobCustomValidator) validateCronJobSpec(castedObj *CronJob) *field.Error {
	// The field helpers from the kubernetes API machinery help us return nicely
	// structured validation errors.
	return validateScheduleFormat(
		castedObj.Spec.Schedule,
		field.NewPath("spec").Child("schedule"))
}

/*
We'll need to validate the [cron](https://en.wikipedia.org/wiki/Cron) schedule
is well-formatted.
*/

func validateScheduleFormat(schedule string, fldPath *field.Path) *field.Error {
	if _, err := cron.ParseStandard(schedule); err != nil {
		return field.Invalid(fldPath, schedule, err.Error())
	}
	return nil
}

/*
Validating the length of a string field can be done declaratively by
the validation schema.

But the `ObjectMeta.Name` field is defined in a shared package under
the apimachinery repo, so we can't declaratively validate it using
the validation schema.
*/

func (v *CronJobCustomValidator) validateCronJobName(castedObj *CronJob) *field.Error {
	if len(castedObj.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		// The job name length is 63 characters like all Kubernetes objects
		// (which must fit in a DNS subdomain). The cronjob controller appends
		// a 11-character suffix to the cronjob (`-$TIMESTAMP`) when creating
		// a job. The job name length limit is 63 characters. Therefore cronjob
		// names must have length <= 63-11=52. If we don't validate this here,
		// then job creation will fail later.
		return field.Invalid(field.NewPath("metadata").Child("name"), castedObj.Name, "must be no more than 52 characters")
	}
	return nil
}

// +kubebuilder:docs-gen:collapse=Validate object name
