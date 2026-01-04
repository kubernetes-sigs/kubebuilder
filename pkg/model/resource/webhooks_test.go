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

		It("should succeed for valid Webhooks with unique spoke versions", func() {
			Expect(Webhooks{WebhookVersion: v1, Spoke: []string{"v1", "v2", "v3"}}.Validate()).To(Succeed())
		})

		DescribeTable("should fail for invalid Webhooks",
			func(webhooks Webhooks) { Expect(webhooks.Validate()).NotTo(Succeed()) },
			// Ensure that the rest of the fields are valid to check each part
			Entry("empty webhook version", Webhooks{}),
			Entry("invalid webhook version", Webhooks{WebhookVersion: "1"}),
			Entry("duplicate spoke versions", Webhooks{WebhookVersion: v1, Spoke: []string{"v1", "v2", "v1"}}),
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

		Context("Custom webhook paths", func() {
			It("should set the defaulting path if provided and not previously set", func() {
				webhook = Webhooks{}
				other = Webhooks{DefaultingPath: "/custom-defaulting"}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.DefaultingPath).To(Equal("/custom-defaulting"))
			})

			It("should update the defaulting path if other provides a new one", func() {
				webhook = Webhooks{DefaultingPath: "/old-path"}
				other = Webhooks{DefaultingPath: "/new-path"}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.DefaultingPath).To(Equal("/new-path"))
			})

			It("should set the validation path if provided and not previously set", func() {
				webhook = Webhooks{}
				other = Webhooks{ValidationPath: "/custom-validation"}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.ValidationPath).To(Equal("/custom-validation"))
			})

			It("should update the validation path if other provides a new one", func() {
				webhook = Webhooks{ValidationPath: "/old-path"}
				other = Webhooks{ValidationPath: "/new-path"}
				Expect(webhook.Update(&other)).To(Succeed())
				Expect(webhook.ValidationPath).To(Equal("/new-path"))
			})
		})
	})

	Context("IsEmpty", func() {
		var (
			none       Webhooks
			defaulting Webhooks
			validation Webhooks
			conversion Webhooks

			defaultingAndValidation Webhooks
			defaultingAndConversion Webhooks
			validationAndConversion Webhooks

			all Webhooks
		)

		BeforeEach(func() {
			none = Webhooks{}
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
		})

		It("should return true fo an empty object", func() {
			Expect(none.IsEmpty()).To(BeTrue())
		})

		DescribeTable("should return false for non-empty objects",
			func(get func() Webhooks) {
				Expect(get().IsEmpty()).To(BeFalse())
			},
			Entry("defaulting", func() Webhooks { return defaulting }),
			Entry("validation", func() Webhooks { return validation }),
			Entry("conversion", func() Webhooks { return conversion }),
			Entry("defaulting and validation", func() Webhooks { return defaultingAndValidation }),
			Entry("defaulting and conversion", func() Webhooks { return defaultingAndConversion }),
			Entry("validation and conversion", func() Webhooks { return validationAndConversion }),
			Entry("defaulting and validation and conversion", func() Webhooks { return all }),
		)
	})

	Context("AddSpoke", func() {
		It("should add a spoke version if not already present", func() {
			webhook := Webhooks{}
			webhook.AddSpoke("v1")
			Expect(webhook.Spoke).To(Equal([]string{"v1"}))

			webhook.AddSpoke("v2")
			Expect(webhook.Spoke).To(ConsistOf("v1", "v2"))
		})

		It("should not add a duplicate spoke version", func() {
			webhook := Webhooks{Spoke: []string{"v1"}}
			webhook.AddSpoke("v1")
			Expect(webhook.Spoke).To(Equal([]string{"v1"}))
		})
	})

	Context("Copy", func() {
		It("should return an exact copy", func() {
			webhook := Webhooks{
				WebhookVersion: v1,
				Defaulting:     true,
				Validation:     true,
				Conversion:     true,
				Spoke:          []string{"v1", "v2"},
				DefaultingPath: "/custom-defaulting",
				ValidationPath: "/custom-validation",
			}
			other := webhook.Copy()

			Expect(other.WebhookVersion).To(Equal(webhook.WebhookVersion))
			Expect(other.Defaulting).To(Equal(webhook.Defaulting))
			Expect(other.Validation).To(Equal(webhook.Validation))
			Expect(other.Conversion).To(Equal(webhook.Conversion))
			Expect(other.Spoke).To(Equal(webhook.Spoke))
			Expect(other.DefaultingPath).To(Equal(webhook.DefaultingPath))
			Expect(other.ValidationPath).To(Equal(webhook.ValidationPath))
		})

		It("modifying the copy should not affect the original", func() {
			webhook := Webhooks{
				WebhookVersion: v1,
				Defaulting:     true,
				Validation:     true,
				Conversion:     true,
				Spoke:          []string{"v1", "v2"},
				DefaultingPath: "/custom-defaulting",
				ValidationPath: "/custom-validation",
			}
			other := webhook.Copy()

			// Modify the copy
			other.WebhookVersion = "v1beta1"
			other.Defaulting = false
			other.Validation = false
			other.Conversion = false
			other.Spoke[0] = "v3"
			other.Spoke = append(other.Spoke, "v4")
			other.DefaultingPath = "/new-defaulting"
			other.ValidationPath = "/new-validation"

			// Original should remain unchanged
			Expect(webhook.WebhookVersion).To(Equal(v1))
			Expect(webhook.Defaulting).To(BeTrue())
			Expect(webhook.Validation).To(BeTrue())
			Expect(webhook.Conversion).To(BeTrue())
			Expect(webhook.Spoke).To(Equal([]string{"v1", "v2"}))
			Expect(webhook.DefaultingPath).To(Equal("/custom-defaulting"))
			Expect(webhook.ValidationPath).To(Equal("/custom-validation"))
		})

		It("should handle nil Spoke slice", func() {
			webhook := Webhooks{
				WebhookVersion: v1,
				Spoke:          nil,
			}
			other := webhook.Copy()

			Expect(other.Spoke).To(BeNil())
		})
	})
})
