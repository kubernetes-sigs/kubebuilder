/*
Copyright 2026 The Kubernetes Authors.

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

var _ = Describe("Webhooks Set", func() {
	var webhooks *Webhooks

	BeforeEach(func() {
		webhooks = &Webhooks{
			WebhookVersion: "v1",
			Defaulting:     true,
			Validation:     true,
			Conversion:     false,
			Spoke:          []string{"v1beta1"},
			DefaultingPath: "/mutate",
			ValidationPath: "/validate",
		}
	})

	Context("Set", func() {
		It("should completely replace webhooks with new configuration", func() {
			newWebhooks := &Webhooks{
				WebhookVersion: "v1",
				Defaulting:     false,
				Validation:     true,
				Conversion:     true,
				Spoke:          []string{"v2"},
			}

			webhooks.Set(newWebhooks)

			Expect(webhooks.Defaulting).To(BeFalse())
			Expect(webhooks.Validation).To(BeTrue())
			Expect(webhooks.Conversion).To(BeTrue())
			Expect(webhooks.Spoke).To(Equal([]string{"v2"}))
			Expect(webhooks.DefaultingPath).To(BeEmpty())
			Expect(webhooks.ValidationPath).To(BeEmpty())
		})

		It("should clear all fields when nil is provided", func() {
			webhooks.Set(nil)

			Expect(webhooks.WebhookVersion).To(BeEmpty())
			Expect(webhooks.Defaulting).To(BeFalse())
			Expect(webhooks.Validation).To(BeFalse())
			Expect(webhooks.Conversion).To(BeFalse())
			Expect(webhooks.Spoke).To(BeNil())
			Expect(webhooks.DefaultingPath).To(BeEmpty())
			Expect(webhooks.ValidationPath).To(BeEmpty())
		})

		It("should handle empty spoke slices correctly", func() {
			newWebhooks := &Webhooks{
				WebhookVersion: "v1",
				Validation:     true,
			}

			webhooks.Set(newWebhooks)

			Expect(webhooks.Spoke).To(BeNil())
		})

		It("should deep copy spoke versions", func() {
			spoke := []string{"v1beta1", "v1beta2"}
			newWebhooks := &Webhooks{
				WebhookVersion: "v1",
				Defaulting:     true,
				Spoke:          spoke,
			}

			webhooks.Set(newWebhooks)

			// Modify original spoke slice
			spoke[0] = "modified"

			// Webhook's spoke should not be affected
			Expect(webhooks.Spoke[0]).To(Equal("v1beta1"))
		})
	})
})
