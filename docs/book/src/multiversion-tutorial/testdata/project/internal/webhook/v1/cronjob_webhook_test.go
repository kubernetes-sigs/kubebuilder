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

package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("CronJob Webhook", func() {
	var (
		obj       *batchv1.CronJob
		oldObj    *batchv1.CronJob
		validator CronJobCustomValidator
		defaulter CronJobCustomDefaulter
	)

	BeforeEach(func() {
		obj = &batchv1.CronJob{
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
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating CronJob under Defaulting Webhook", func() {
		It("Should apply defaults when a required field is empty", func() {
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
		})
	})

	Context("When creating or updating CronJob under Validating Webhook", func() {
		It("Should deny creation if the name is too long", func() {
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
		})
	})

	Context("When creating CronJob under Conversion Webhook", func() {
		// TODO (user): Add logic to convert the object to the desired version and verify the conversion
		// Example:
		// It("Should convert the object correctly", func() {
		//     convertedObj := &batchv1.CronJob{}
		//     Expect(obj.ConvertTo(convertedObj)).To(Succeed())
		//     Expect(convertedObj).ToNot(BeNil())
		// })
	})

})
