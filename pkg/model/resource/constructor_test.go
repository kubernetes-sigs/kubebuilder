/*
Copyright 2021 The Kubernetes Authors.

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
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("New", func() {
	var (
		group   = "group"
		domain  = "my.domain"
		version = "v1"
		kind    = "Kind"
		plural  = "types"
		rPath   = "sigs.kubebuilder.io/kubebuilder/test/apis"
		repo    = "sigs.kubebuilder.io/kubebuilder/test"

		gvk = GVK{
			Group:   group,
			Domain:  domain,
			Version: version,
			Kind:    kind,
		}
	)

	Context("without any options", func() {
		It("should success", func() {
			resource, err := New(gvk)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.GVK.IsEqualTo(gvk)).To(BeTrue())

			Expect(resource.Plural).To(Equal(RegularPlural(gvk.Kind)))
		})
	})

	Context("WithPlural", func() {
		It("should success", func() {
			resource, err := New(gvk, WithPlural(plural))
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.GVK.IsEqualTo(gvk)).To(BeTrue())

			Expect(resource.Plural).To(Equal(plural))
		})
	})

	Context("WithPath", func() {
		It("should success", func() {
			resource, err := New(gvk, WithPath(rPath))
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.GVK.IsEqualTo(gvk)).To(BeTrue())

			Expect(resource.Path).To(Equal(rPath))
		})
	})

	Context("WithLocalPath", func() {
		DescribeTable("should success",
			func(multiGroup bool) {
				resource, err := New(gvk, WithLocalPath(repo, multiGroup))
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.GVK.IsEqualTo(gvk)).To(BeTrue())

				Expect(resource.Path).To(Equal(APIPackagePath(repo, gvk.Group, gvk.Version, multiGroup)))
			},
			Entry("for single-group configuration", false),
			Entry("for multi-group configuration", true),
		)
	})

	Context("WithBuiltInPath", func() {
		It("should success", func() {
			resource, err := New(gvk, WithBuiltInPath())
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.GVK.IsEqualTo(gvk)).To(BeTrue())

			Expect(resource.Path).To(Equal(path.Join("k8s.io", "api", resource.Group, resource.Version)))
		})
	})

	Context("ScaffoldAPI", func() {
		It("should success", func() {
			resource, err := New(gvk, ScaffoldAPI(version))
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.GVK.IsEqualTo(gvk)).To(BeTrue())

			Expect(resource.HasAPI()).To(BeTrue())
			Expect(resource.API.CRDVersion).To(Equal(version))
		})
	})

	Context("WithScope", func() {
		DescribeTable("should success",
			func(namespaced bool) {
				resource, err := New(gvk, WithScope(namespaced))
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.GVK.IsEqualTo(gvk)).To(BeTrue())

				Expect(resource.API.Namespaced).To(Equal(namespaced))
			},
			Entry("for cluster-scoped resources", false),
			Entry("for namespaced resources", true),
		)
	})

	Context("ScaffoldController", func() {
		It("should success", func() {
			resource, err := New(gvk, ScaffoldController())
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.GVK.IsEqualTo(gvk)).To(BeTrue())

			Expect(resource.Controller).To(BeTrue())
		})
	})

	Context("Scaffold*Webhook", func() {
		DescribeTable("should success",
			func(defaulting, validation, conversion bool) {
				options := make([]Option, 0, 3)
				if defaulting {
					options = append(options, ScaffoldDefaultingWebhook(version))
				}
				if validation {
					options = append(options, ScaffoldValidationWebhook(version))
				}
				if conversion {
					options = append(options, ScaffoldConversionWebhook(version))
				}

				resource, err := New(gvk, options...)
				Expect(err).NotTo(HaveOccurred())
				Expect(resource.GVK.IsEqualTo(gvk)).To(BeTrue())

				if defaulting || validation || conversion {
					Expect(resource.Webhooks.WebhookVersion).To(Equal(version))
				}
				Expect(resource.HasDefaultingWebhook()).To(Equal(defaulting))
				Expect(resource.HasValidationWebhook()).To(Equal(validation))
				Expect(resource.HasConversionWebhook()).To(Equal(conversion))
			},
			Entry("for no webhook", false, false, false),
			Entry("for defaulting webhook", true, false, false),
			Entry("for validation webhook", false, true, false),
			Entry("for conversion webhook", false, false, true),
			Entry("for defaulting and validation webhooks", true, true, false),
			Entry("for defaulting and conversion webhooks", true, false, true),
			Entry("for validation and conversion webhooks", false, true, true),
			Entry("for defaulting and validation and conversion webhooks", true, true, true),
		)

		DescribeTable("should fail",
			func(options ...Option) {
				_, err := New(gvk, options...)
				Expect(err).To(HaveOccurred())
			},
			Entry("for wrong defaulting version", ScaffoldConversionWebhook(version), ScaffoldDefaultingWebhook("v2")),
			Entry("for wrong validation version", ScaffoldDefaultingWebhook(version), ScaffoldValidationWebhook("v2")),
			Entry("for wrong conversion version", ScaffoldValidationWebhook(version), ScaffoldConversionWebhook("v2")),
		)
	})
})
