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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GVK", func() {
	const (
		group           = "group"
		domain          = "my.domain"
		version         = "v1"
		kind            = "Kind"
		internalVersion = "__internal"
	)

	gvk := GVK{Group: group, Domain: domain, Version: version, Kind: kind}

	Context("Validate", func() {
		DescribeTable("should pass valid GVKs",
			func(gvk GVK) { Expect(gvk.Validate()).To(Succeed()) },
			Entry("Standard GVK", gvk),
			Entry("Version (__internal)", GVK{Group: group, Domain: domain, Version: internalVersion, Kind: kind}),
		)

		DescribeTable("should fail for invalid GVKs",
			func(gvk GVK) { Expect(gvk.Validate()).NotTo(Succeed()) },
			// Ensure that the rest of the fields are valid to check each part
			Entry("Group (uppercase)", GVK{Group: "Group", Domain: domain, Version: version, Kind: kind}),
			Entry("Group (non-alpha characters)", GVK{Group: "_*?", Domain: domain, Version: version, Kind: kind}),
			Entry("Domain (uppercase)", GVK{Group: group, Domain: "Domain", Version: version, Kind: kind}),
			Entry("Domain (non-alpha characters)", GVK{Group: group, Domain: "_*?", Version: version, Kind: kind}),
			Entry("Group and Domain (empty)", GVK{Group: "", Domain: "", Version: version, Kind: kind}),
			Entry("Version (empty)", GVK{Group: group, Domain: domain, Version: "", Kind: kind}),
			Entry("Version (wrong prefix)", GVK{Group: group, Domain: domain, Version: "-example.com", Kind: kind}),
			Entry("Version (wrong suffix)", GVK{Group: group, Domain: domain, Version: "example.com-", Kind: kind}),
			Entry("Version (uppercase)", GVK{Group: group, Domain: domain, Version: "Example.com", Kind: kind}),
			Entry("Version (special characters)", GVK{Group: group, Domain: domain, Version: "example!domain.com", Kind: kind}),
			Entry("Version (consecutive dots)", GVK{Group: group, Domain: domain, Version: "example..com", Kind: kind}),
			Entry("Kind (empty)", GVK{Group: group, Domain: domain, Version: version, Kind: ""}),
			Entry("Kind (whitespaces)", GVK{Group: group, Domain: domain, Version: version, Kind: "Ki nd"}),
			Entry("Kind (lowercase)", GVK{Group: group, Domain: domain, Version: version, Kind: "kind"}),
			Entry("Kind (starts with number)", GVK{Group: group, Domain: domain, Version: version, Kind: "1Kind"}),
			Entry("Kind (ends with `-`)", GVK{Group: group, Domain: domain, Version: version, Kind: "Kind-"}),
			Entry("Kind (non-alpha characters)", GVK{Group: group, Domain: domain, Version: version, Kind: "_*?"}),
			Entry("Kind (too long)",
				GVK{Group: group, Domain: domain, Version: version, Kind: strings.Repeat("a", 64)}),
		)
	})

	Context("QualifiedGroup", func() {
		DescribeTable("should return the correct string",
			func(gvk GVK, qualifiedGroup string) { Expect(gvk.QualifiedGroup()).To(Equal(qualifiedGroup)) },
			Entry("fully qualified resource", gvk, group+"."+domain),
			Entry("empty group name", GVK{Domain: domain, Version: version, Kind: kind}, domain),
			Entry("empty domain", GVK{Group: group, Version: version, Kind: kind}, group),
		)
	})

	Context("IsEqualTo", func() {
		It("should return true for the same resource", func() {
			Expect(gvk.IsEqualTo(GVK{Group: group, Domain: domain, Version: version, Kind: kind})).To(BeTrue())
		})

		DescribeTable("should return false for different resources",
			func(other GVK) { Expect(gvk.IsEqualTo(other)).To(BeFalse()) },
			Entry("different kind", GVK{Group: group, Domain: domain, Version: version, Kind: "Kind2"}),
			Entry("different version", GVK{Group: group, Domain: domain, Version: "v2", Kind: kind}),
			Entry("different domain", GVK{Group: group, Domain: "other.domain", Version: version, Kind: kind}),
			Entry("different group", GVK{Group: "group2", Domain: domain, Version: version, Kind: kind}),
		)
	})
})
