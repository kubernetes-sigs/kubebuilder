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

package cronjob

const webhookIntro = `batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

// +kubebuilder:docs-gen:collapse=Go imports

/*
Next, we'll setup a logger for the webhooks.
*/

`

const webhookDefaultingSettings = `// Set default values
	d.applyDefaults(cronjob)
	return nil
}

// applyDefaults applies default values to CronJob fields.
func (d *CronJobCustomDefaulter) applyDefaults(cronJob *batchv1.CronJob) {
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

const webhooksNoticeMarker = `
/*
Notice that we use kubebuilder markers to generate webhook manifests.
This marker is responsible for generating a mutating webhook manifest.

The meaning of each marker can be found [here](/reference/markers/webhook.md).
*/

/*
This marker is responsible for generating a mutation webhook manifest.
*/
`

const explanationValidateCRD = `
/*
We can validate our CRD beyond what's possible with declarative
validation. Generally, declarative validation should be sufficient, but
sometimes more advanced use cases call for complex validation.

For instance, we'll see below that we use this to validate a well-formed cron
schedule without making up a long regular expression.

If` + " `" + `webhook.CustomValidator` + "`" + ` interface is implemented, a webhook will automatically be
served that calls the validation.

The` + " `" + `ValidateCreate` + "`" + `, ` + "`" + `ValidateUpdate` + "`" + ` and` + " `" + `ValidateDelete` + "`" + ` methods are expected
to validate its receiver upon creation, update and deletion respectively.
We separate out ValidateCreate from ValidateUpdate to allow behavior like making
certain fields immutable, so that they can only be set on creation.
ValidateDelete is also separated from ValidateUpdate to allow different
validation behavior on deletion.
Here, however, we just use the same shared validation for` + " `" + `ValidateCreate` + "`" + ` and
` + "`" + `ValidateUpdate` + "`" + `. And we do nothing in` + " `" + `ValidateDelete` + "`" + `, since we don't need to
validate anything on deletion.
*/

/*
This marker is responsible for generating a validation webhook manifest.
*/`

const customInterfaceDefaultInfo = `/*
We use the ` + "`" + `webhook.CustomDefaulter` + "`" + `interface to set defaults to our CRD.
A webhook will automatically be served that calls this defaulting.

The ` + "`" + `Default` + "`" + `method is expected to mutate the receiver, setting the defaults.
*/

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind CronJob.`

const webhookValidateSpecMethods = `
/*
We validate the name and the spec of the CronJob.
*/

// validateCronJob validates the fields of a CronJob object.
func validateCronJob(cronjob *batchv1.CronJob) error {
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

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "batch.tutorial.kubebuilder.io", Kind: "CronJob"},
		cronjob.Name, allErrs)
}

/*
Some fields are declaratively validated by OpenAPI schema.
You can find kubebuilder validation markers (prefixed
with ` + "`" + `// +kubebuilder:validation` + "`" + `) in the
[Designing an API](api-design.md) section.
You can find all of the kubebuilder supported markers for
declaring validation by running ` + "`" + `controller-gen crd -w` + "`" + `,
or [here](/reference/markers/crd-validation.md).
*/

func validateCronJobSpec(cronjob *batchv1.CronJob) *field.Error {
	// The field helpers from the kubernetes API machinery help us return nicely
	// structured validation errors.
	return validateScheduleFormat(
		cronjob.Spec.Schedule,
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

But the ` + "`" + `ObjectMeta.Name` + "`" + ` field is defined in a shared package under
the apimachinery repo, so we can't declaratively validate it using
the validation schema.
*/

func validateCronJobName(cronjob *batchv1.CronJob) *field.Error {
	if len(cronjob.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		// The job name length is 63 characters like all Kubernetes objects
		// (which must fit in a DNS subdomain). The cronjob controller appends
		// a 11-character suffix to the cronjob (` + "`" + `-$TIMESTAMP` + "`" + `) when creating
		// a job. The job name length limit is 63 characters. Therefore cronjob
		// names must have length <= 63-11=52. If we don't validate this here,
		// then job creation will fail later.
		return field.Invalid(field.NewPath("metadata").Child("name"), cronjob.ObjectMeta.Name, "must be no more than 52 characters")
	}
	return nil
}

// +kubebuilder:docs-gen:collapse=Validate object name`

const fragmentForDefaultFields = `
	// Default values for various CronJob fields
	DefaultConcurrencyPolicy      batchv1.ConcurrencyPolicy
	DefaultSuspend                bool
	DefaultSuccessfulJobsHistoryLimit int32
	DefaultFailedJobsHistoryLimit int32
`

const webhookTestCreateDefaultingFragment = `// TODO (user): Add logic for defaulting webhooks
		// Example:
		// It("Should apply defaults when a required field is empty", func() {
		//     By("simulating a scenario where defaults should be applied")
		//     obj.SomeFieldWithDefault = ""
		//     By("calling the Default method to apply defaults")
		//     defaulter.Default(ctx, obj)
		//     By("checking that the default values are set")
		//     Expect(obj.SomeFieldWithDefault).To(Equal("default_value"))
		// })`

const webhookTestCreateDefaultingReplaceFragment = `It("Should apply defaults when a required field is empty", func() {
			By("simulating a scenario where defaults should be applied")
			obj.Spec.ConcurrencyPolicy = ""           // This should default to AllowConcurrent
			obj.Spec.Suspend = nil                    // This should default to false
			obj.Spec.SuccessfulJobsHistoryLimit = nil // This should default to 3
			obj.Spec.FailedJobsHistoryLimit = nil     // This should default to 1

			By("calling the Default method to apply defaults")
			defaulter.Default(ctx, obj)

			By("checking that the default values are set")
			Expect(obj.Spec.ConcurrencyPolicy).To(Equal(batchv1.AllowConcurrent), "Expected ConcurrencyPolicy to default to AllowConcurrent")
			Expect(*obj.Spec.Suspend).To(BeFalse(), "Expected Suspend to default to false")
			Expect(*obj.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(3)), "Expected SuccessfulJobsHistoryLimit to default to 3")
			Expect(*obj.Spec.FailedJobsHistoryLimit).To(Equal(int32(1)), "Expected FailedJobsHistoryLimit to default to 1")
		})

		It("Should not overwrite fields that are already set", func() {
			By("setting fields that would normally get a default")
			obj.Spec.ConcurrencyPolicy = batchv1.ForbidConcurrent
			obj.Spec.Suspend = new(bool)
			*obj.Spec.Suspend = true
			obj.Spec.SuccessfulJobsHistoryLimit = new(int32)
			*obj.Spec.SuccessfulJobsHistoryLimit = 5
			obj.Spec.FailedJobsHistoryLimit = new(int32)
			*obj.Spec.FailedJobsHistoryLimit = 2

			By("calling the Default method to apply defaults")
			defaulter.Default(ctx, obj)
			
			By("checking that the fields were not overwritten")
			Expect(obj.Spec.ConcurrencyPolicy).To(Equal(batchv1.ForbidConcurrent), "Expected ConcurrencyPolicy to retain its set value")
			Expect(*obj.Spec.Suspend).To(BeTrue(), "Expected Suspend to retain its set value")
			Expect(*obj.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(5)), "Expected SuccessfulJobsHistoryLimit to retain its set value")
			Expect(*obj.Spec.FailedJobsHistoryLimit).To(Equal(int32(2)), "Expected FailedJobsHistoryLimit to retain its set value")
		})`

const webhookTestingValidatingTodoFragment = `// TODO (user): Add logic for validating webhooks
		// Example:
		// It("Should deny creation if a required field is missing", func() {
		//     By("simulating an invalid creation scenario")
		//     obj.SomeRequiredField = ""
		//     Expect(validator.ValidateCreate(ctx, obj)).Error().To(HaveOccurred())
		// })
		//
		// It("Should admit creation if all required fields are present", func() {
		//     By("simulating an invalid creation scenario")
		//     obj.SomeRequiredField = "valid_value"
		//     Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
		// })
		//
		// It("Should validate updates correctly", func() {
		//     By("simulating a valid update scenario")
		//     oldObj.SomeRequiredField = "updated_value"
		//     obj.SomeRequiredField = "updated_value"
		//     Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
		// })`

const webhookTestingValidatingExampleFragment = `It("Should deny creation if the name is too long", func() {
			obj.ObjectMeta.Name = "this-name-is-way-too-long-and-should-fail-validation-because-it-is-way-too-long"
			Expect(validator.ValidateCreate(ctx, obj)).Error().To(
				MatchError(ContainSubstring("must be no more than 52 characters")),
				"Expected name validation to fail for a too-long name")
		})

		It("Should admit creation if the name is valid", func() {
			obj.ObjectMeta.Name = "valid-cronjob-name"
			Expect(validator.ValidateCreate(ctx, obj)).To(BeNil(),
				"Expected name validation to pass for a valid name")
		})

		It("Should deny creation if the schedule is invalid", func() {
			obj.Spec.Schedule = "invalid-cron-schedule"
			Expect(validator.ValidateCreate(ctx, obj)).Error().To(
				MatchError(ContainSubstring("Expected exactly 5 fields, found 1: invalid-cron-schedule")),
				"Expected spec validation to fail for an invalid schedule")
		})

		It("Should admit creation if the schedule is valid", func() {
			obj.Spec.Schedule = "*/5 * * * *"
			Expect(validator.ValidateCreate(ctx, obj)).To(BeNil(),
				"Expected spec validation to pass for a valid schedule")
		})

		It("Should deny update if both name and spec are invalid", func() {
			oldObj.ObjectMeta.Name = "valid-cronjob-name"
			oldObj.Spec.Schedule = "*/5 * * * *"

			By("simulating an update")
			obj.ObjectMeta.Name = "this-name-is-way-too-long-and-should-fail-validation-because-it-is-way-too-long"
			obj.Spec.Schedule = "invalid-cron-schedule"

			By("validating an update")
			Expect(validator.ValidateUpdate(ctx, oldObj, obj)).Error().To(HaveOccurred(),
				"Expected validation to fail for both name and spec")
		})

		It("Should admit update if both name and spec are valid", func() {
			oldObj.ObjectMeta.Name = "valid-cronjob-name"
			oldObj.Spec.Schedule = "*/5 * * * *"

			By("simulating an update")
			obj.ObjectMeta.Name = "valid-cronjob-name-updated"
			obj.Spec.Schedule = "0 0 * * *"

			By("validating an update")
			Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil(),
				"Expected validation to pass for a valid update")
		})`

const webhookTestsBeforeEachOriginal = `obj = &batchv1.CronJob{}
		oldObj = &batchv1.CronJob{}
		validator = CronJobCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = CronJobCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		// TODO (user): Add any setup logic common to all tests`

const webhookTestsBeforeEachChanged = `obj = &batchv1.CronJob{
			Spec: batchv1.CronJobSpec{
				Schedule:                   "*/5 * * * *",
				ConcurrencyPolicy:          batchv1.AllowConcurrent,
				SuccessfulJobsHistoryLimit: new(int32),
				FailedJobsHistoryLimit:     new(int32),
			},
		}
		*obj.Spec.SuccessfulJobsHistoryLimit = 3
		*obj.Spec.FailedJobsHistoryLimit = 1

		oldObj = &batchv1.CronJob{
			Spec: batchv1.CronJobSpec{
				Schedule:                   "*/5 * * * *",
				ConcurrencyPolicy:          batchv1.AllowConcurrent,
				SuccessfulJobsHistoryLimit: new(int32),
				FailedJobsHistoryLimit:     new(int32),
			},
		}
		*oldObj.Spec.SuccessfulJobsHistoryLimit = 3
		*oldObj.Spec.FailedJobsHistoryLimit = 1

		validator = CronJobCustomValidator{}
		defaulter = CronJobCustomDefaulter{
			DefaultConcurrencyPolicy:          batchv1.AllowConcurrent,
			DefaultSuspend:                    false,
			DefaultSuccessfulJobsHistoryLimit: 3,
			DefaultFailedJobsHistoryLimit:     1,
		}

		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")`
