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

package cli

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

var _ = Describe("ResourceOptions", func() {
	const (
		group   = "crew"
		domain  = "test.io"
		version = "v1"
		kind    = "FirstMate"
	)

	var (
		fullGVK = resource.GVK{
			Group:   group,
			Domain:  domain,
			Version: version,
			Kind:    kind,
		}
		noDomainGVK = resource.GVK{
			Group:   group,
			Version: version,
			Kind:    kind,
		}
		noGroupGVK = resource.GVK{
			Domain:  domain,
			Version: version,
			Kind:    kind,
		}
	)

	Context("Validate", func() {
		DescribeTable("should succeed for valid options",
			func(options ResourceOptions) { Expect(options.Validate()).To(Succeed()) },
			Entry("full GVK", ResourceOptions{GVK: fullGVK}),
			Entry("missing domain", ResourceOptions{GVK: noDomainGVK}),
			Entry("missing group", ResourceOptions{GVK: noGroupGVK}),
		)

		DescribeTable("should fail for invalid options",
			func(options ResourceOptions) { Expect(options.Validate()).NotTo(Succeed()) },
			Entry("group flag captured another flag", ResourceOptions{GVK: resource.GVK{Group: "--version"}}),
			Entry("version flag captured another flag", ResourceOptions{GVK: resource.GVK{Version: "--kind"}}),
			Entry("kind flag captured another flag", ResourceOptions{GVK: resource.GVK{Kind: "--group"}}),
		)
	})

	Context("NewResource", func() {
		DescribeTable("should succeed if the Resource is valid",
			func(options ResourceOptions) {
				Expect(options.Validate()).To(Succeed())

				resource := options.NewResource()
				Expect(resource.Validate()).To(Succeed())
				Expect(resource.GVK.IsEqualTo(options.GVK)).To(BeTrue())
				// Plural is checked in the next test
				Expect(resource.Path).To(Equal(""))
				Expect(resource.API).NotTo(BeNil())
				Expect(resource.API.CRDVersion).To(Equal(""))
				Expect(resource.API.Namespaced).To(BeFalse())
				Expect(resource.Controller).To(BeFalse())
				Expect(resource.Webhooks).NotTo(BeNil())
				Expect(resource.Webhooks.WebhookVersion).To(Equal(""))
				Expect(resource.Webhooks.Defaulting).To(BeFalse())
				Expect(resource.Webhooks.Validation).To(BeFalse())
				Expect(resource.Webhooks.Conversion).To(BeFalse())
			},
			Entry("full GVK", ResourceOptions{GVK: fullGVK}),
			Entry("missing domain", ResourceOptions{GVK: noDomainGVK}),
			Entry("missing group", ResourceOptions{GVK: noGroupGVK}),
		)

		DescribeTable("should default the Plural by pluralizing the Kind",
			func(kind, plural string) {
				options := ResourceOptions{GVK: resource.GVK{Group: group, Version: version, Kind: kind}}
				Expect(options.Validate()).To(Succeed())

				resource := options.NewResource()
				Expect(resource.Validate()).To(Succeed())
				Expect(resource.GVK.IsEqualTo(options.GVK)).To(BeTrue())
				Expect(resource.Plural).To(Equal(plural))
			},
			Entry("for `FirstMate`", "FirstMate", "firstmates"),
			Entry("for `Fish`", "Fish", "fish"),
			Entry("for `Helmswoman`", "Helmswoman", "helmswomen"),
		)
	})
})
