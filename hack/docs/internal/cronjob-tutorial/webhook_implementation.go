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

const WebhookIntro = `"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:docs-gen:collapse=Go imports

/*
Next, we'll setup a logger for the webhooks.
*/

`

const WebhookMarker = `/*
Notice that we use kubebuilder markers to generate webhook manifests.
This marker is responsible for generating a mutating webhook manifest.

The meaning of each marker can be found [here](/reference/markers/webhook.md).
*/

// +kubebuilder:webhook:path=/mutate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=true,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=mcronjob.kb.io,sideEffects=None,admissionReviewVersions=v1

/*
We use the` + " `" + `webhook.CustomDefaulter` + "`" + ` interface to set defaults to our CRD.
A webhook will automatically be served that calls this defaulting.

The` + " `" + `Default` + "`" + ` method is expected to mutate the receiver, setting the defaults.
*/
`

const WebhookDefaultingSettings = `

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
`
const WebhookValidateSpec = `
/*
We validate the name and the spec of the CronJob.
*/

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

/*
Some fields are declaratively validated by OpenAPI schema.
You can find kubebuilder validation markers (prefixed
with ` + "`" + `// +kubebuilder:validation` + "`" + `) in the
[Designing an API](api-design.md) section.
You can find all of the kubebuilder supported markers for
declaring validation by running ` + "`" + `controller-gen crd -w` + "`" + `,
or [here](/reference/markers/crd-validation.md).
*/

func (r *CronJob) validateCronJobSpec() *field.Error {
	// The field helpers from the kubernetes API machinery help us return nicely
	// structured validation errors.
	return validateScheduleFormat(
		r.Spec.Schedule,
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

func (r *CronJob) validateCronJobName() *field.Error {
	if len(r.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		// The job name length is 63 characters like all Kubernetes objects
		// (which must fit in a DNS subdomain). The cronjob controller appends
		// a 11-character suffix to the cronjob (` + "`" + `-$TIMESTAMP` + "`" + `) when creating
		// a job. The job name length limit is 63 characters. Therefore cronjob
		// names must have length <= 63-11=52. If we don't validate this here,
		// then job creation will fail later.
		return field.Invalid(field.NewPath("metadata").Child("name"), r.ObjectMeta.Name, "must be no more than 52 characters")
	}
	return nil
}

// +kubebuilder:docs-gen:collapse=Validate object name`

const fragmentForDefaultFields = `
	// Default values for various CronJob fields
	DefaultConcurrencyPolicy      ConcurrencyPolicy
	DefaultSuspend                bool
	DefaultSuccessfulJobsHistoryLimit int32
	DefaultFailedJobsHistoryLimit int32
`

const webhookTestCreateDefaultingFragment = `// TODO (user): Add logic for defaulting webhooks
		// Example:
		// It("Should apply defaults when a required field is empty", func() {
		//     By("simulating a scenario where defaults should be applied")
		//     obj.SomeFieldWithDefault = ""
		//     Expect(obj.Default(ctx)).To(Succeed())
		//     Expect(obj.SomeFieldWithDefault).To(Equal("default_value"))
		// })`

const webhookTestCreateDefaultingReplaceFragment = `It("Should apply defaults when a required field is empty", func() {
			By("simulating a scenario where defaults should be applied")
			obj.Spec.ConcurrencyPolicy = ""           // This should default to AllowConcurrent
			obj.Spec.Suspend = nil                    // This should default to false
			obj.Spec.SuccessfulJobsHistoryLimit = nil // This should default to 3
			obj.Spec.FailedJobsHistoryLimit = nil     // This should default to 1

			By("calling the Default method to apply defaults")
			obj.Default()

			By("checking that the default values are set")
			Expect(obj.Spec.ConcurrencyPolicy).To(Equal(AllowConcurrent), "Expected ConcurrencyPolicy to default to AllowConcurrent")
			Expect(*obj.Spec.Suspend).To(BeFalse(), "Expected Suspend to default to false")
			Expect(*obj.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(3)), "Expected SuccessfulJobsHistoryLimit to default to 3")
			Expect(*obj.Spec.FailedJobsHistoryLimit).To(Equal(int32(1)), "Expected FailedJobsHistoryLimit to default to 1")
		})

		It("Should not overwrite fields that are already set", func() {
			By("setting fields that would normally get a default")
			obj.Spec.ConcurrencyPolicy = ForbidConcurrent
			obj.Spec.Suspend = new(bool)
			*obj.Spec.Suspend = true
			obj.Spec.SuccessfulJobsHistoryLimit = new(int32)
			*obj.Spec.SuccessfulJobsHistoryLimit = 5
			obj.Spec.FailedJobsHistoryLimit = new(int32)
			*obj.Spec.FailedJobsHistoryLimit = 2

			By("calling the Default method to apply defaults")
			obj.Default()
			
			By("checking that the fields were not overwritten")
			Expect(obj.Spec.ConcurrencyPolicy).To(Equal(ForbidConcurrent), "Expected ConcurrencyPolicy to retain its set value")
			Expect(*obj.Spec.Suspend).To(BeTrue(), "Expected Suspend to retain its set value")
			Expect(*obj.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(5)), "Expected SuccessfulJobsHistoryLimit to retain its set value")
			Expect(*obj.Spec.FailedJobsHistoryLimit).To(Equal(int32(2)), "Expected FailedJobsHistoryLimit to retain its set value")
		})`

const webhookTestingValidatingTodoFragment = `// TODO (user): Add logic for validating webhooks
		// Example:
		// It("Should deny creation if a required field is missing", func() {
		//     By("simulating an invalid creation scenario")
		//     obj.SomeRequiredField = ""
		//     Expect(obj.ValidateCreate(ctx)).Error().To(HaveOccurred())
		// })
		//
		// It("Should admit creation if all required fields are present", func() {
		//     By("simulating an invalid creation scenario")
		//     obj.SomeRequiredField = "valid_value"
		//     Expect(obj.ValidateCreate(ctx)).To(BeNil())
		// })
		//
		// It("Should validate updates correctly", func() {
		//     By("simulating a valid update scenario")
		//     oldObj := &Captain{SomeRequiredField: "valid_value"}
		//     obj.SomeRequiredField = "updated_value"
		//     Expect(obj.ValidateUpdate(ctx, oldObj)).To(BeNil())
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

const webhookTestsBeforeEachOriginal = `obj = &CronJob{}
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")

		// TODO (user): Add any setup logic common to all tests`

const webhookTestsBeforeEachChanged = `obj = &CronJob{
			Spec: CronJobSpec{
				Schedule:                   "*/5 * * * *",
				ConcurrencyPolicy:          AllowConcurrent,
				SuccessfulJobsHistoryLimit: new(int32),
				FailedJobsHistoryLimit:     new(int32),
			},
		}
		*obj.Spec.SuccessfulJobsHistoryLimit = 3
		*obj.Spec.FailedJobsHistoryLimit = 1

		oldObj = &CronJob{
			Spec: CronJobSpec{
				Schedule:                   "*/5 * * * *",
				ConcurrencyPolicy:          AllowConcurrent,
				SuccessfulJobsHistoryLimit: new(int32),
				FailedJobsHistoryLimit:     new(int32),
			},
		}
		*oldObj.Spec.SuccessfulJobsHistoryLimit = 3
		*oldObj.Spec.FailedJobsHistoryLimit = 1

		validator = CronJobCustomValidator{}

		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")`
