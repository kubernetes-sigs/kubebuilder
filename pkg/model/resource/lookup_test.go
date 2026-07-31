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
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	issuerGroup     = "cert-manager"
	issuerVersion   = "v1"
	issuerKind      = "Issuer"
	issuerPlural    = "issuers"
	ownDomain       = "example.com"
	thirdDomain     = "third.io"
	coreDomain      = "k8s.io"
	certManagerPath = "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	otherVendorPath = "github.com/other-vendor/cert-manager/apis/certmanager/v1"
	thirdVendorPath = "github.com/third-vendor/cert-manager/apis/certmanager/v1"
	coreAPIPath     = "k8s.io/api/cert-manager/v1"
)

// issuerGVK is what the commands know: the Group, the Version and the Kind, with the domain of
// the project because they cannot know the one of an API defined outside it.
var issuerGVK = GVK{Group: issuerGroup, Domain: ownDomain, Version: issuerVersion, Kind: issuerKind}

func issuer(domain string) Resource {
	return Resource{
		GVK:      GVK{Group: issuerGroup, Domain: domain, Version: issuerVersion, Kind: issuerKind},
		Plural:   issuerPlural,
		API:      &API{},
		Webhooks: &Webhooks{},
	}
}

func externalIssuer(domain, apiPath string) Resource {
	res := issuer(domain)
	res.External = true
	res.Path = apiPath

	return res
}

func coreIssuer(domain string) Resource {
	res := issuer(domain)
	res.Core = true
	res.Path = coreAPIPath

	return res
}

// permutations returns every order in which the resources can be listed in the PROJECT file.
func permutations(resources []Resource) [][]Resource {
	if len(resources) <= 1 {
		return [][]Resource{resources}
	}

	var orders [][]Resource
	for i := range resources {
		rest := make([]Resource, 0, len(resources)-1)
		rest = append(rest, resources[:i]...)
		rest = append(rest, resources[i+1:]...)
		for _, order := range permutations(rest) {
			orders = append(orders, append([]Resource{resources[i]}, order...))
		}
	}

	return orders
}

// outcome describes the whole result, so that the tests compare every order of the PROJECT file.
func outcome(selected *Resource, err error) string {
	switch {
	case err != nil:
		return "error: " + err.Error()
	case selected == nil:
		return "not tracked"
	default:
		return fmt.Sprintf("domain=%q path=%q external=%t core=%t",
			selected.Domain, selected.Path, selected.External, selected.Core)
	}
}

// selectAll runs Select for every order of the resources and returns the outcome, failing when
// the order changes it.
func selectAll(resources []Resource, domain, apiPath string) string {
	orders := permutations(resources)
	results := make([]string, 0, len(orders))
	for _, order := range orders {
		results = append(results, outcome(Select(order, issuerGVK, domain, apiPath)))
	}

	for _, result := range results[1:] {
		ExpectWithOffset(1, result).To(Equal(results[0]), "the order of the resources changed the result")
	}

	return results[0]
}

func external(domain, apiPath string) string {
	return fmt.Sprintf("domain=%q path=%q external=true core=false", domain, apiPath)
}

var _ = Describe("Select", func() {
	var certManager, otherVendor, thirdVendor Resource

	BeforeEach(func() {
		certManager = externalIssuer("io", certManagerPath)
		otherVendor = externalIssuer(coreDomain, otherVendorPath)
		thirdVendor = externalIssuer(thirdDomain, thirdVendorPath)
	})

	Context("when no resource has the same Group, Version and Kind", func() {
		It("should select none", func() {
			other := externalIssuer("io", certManagerPath)
			other.Kind = "Certificate"

			Expect(selectAll([]Resource{other}, "", "")).To(Equal("not tracked"))
		})

		It("should select none whatever the flags", func() {
			Expect(selectAll(nil, thirdDomain, thirdVendorPath)).To(Equal("not tracked"))
		})
	})

	Context("when one resource has the same Group, Version and Kind", func() {
		DescribeTable("should select the same resource whatever the flags",
			func(domain, apiPath, expected string) {
				Expect(selectAll([]Resource{certManager}, domain, apiPath)).To(ContainSubstring(expected))
			},
			Entry("without flags", "", "", external("io", certManagerPath)),
			Entry("with the tracked domain", "io", "", external("io", certManagerPath)),
			Entry("with the tracked path", "", certManagerPath, external("io", certManagerPath)),
			Entry("with the tracked domain and path", "io", certManagerPath, external("io", certManagerPath)),
			Entry("with another domain and a path", thirdDomain, thirdVendorPath, "not tracked"),
			Entry("with the tracked domain and another path", "io", thirdVendorPath,
				`error: resource cert-manager/v1, Kind Issuer has the path`),
			Entry("with another domain", thirdDomain, "",
				`error: no resource matches cert-manager/v1, Kind Issuer with the domain "third.io"`),
			Entry("with another path", "", thirdVendorPath,
				`error: resource cert-manager/v1, Kind Issuer has the path`),
		)

		It("should select a resource that has no path yet", func() {
			noPath := issuer("io")
			noPath.External = true

			Expect(selectAll([]Resource{noPath}, "", certManagerPath)).To(Equal(external("io", "")))
		})
	})

	Context("when several resources have the same Group, Version and Kind", func() {
		DescribeTable("should select the same resource whatever the flags",
			func(domain, apiPath, expected string) {
				Expect(selectAll([]Resource{certManager, otherVendor}, domain, apiPath)).To(ContainSubstring(expected))
			},
			Entry("with the domain of the first one", "io", "", external("io", certManagerPath)),
			Entry("with the domain of the second one", coreDomain, "", external(coreDomain, otherVendorPath)),
			Entry("with the path of the first one", "", certManagerPath, external("io", certManagerPath)),
			Entry("with the path of the second one", "", otherVendorPath, external(coreDomain, otherVendorPath)),
			Entry("with an untracked domain and a path", thirdDomain, thirdVendorPath, "not tracked"),
			Entry("with a domain and a path of different resources", "io", otherVendorPath,
				"error: resource cert-manager/v1, Kind Issuer has the path"),
			Entry("without flags", "", "", "error: 2 resources match cert-manager/v1, Kind Issuer"),
			Entry("with an untracked path", "", thirdVendorPath,
				"error: 2 resources match cert-manager/v1, Kind Issuer"),
			Entry("with an untracked domain", thirdDomain, "",
				`error: no resource matches cert-manager/v1, Kind Issuer with the domain "third.io"`),
		)

		It("should carry the matches in the error, for the caller to list them", func() {
			_, err := Select([]Resource{certManager, otherVendor, thirdVendor}, issuerGVK, "", "")

			var ambiguous AmbiguousSelectionError
			Expect(errors.As(err, &ambiguous)).To(BeTrue())
			Expect(ambiguous.Matches).To(HaveLen(3))
			Expect(ambiguous.GVK.Kind).To(Equal(issuerKind))
		})

		It("should select the same resource among three of them", func() {
			tracked := []Resource{certManager, otherVendor, thirdVendor}

			Expect(selectAll(tracked, "io", "")).To(Equal(external("io", certManagerPath)))
			Expect(selectAll(tracked, coreDomain, "")).To(Equal(external(coreDomain, otherVendorPath)))
			Expect(selectAll(tracked, thirdDomain, "")).To(Equal(external(thirdDomain, thirdVendorPath)))
		})

		It("should not select by path when the same path is tracked twice", func() {
			tracked := []Resource{certManager, externalIssuer(coreDomain, certManagerPath)}

			Expect(selectAll(tracked, "", certManagerPath)).To(ContainSubstring("2 resources match"))
		})

		It("should tell a core type from an external one", func() {
			core := coreIssuer(coreDomain)
			tracked := []Resource{core, certManager}

			Expect(selectAll(tracked, coreDomain, "")).To(Equal(
				fmt.Sprintf(`domain=%q path=%q external=false core=true`, coreDomain, coreAPIPath)))
			Expect(selectAll(tracked, "io", "")).To(Equal(external("io", certManagerPath)))
		})
	})

	Context("when the project defines one of the resources itself", func() {
		var own Resource

		BeforeEach(func() {
			own = issuer(ownDomain)
			own.Path = "github.com/example/test/api/v1"
		})

		It("should select it when no domain and no path are given", func() {
			tracked := []Resource{own, certManager, otherVendor}

			Expect(selectAll(tracked, "", "")).To(Equal(
				`domain="example.com" path="github.com/example/test/api/v1" external=false core=false`))
		})

		It("should still select an API defined outside the project by its domain", func() {
			tracked := []Resource{own, certManager}

			Expect(selectAll(tracked, "io", "")).To(Equal(external("io", certManagerPath)))
		})

		It("should not choose when every resource is defined outside the project", func() {
			core := coreIssuer("")
			tracked := []Resource{core, certManager}

			Expect(selectAll(tracked, "", "")).To(ContainSubstring("2 resources match"))
		})
	})
})

var _ = Describe("AdoptTracked", func() {
	It("should copy the values that identify the tracked resource", func() {
		tracked := externalIssuer("io", certManagerPath)
		tracked.Module = "github.com/cert-manager/cert-manager@v1.20.2"
		tracked.Plural = "issuerlist"

		res := issuer(ownDomain)
		res.AdoptTracked(tracked)

		Expect(res.Domain).To(Equal("io"))
		Expect(res.Path).To(Equal(certManagerPath))
		Expect(res.Plural).To(Equal("issuerlist"))
		Expect(res.External).To(BeTrue())
		Expect(res.Core).To(BeFalse())
		Expect(res.Module).To(Equal("github.com/cert-manager/cert-manager@v1.20.2"))
	})

	It("should copy the values of a core type", func() {
		res := issuer(ownDomain)
		res.AdoptTracked(coreIssuer(coreDomain))

		Expect(res.Core).To(BeTrue())
		Expect(res.External).To(BeFalse())
		Expect(res.Domain).To(Equal(coreDomain))
	})
})

var _ = Describe("IsDefinedInProject", func() {
	DescribeTable("should tell the APIs of the project from the ones defined outside it",
		func(res Resource, expected bool) {
			Expect(res.IsDefinedInProject()).To(Equal(expected))
		},
		Entry("an API of the project", issuer(ownDomain), true),
		Entry("an external API", externalIssuer("io", certManagerPath), false),
		Entry("a core type", coreIssuer(coreDomain), false),
	)
})
