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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("GVK", func() {
	const (
		group   = "group"
		domain  = "my.domain"
		version = "v1"
		kind    = "Kind"
	)

	Context("QualifiedGroup", func() {
		DescribeTable("should return the correct string",
			func(gvk GVK, qualifiedGroup string) { Expect(gvk.QualifiedGroup()).To(Equal(qualifiedGroup)) },
			Entry("fully qualified resource", GVK{Group: group, Domain: domain, Version: version, Kind: kind},
				group+"."+domain),
			Entry("empty group name", GVK{Domain: domain, Version: version, Kind: kind}, domain),
			Entry("empty domain", GVK{Group: group, Version: version, Kind: kind}, group),
		)
	})

	Context("IsEqualTo", func() {
		var gvk = GVK{Group: group, Domain: domain, Version: version, Kind: kind}

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
