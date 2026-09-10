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
			Entry("group flag captured another flag", resourceOptions{GVK: resource.GVK{Group: "--version"}}),
			Entry("version flag captured another flag", resourceOptions{GVK: resource.GVK{Version: "--kind"}}),
			Entry("kind flag captured another flag", resourceOptions{GVK: resource.GVK{Kind: "--group"}}),
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

	Context("resolveDomain", func() {
		const externalDomain = "cert-manager.io"

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

		DescribeTable("keeps the project domain",
			func(getCfg func() *cfgv3.Cfg) {
				opts := resourceOptions{GVK: fullGVK}
				Expect(opts.resolveDomain(getCfg())).To(Equal(domain))
			},
			Entry("when nothing is tracked", func() *cfgv3.Cfg { return newCfg() }),
			Entry("when the exact GVK is tracked", func() *cfgv3.Cfg {
				return newCfg(resource.Resource{GVK: fullGVK})
			}),
			Entry("when multiple G+V+K matches are ambiguous", func() *cfgv3.Cfg {
				return newCfg(external(externalDomain), external("other-vendor.io"))
			}),
		)

		It("adopts the stored external domain and flows it through newResource", func() {
			opts := resourceOptions{GVK: fullGVK}
			cfg := newCfg(external(externalDomain))

			opts.Domain = opts.resolveDomain(cfg)
			res := opts.newResource()

			Expect(opts.Domain).To(Equal(externalDomain))
			Expect(res.Domain).To(Equal(externalDomain))
			Expect(res.QualifiedGroup()).To(Equal(group + "." + externalDomain))
		})
	})
})
