/*
Copyright 2025 The Kubernetes Authors.

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

var _ = Describe("Webhooks Copy and AddSpoke", func() {
	Context("Copy", func() {
		It("should create a deep copy of Webhooks", func() {
			original := Webhooks{
				WebhookVersion: v1,
				Defaulting:     true,
				Validation:     true,
				Conversion:     false,
				Spoke:          []string{"v1", "v2"},
			}

			copied := original.Copy()

			Expect(copied.WebhookVersion).To(Equal(original.WebhookVersion))
			Expect(copied.Defaulting).To(Equal(original.Defaulting))
			Expect(copied.Validation).To(Equal(original.Validation))
			Expect(copied.Conversion).To(Equal(original.Conversion))
			Expect(copied.Spoke).To(Equal(original.Spoke))
		})

		It("should not affect original when modifying the copy", func() {
			original := Webhooks{
				WebhookVersion: v1,
				Defaulting:     true,
				Spoke:          []string{"v1"},
			}

			copied := original.Copy()
			copied.Defaulting = false
			copied.Spoke = append(copied.Spoke, "v2")

			Expect(original.Defaulting).To(BeTrue())
			Expect(original.Spoke).To(Equal([]string{"v1"}))
			Expect(copied.Defaulting).To(BeFalse())
			Expect(copied.Spoke).To(Equal([]string{"v1", "v2"}))
		})

		It("should handle empty Spoke slice", func() {
			original := Webhooks{
				WebhookVersion: v1,
				Spoke:          []string{},
			}

			copied := original.Copy()
			Expect(copied.Spoke).To(BeNil())
		})

		It("should handle nil Spoke slice", func() {
			original := Webhooks{
				WebhookVersion: v1,
				Spoke:          nil,
			}

			copied := original.Copy()
			Expect(copied.Spoke).To(BeNil())
		})

		It("should create independent Spoke slices", func() {
			original := Webhooks{
				Spoke: []string{"v1"},
			}

			copied := original.Copy()
			copied.Spoke[0] = "v2"

			Expect(original.Spoke[0]).To(Equal("v1"))
			Expect(copied.Spoke[0]).To(Equal("v2"))
		})
	})

	Context("AddSpoke", func() {
		It("should add a new spoke version", func() {
			webhook := &Webhooks{}
			webhook.AddSpoke("v1")

			Expect(webhook.Spoke).To(HaveLen(1))
			Expect(webhook.Spoke).To(ContainElement("v1"))
		})

		It("should not add duplicate spoke versions", func() {
			webhook := &Webhooks{
				Spoke: []string{"v1"},
			}
			webhook.AddSpoke("v1")

			Expect(webhook.Spoke).To(HaveLen(1))
			Expect(webhook.Spoke).To(Equal([]string{"v1"}))
		})

		It("should add multiple different spoke versions", func() {
			webhook := &Webhooks{}
			webhook.AddSpoke("v1")
			webhook.AddSpoke("v2")
			webhook.AddSpoke("v3")

			Expect(webhook.Spoke).To(HaveLen(3))
			Expect(webhook.Spoke).To(ContainElements("v1", "v2", "v3"))
		})

		It("should handle adding existing version in the middle", func() {
			webhook := &Webhooks{
				Spoke: []string{"v1", "v2", "v3"},
			}
			webhook.AddSpoke("v2")

			Expect(webhook.Spoke).To(HaveLen(3))
			Expect(webhook.Spoke).To(Equal([]string{"v1", "v2", "v3"}))
		})
	})

	Context("Validate with duplicate Spoke versions", func() {
		It("should fail validation with duplicate spoke versions", func() {
			webhook := Webhooks{
				WebhookVersion: v1,
				Spoke:          []string{"v1", "v1"},
			}

			Expect(webhook.Validate()).NotTo(Succeed())
		})

		It("should succeed validation with unique spoke versions", func() {
			webhook := Webhooks{
				WebhookVersion: v1,
				Spoke:          []string{"v1", "v2", "v3"},
			}

			Expect(webhook.Validate()).To(Succeed())
		})
	})

	Context("IsEmpty with Spoke", func() {
		It("should return false when only Spoke is set", func() {
			webhook := Webhooks{
				Spoke: []string{"v1"},
			}

			Expect(webhook.IsEmpty()).To(BeFalse())
		})

		It("should return true when Spoke is empty array", func() {
			webhook := Webhooks{
				Spoke: []string{},
			}

			Expect(webhook.IsEmpty()).To(BeTrue())
		})
	})
})
