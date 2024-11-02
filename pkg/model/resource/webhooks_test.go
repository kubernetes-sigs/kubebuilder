/*
Copyright 2022 The Kubernetes Authors.

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

package resource

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//nolint:dupl
var _ = Describe("Webhooks", func() {
	Context("Validate", func() {
		It("should succeed for a valid Webhooks", func() {
			Expect(Webhooks{WebhookVersion: v1}.Validate()).To(Succeed())
		})

		DescribeTable("should fail for invalid Webhooks",
			func(webhooks Webhooks) { Expect(webhooks.Validate()).NotTo(Succeed()) },
			// Ensure that the rest of the fields are valid to check each part
			Entry("empty webhook version", Webhooks{}),
			Entry("invalid webhook version", Webhooks{WebhookVersion: "1"}),
		)
	})

	Context("Update", func() {
		var webhook, other Webhooks

		It("should do nothing if provided a nil pointer", func() {
			webhook = Webhooks{}
			Expect(webhook.Update(nil)).To(Succeed())
			Expect(webhook.WebhookVersion).To(Equal(""))
			Expect(webhook.Defaulting).To(BeFalse())
			Expect(webhook.Validation).To(BeFalse())
			Expect(webhook.Conversion).To(BeFalse())

			webhook = Webhooks{
				WebhookVersion: v1,
				Defaulting:     true,
				Validation:     true,
				Conversion:     true,
				Spoke:          []string{"v2"},
			}
			Expect(webhook.Update(nil)).To(Succeed())
			Expect(webhook.WebhookVersion).To(Equal(v1))
			Expect(webhook.Defaulting).To(BeTrue())
			Expect(webhook.Validation).To(BeTrue())
			Expect(webhook.Conversion).To(BeTrue())
			Expect(webhook.Spoke).To(Equal([]string{"v2"}))
		})

		It("should merge Spoke values without duplicates", func() {
			webhook = Webhooks{
				Spoke: []string{"v1"},
			}
			other = Webhooks{
				Spoke: []string{"v1", "v2"},
			}
			Expect(webhook.Update(&other)).To(Succeed())
			Expect(webhook.Spoke).To(ConsistOf("v1", "v2")) // Ensure no duplicates
		})

		Context("webhooks version", func() {
			It("should modify the webhooks version if provided and not previously set", func() {
				webhook = Webhooks{}
				other = Webhooks{WebhookVersion: v1}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.WebhookVersion).To(Equal(v1))
			})

			It("should keep the webhooks version if not provided", func() {
				webhook = Webhooks{WebhookVersion: v1}
				other = Webhooks{}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.WebhookVersion).To(Equal(v1))
			})

			It("should keep the webhooks version if provided the same as previously set", func() {
				webhook = Webhooks{WebhookVersion: v1}
				other = Webhooks{WebhookVersion: v1}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.WebhookVersion).To(Equal(v1))
			})

			It("should fail if previously set and provided webhooks versions do not match", func() {
				webhook = Webhooks{WebhookVersion: v1}
				other = Webhooks{WebhookVersion: "v1beta1"}
				Expect(webhook.Update(&other)).NotTo(Succeed())
			})
		})

		Context("Defaulting", func() {
			It("should set the defaulting webhook if provided and not previously set", func() {
				webhook = Webhooks{}
				other = Webhooks{Defaulting: true}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Defaulting).To(BeTrue())
			})

			It("should keep the defaulting webhook if previously set", func() {
				webhook = Webhooks{Defaulting: true}

				By("not providing it")
				other = Webhooks{}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Defaulting).To(BeTrue())

				By("providing it")
				other = Webhooks{Defaulting: true}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Defaulting).To(BeTrue())
			})

			It("should not set the defaulting webhook if not provided and not previously set", func() {
				webhook = Webhooks{}
				other = Webhooks{}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Defaulting).To(BeFalse())
			})
		})

		Context("Validation", func() {
			It("should set the validation webhook if provided and not previously set", func() {
				webhook = Webhooks{}
				other = Webhooks{Validation: true}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Validation).To(BeTrue())
			})

			It("should keep the validation webhook if previously set", func() {
				webhook = Webhooks{Validation: true}

				By("not providing it")
				other = Webhooks{}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Validation).To(BeTrue())

				By("providing it")
				other = Webhooks{Validation: true}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Validation).To(BeTrue())
			})

			It("should not set the validation webhook if not provided and not previously set", func() {
				webhook = Webhooks{}
				other = Webhooks{}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Validation).To(BeFalse())
			})
		})

		Context("Conversion", func() {
			It("should set the conversion webhook if provided and not previously set", func() {
				webhook = Webhooks{}
				other = Webhooks{Conversion: true}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Conversion).To(BeTrue())
			})

			It("should keep the conversion webhook if previously set", func() {
				webhook = Webhooks{Conversion: true}

				By("not providing it")
				other = Webhooks{}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Conversion).To(BeTrue())

				By("providing it")
				other = Webhooks{Conversion: true}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Conversion).To(BeTrue())
			})

			It("should not set the conversion webhook if not provided and not previously set", func() {
				webhook = Webhooks{}
				other = Webhooks{}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.Conversion).To(BeFalse())
			})
		})
	})

	Context("IsEmpty", func() {
		var (
			none       = Webhooks{}
			defaulting = Webhooks{
				WebhookVersion: "v1",
				Defaulting:     true,
				Validation:     false,
				Conversion:     false,
			}
			validation = Webhooks{
				WebhookVersion: "v1",
				Defaulting:     false,
				Validation:     true,
				Conversion:     false,
			}
			conversion = Webhooks{
				WebhookVersion: "v1",
				Defaulting:     false,
				Validation:     false,
				Conversion:     true,
			}
			defaultingAndValidation = Webhooks{
				WebhookVersion: "v1",
				Defaulting:     true,
				Validation:     true,
				Conversion:     false,
			}
			defaultingAndConversion = Webhooks{
				WebhookVersion: "v1",
				Defaulting:     true,
				Validation:     false,
				Conversion:     true,
			}
			validationAndConversion = Webhooks{
				WebhookVersion: "v1",
				Defaulting:     false,
				Validation:     true,
				Conversion:     true,
			}
			all = Webhooks{
				WebhookVersion: "v1",
				Defaulting:     true,
				Validation:     true,
				Conversion:     true,
			}
		)

		It("should return true fo an empty object", func() {
			Expect(none.IsEmpty()).To(BeTrue())
		})

		DescribeTable("should return false for non-empty objects",
			func(webhooks Webhooks) { Expect(webhooks.IsEmpty()).To(BeFalse()) },
			Entry("defaulting", defaulting),
			Entry("validation", validation),
			Entry("conversion", conversion),
			Entry("defaulting and validation", defaultingAndValidation),
			Entry("defaulting and conversion", defaultingAndConversion),
			Entry("validation and conversion", validationAndConversion),
			Entry("defaulting and validation and conversion", all),
		)
	})
})
