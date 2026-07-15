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

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ = Describe("resourceOptions", func() {
	const (
		group   = "crew"
		domain  = "test.io"
		version = "v1"
		kind    = "FirstMate"
	)

	var (
		fullGVK     resource.GVK
		noDomainGVK resource.GVK
		noGroupGVK  resource.GVK
	)

	BeforeEach(func() {
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
	})

	Context("validate", func() {
		DescribeTable("should succeed for valid options",
			func(options resourceOptions) { Expect(options.validate()).To(Succeed()) },
			Entry("full GVK", resourceOptions{GVK: fullGVK}),
			Entry("missing domain", resourceOptions{GVK: noDomainGVK}),
			Entry("missing group", resourceOptions{GVK: noGroupGVK}),
		)

		DescribeTable("should fail for invalid options",
			func(options resourceOptions) { Expect(options.validate()).NotTo(Succeed()) },
			Entry("group flag captured another flag", resourceOptions{GVK: resource.GVK{Group: versionFlagArg}}),
			Entry("version flag captured another flag", resourceOptions{GVK: resource.GVK{Version: kindFlagArg}}),
			Entry("kind flag captured another flag", resourceOptions{GVK: resource.GVK{Kind: groupFlagArg}}),
			Entry("domain flag captured another flag", resourceOptions{GVK: resource.GVK{Domain: kindFlagArg}}),
		)
	})

	Context("newResource", func() {
		DescribeTable("should succeed if the Resource is valid",
			func(getOpts func() resourceOptions) {
				options := getOpts()

				Expect(options.validate()).To(Succeed())

				resource := options.newResource()
				Expect(resource.Validate()).To(Succeed())
				Expect(resource.GVK.IsEqualTo(options.GVK)).To(BeTrue())
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
			Entry("full GVK", func() resourceOptions { return resourceOptions{GVK: fullGVK} }),
			Entry("missing domain", func() resourceOptions { return resourceOptions{GVK: noDomainGVK} }),
			Entry("missing group", func() resourceOptions { return resourceOptions{GVK: noGroupGVK} }),
		)

		DescribeTable("should default the Plural by pluralizing the Kind",
			func(kind, plural string) {
				options := resourceOptions{GVK: resource.GVK{Group: group, Version: version, Kind: kind}}
				Expect(options.validate()).To(Succeed())

				resource := options.newResource()
				Expect(resource.Validate()).To(Succeed())
				Expect(resource.GVK.IsEqualTo(options.GVK)).To(BeTrue())
				Expect(resource.Plural).To(Equal(plural))
			},
			Entry("for `FirstMate`", "FirstMate", "firstmates"),
			Entry("for `Fish`", "Fish", "fish"),
			Entry("for `Helmswoman`", "Helmswoman", "helmswomen"),
		)
	})

	Context("domain resolution", func() {
		const externalDomain = "cert-manager.io"
		const otherDomain = "other-domain.io"

		external := func(d string) resource.Resource {
			return resource.Resource{
				GVK:      resource.GVK{Group: group, Domain: d, Version: version, Kind: kind},
				External: true,
			}
		}

		newCfg := func(tracked ...resource.Resource) *cfgv3.Cfg {
			c := &cfgv3.Cfg{Version: cfgv3.Version, Domain: domain}
			for _, r := range tracked {
				Expect(c.AddResource(r)).To(Succeed())
			}
			return c
		}

		Context("trackedDomains", func() {
			It("collects the domain of every group, version and kind match in project file order", func() {
				opts := resourceOptions{GVK: noDomainGVK}
				cfg := newCfg(external(externalDomain), external(otherDomain))

				Expect(opts.trackedDomains(cfg)).To(Equal([]string{externalDomain, otherDomain}))
			})

			It("ignores resources with a different group, version or kind", func() {
				opts := resourceOptions{GVK: noDomainGVK}
				cfg := newCfg(
					external(externalDomain),
					resource.Resource{GVK: resource.GVK{
						Group: group, Domain: otherDomain, Version: "v2", Kind: kind,
					}},
					resource.Resource{GVK: resource.GVK{
						Group: "other", Domain: otherDomain, Version: version, Kind: kind,
					}},
				)

				Expect(opts.trackedDomains(cfg)).To(Equal([]string{externalDomain}))
			})

			It("returns nothing when no resource matches", func() {
				opts := resourceOptions{GVK: noDomainGVK}

				Expect(opts.trackedDomains(newCfg())).To(BeEmpty())
			})
		})

		Context("resolveDomain", func() {
			It("falls back to the project domain when nothing is tracked", func() {
				opts := resourceOptions{GVK: noDomainGVK}

				Expect(opts.resolveDomain(nil, domain)).To(Equal(domain))
			})

			It("adopts the single tracked domain and flows it through newResource", func() {
				opts := resourceOptions{GVK: noDomainGVK}
				cfg := newCfg(external(externalDomain))

				opts.Domain = opts.resolveDomain(opts.trackedDomains(cfg), domain)
				res := opts.newResource()

				Expect(opts.Domain).To(Equal(externalDomain))
				Expect(res.Domain).To(Equal(externalDomain))
				Expect(res.QualifiedGroup()).To(Equal(group + "." + externalDomain))
			})

			It("resolves to nothing when several resources match, whatever their order", func() {
				opts := resourceOptions{GVK: noDomainGVK}
				forward := newCfg(external(externalDomain), external(otherDomain))
				reversed := newCfg(external(otherDomain), external(externalDomain))

				Expect(opts.resolveDomain(opts.trackedDomains(forward), domain)).To(BeEmpty())
				Expect(opts.resolveDomain(opts.trackedDomains(reversed), domain)).To(BeEmpty())
			})

			It("keeps an explicit --domain even when several resources match", func() {
				opts := resourceOptions{GVK: resource.GVK{
					Group: group, Version: version, Kind: kind, Domain: externalDomain,
				}}
				cfg := newCfg(external(externalDomain), external(otherDomain))

				Expect(opts.resolveDomain(opts.trackedDomains(cfg), domain)).To(Equal(externalDomain))
			})

			It("keeps an explicit --domain even when the exact GVK is already tracked", func() {
				opts := resourceOptions{GVK: fullGVK}
				cfg := newCfg(resource.Resource{GVK: fullGVK})

				Expect(opts.resolveDomain(opts.trackedDomains(cfg), domain)).To(Equal(domain))
			})
		})

		Context("checkDomain", func() {
			It("accepts anything when nothing is tracked", func() {
				opts := resourceOptions{GVK: noDomainGVK}

				Expect(opts.checkDomain(nil, "")).To(Succeed())
				Expect(opts.checkDomain(nil, externalDomain)).To(Succeed())
			})

			It("accepts a domain matching one of the tracked resources", func() {
				opts := resourceOptions{GVK: noDomainGVK}
				tracked := []string{externalDomain, otherDomain}

				Expect(opts.checkDomain(tracked, otherDomain)).To(Succeed())
			})

			It("names the matching qualified groups when the ambiguity went unresolved", func() {
				opts := resourceOptions{GVK: noDomainGVK}

				err := opts.checkDomain([]string{externalDomain, otherDomain}, "")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("match more than one resource"))
				Expect(err.Error()).To(ContainSubstring(group + "." + externalDomain))
				Expect(err.Error()).To(ContainSubstring(group + "." + otherDomain))
				Expect(err.Error()).To(ContainSubstring("--domain"))
			})

			It("allows recording a new resource beside the tracked ones", func() {
				opts := resourceOptions{GVK: noDomainGVK}

				Expect(opts.checkDomain([]string{externalDomain}, "brand-new.io")).To(Succeed())
			})
		})
	})
})
