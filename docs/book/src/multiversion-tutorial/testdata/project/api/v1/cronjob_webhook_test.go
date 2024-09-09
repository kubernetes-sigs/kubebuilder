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

package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("CronJob Webhook", func() {
	var (
		obj       *CronJob
		oldObj    *CronJob
		validator CronJobCustomValidator
	)

	BeforeEach(func() {
		obj = &CronJob{
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
		})
	})

	Context("When creating or updating CronJob under Validating Webhook", func() {
		It("Should deny creation if the name is too long", func() {
			obj.ObjectMeta.Name = "this-name-is-way-too-long-and-should-fail-validation-because-it-is-way-too-long"
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred(), "Expected name validation to fail for a too-long name")
			Expect(warnings).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("must be no more than 52 characters"))
		})

		It("Should admit creation if the name is valid", func() {
			obj.ObjectMeta.Name = "valid-cronjob-name"
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred(), "Expected name validation to pass for a valid name")
			Expect(warnings).To(BeNil())
		})

		It("Should deny creation if the schedule is invalid", func() {
			obj.Spec.Schedule = "invalid-cron-schedule"
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred(), "Expected spec validation to fail for an invalid schedule")
			Expect(warnings).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("Expected exactly 5 fields, found 1: invalid-cron-schedule"))
		})

		It("Should admit creation if the schedule is valid", func() {
			obj.Spec.Schedule = "*/5 * * * *"
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred(), "Expected spec validation to pass for a valid schedule")
			Expect(warnings).To(BeNil())
		})

		It("Should deny update if both name and spec are invalid", func() {
			oldObj.ObjectMeta.Name = "valid-cronjob-name"
			oldObj.Spec.Schedule = "*/5 * * * *"

			By("simulating an update")
			obj.ObjectMeta.Name = "this-name-is-way-too-long-and-should-fail-validation-because-it-is-way-too-long"
			obj.Spec.Schedule = "invalid-cron-schedule"

			By("validating an update")
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred(), "Expected validation to fail for both name and spec")
			Expect(warnings).To(BeNil())
		})

		It("Should admit update if both name and spec are valid", func() {
			oldObj.ObjectMeta.Name = "valid-cronjob-name"
			oldObj.Spec.Schedule = "*/5 * * * *"

			By("simulating an update")
			obj.ObjectMeta.Name = "valid-cronjob-name-updated"
			obj.Spec.Schedule = "0 0 * * *"

			By("validating an update")
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).NotTo(HaveOccurred(), "Expected validation to pass for a valid update")
			Expect(warnings).To(BeNil())
		})
	})

})
