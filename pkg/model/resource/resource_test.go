/*
Copyright 2020 The Kubernetes Authors.

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

var _ = Describe("Resource", func() {
	const (
		group   = "group"
		domain  = "test.io"
		version = "v1"
		kind    = "Kind"
		plural  = "kinds"
		v1beta1 = "v1beta1"
	)

	var (
		gvk = GVK{
			Group:   group,
			Domain:  domain,
			Version: version,
			Kind:    kind,
		}
		res = Resource{
			GVK:    gvk,
			Plural: plural,
		}
	)

	Context("Validate", func() {
		It("should succeed for a valid Resource", func() {
			Expect(res.Validate()).To(Succeed())
		})

		DescribeTable("should fail for invalid Resources",
			func(res Resource) { Expect(res.Validate()).NotTo(Succeed()) },
			// Ensure that the rest of the fields are valid to check each part
			Entry("invalid GVK", Resource{GVK: GVK{}, Plural: "plural"}),
			Entry("invalid Plural", Resource{GVK: gvk, Plural: "Plural"}),
			Entry("invalid API", Resource{GVK: gvk, Plural: "plural", API: &API{CRDVersion: "1"}}),
			Entry("invalid Webhooks", Resource{GVK: gvk, Plural: "plural", Webhooks: &Webhooks{WebhookVersion: "1"}}),
		)
	})

	Context("compound field", func() {
		const (
			safeDomain    = "testio"
			groupVersion  = group + version
			domainVersion = safeDomain + version
			safeGroup     = "mygroup"
			safeAlias     = safeGroup + version
		)

		var (
			resNoGroup = Resource{
				GVK: GVK{
					// Empty group
					Domain:  domain,
					Version: version,
					Kind:    kind,
				},
			}
			resNoDomain = Resource{
				GVK: GVK{
					Group: group,
					// Empty domain
					Version: version,
					Kind:    kind,
				},
			}
			resHyphenGroup = Resource{
				GVK: GVK{
					Group:   "my-group",
					Domain:  domain,
					Version: version,
					Kind:    kind,
				},
			}
			resDotGroup = Resource{
				GVK: GVK{
					Group:   "my.group",
					Domain:  domain,
					Version: version,
					Kind:    kind,
				},
			}
		)

		DescribeTable("PackageName should return the correct string",
			func(res Resource, packageName string) { Expect(res.PackageName()).To(Equal(packageName)) },
			Entry("fully qualified resource", res, group),
			Entry("empty group name", resNoGroup, safeDomain),
			Entry("empty domain", resNoDomain, group),
			Entry("hyphen-containing group", resHyphenGroup, safeGroup),
			Entry("dot-containing group", resDotGroup, safeGroup),
		)

		DescribeTable("ImportAlias",
			func(res Resource, importAlias string) { Expect(res.ImportAlias()).To(Equal(importAlias)) },
			Entry("fully qualified resource", res, groupVersion),
			Entry("empty group name", resNoGroup, domainVersion),
			Entry("empty domain", resNoDomain, groupVersion),
			Entry("hyphen-containing group", resHyphenGroup, safeAlias),
			Entry("dot-containing group", resDotGroup, safeAlias),
		)
	})

	Context("part check", func() {
		Context("HasAPI", func() {
			It("should return true if the API is scaffolded", func() {
				Expect(Resource{API: &API{CRDVersion: "v1"}}.HasAPI()).To(BeTrue())
			})

			DescribeTable("should return false if the API is not scaffolded",
				func(res Resource) { Expect(res.HasAPI()).To(BeFalse()) },
				Entry("nil API", Resource{API: nil}),
				Entry("empty CRD version", Resource{API: &API{}}),
			)
		})

		Context("HasController", func() {
			It("should return true if the controller is scaffolded", func() {
				Expect(Resource{Controller: true}.HasController()).To(BeTrue())
			})

			It("should return false if the controller is not scaffolded", func() {
				Expect(Resource{Controller: false}.HasController()).To(BeFalse())
			})
		})

		Context("HasDefaultingWebhook", func() {
			It("should return true if the defaulting webhook is scaffolded", func() {
				Expect(Resource{Webhooks: &Webhooks{Defaulting: true}}.HasDefaultingWebhook()).To(BeTrue())
			})

			DescribeTable("should return false if the defaulting webhook is not scaffolded",
				func(res Resource) { Expect(res.HasDefaultingWebhook()).To(BeFalse()) },
				Entry("nil webhooks", Resource{Webhooks: nil}),
				Entry("no defaulting", Resource{Webhooks: &Webhooks{Defaulting: false}}),
			)
		})

		Context("HasValidationWebhook", func() {
			It("should return true if the validation webhook is scaffolded", func() {
				Expect(Resource{Webhooks: &Webhooks{Validation: true}}.HasValidationWebhook()).To(BeTrue())
			})

			DescribeTable("should return false if the validation webhook is not scaffolded",
				func(res Resource) { Expect(res.HasValidationWebhook()).To(BeFalse()) },
				Entry("nil webhooks", Resource{Webhooks: nil}),
				Entry("no validation", Resource{Webhooks: &Webhooks{Validation: false}}),
			)
		})

		Context("HasConversionWebhook", func() {
			It("should return true if the conversion webhook is scaffolded", func() {
				Expect(Resource{Webhooks: &Webhooks{Conversion: true}}.HasConversionWebhook()).To(BeTrue())
			})

			DescribeTable("should return false if the conversion webhook is not scaffolded",
				func(res Resource) { Expect(res.HasConversionWebhook()).To(BeFalse()) },
				Entry("nil webhooks", Resource{Webhooks: nil}),
				Entry("no conversion", Resource{Webhooks: &Webhooks{Conversion: false}}),
			)
		})

		Context("IsRegularPlural", func() {
			It("should return true if the regular plural form is used", func() {
				Expect(res.IsRegularPlural()).To(BeTrue())
			})

			It("should return false if an irregular plural form is used", func() {
				Expect(Resource{GVK: gvk, Plural: "types"}.IsRegularPlural()).To(BeFalse())
			})
		})
	})

	Context("Copy", func() {
		const (
			path           = "api/v1"
			crdVersion     = "v1"
			webhookVersion = "v1"
		)

		res := Resource{
			GVK:    gvk,
			Plural: plural,
			Path:   path,
			API: &API{
				CRDVersion: crdVersion,
				Namespaced: true,
			},
			Controller: true,
			Webhooks: &Webhooks{
				WebhookVersion: webhookVersion,
				Defaulting:     true,
				Validation:     true,
				Conversion:     true,
			},
		}

		It("should return an exact copy", func() {
			other := res.Copy()
			Expect(other.Group).To(Equal(res.Group))
			Expect(other.Domain).To(Equal(res.Domain))
			Expect(other.Version).To(Equal(res.Version))
			Expect(other.Kind).To(Equal(res.Kind))
			Expect(other.Plural).To(Equal(res.Plural))
			Expect(other.Path).To(Equal(res.Path))
			Expect(other.API).NotTo(BeNil())
			Expect(other.API.CRDVersion).To(Equal(res.API.CRDVersion))
			Expect(other.API.Namespaced).To(Equal(res.API.Namespaced))
			Expect(other.Controller).To(Equal(res.Controller))
			Expect(other.Webhooks).NotTo(BeNil())
			Expect(other.Webhooks.WebhookVersion).To(Equal(res.Webhooks.WebhookVersion))
			Expect(other.Webhooks.Defaulting).To(Equal(res.Webhooks.Defaulting))
			Expect(other.Webhooks.Validation).To(Equal(res.Webhooks.Validation))
			Expect(other.Webhooks.Conversion).To(Equal(res.Webhooks.Conversion))
			Expect(other.Webhooks.Spoke).To(Equal(res.Webhooks.Spoke))
		})

		It("modifying the copy should not affect the original", func() {
			other := res.Copy()
			other.Group = "group2"
			other.Domain = "other.domain"
			other.Version = "v2"
			other.Kind = "kind2"
			other.Plural = "kind2s"
			other.Path = "api/v2"
			other.API.CRDVersion = v1beta1
			other.API.Namespaced = false
			other.API = nil // Change fields before changing pointer
			other.Controller = false
			other.Webhooks.WebhookVersion = v1beta1
			other.Webhooks.Defaulting = false
			other.Webhooks.Validation = false
			other.Webhooks.Conversion = false
			other.Webhooks = nil // Change fields before changing pointer

			Expect(res.Group).To(Equal(group))
			Expect(res.Domain).To(Equal(domain))
			Expect(res.Version).To(Equal(version))
			Expect(res.Kind).To(Equal(kind))
			Expect(res.Plural).To(Equal(plural))
			Expect(res.Path).To(Equal(path))
			Expect(res.API).NotTo(BeNil())
			Expect(res.API.CRDVersion).To(Equal(crdVersion))
			Expect(res.API.Namespaced).To(BeTrue())
			Expect(res.Controller).To(BeTrue())
			Expect(res.Webhooks).NotTo(BeNil())
			Expect(res.Webhooks.WebhookVersion).To(Equal(webhookVersion))
			Expect(res.Webhooks.Defaulting).To(BeTrue())
			Expect(res.Webhooks.Validation).To(BeTrue())
			Expect(res.Webhooks.Conversion).To(BeTrue())
		})
	})

	Context("Update", func() {
		var r, other Resource

		It("should fail for nil objects", func() {
			var nilResource *Resource
			Expect(nilResource.Update(other)).NotTo(Succeed())
		})

		It("should fail for different GVKs", func() {
			r = Resource{GVK: gvk}
			other = Resource{
				GVK: GVK{
					Group:   group,
					Domain:  domain,
					Version: version,
					Kind:    "OtherKind",
				},
			}
			Expect(r.Update(other)).NotTo(Succeed())
		})

		It("should fail for different Plurals", func() {
			r = Resource{
				GVK:    gvk,
				Plural: plural,
			}
			other = Resource{
				GVK:    gvk,
				Plural: "types",
			}
			Expect(r.Update(other)).NotTo(Succeed())
		})

		It("should work for a new path", func() {
			const path = "api/v1"
			r = Resource{GVK: gvk}
			other = Resource{
				GVK:  gvk,
				Path: path,
			}
			Expect(r.Update(other)).To(Succeed())
			Expect(r.Path).To(Equal(path))
		})

		It("should fail for different Paths", func() {
			r = Resource{
				GVK:  gvk,
				Path: "api/v1",
			}
			other = Resource{
				GVK:  gvk,
				Path: "apis/group/v1",
			}
			Expect(r.Update(other)).NotTo(Succeed())
		})

		Context("API", func() {
			It("should work with nil APIs", func() {
				r = Resource{GVK: gvk}
				other = Resource{
					GVK: gvk,
					API: &API{CRDVersion: v1},
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.API).NotTo(BeNil())
				Expect(r.API.CRDVersion).To(Equal(v1))
			})

			It("should fail if API.Update fails", func() {
				r = Resource{
					GVK: gvk,
					API: &API{CRDVersion: v1},
				}
				other = Resource{
					GVK: gvk,
					API: &API{CRDVersion: v1beta1},
				}
				Expect(r.Update(other)).NotTo(Succeed())
			})

			// The rest of the cases are tested in API.Update
		})

		Context("Controller", func() {
			It("should set the controller flag if provided and not previously set", func() {
				r = Resource{GVK: gvk}
				other = Resource{
					GVK:        gvk,
					Controller: true,
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Controller).To(BeTrue())
			})

			It("should keep the controller flag if previously set", func() {
				r = Resource{
					GVK:        gvk,
					Controller: true,
				}

				By("not providing it")
				other = Resource{GVK: gvk}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Controller).To(BeTrue())

				By("providing it")
				other = Resource{
					GVK:        gvk,
					Controller: true,
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Controller).To(BeTrue())
			})

			It("should not set the controller flag if not provided and not previously set", func() {
				r = Resource{GVK: gvk}
				other = Resource{GVK: gvk}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Controller).To(BeFalse())
			})
		})

		Context("Webhooks", func() {
			It("should work with nil Webhooks", func() {
				r = Resource{GVK: gvk}
				other = Resource{
					GVK:      gvk,
					Webhooks: &Webhooks{WebhookVersion: v1},
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Webhooks).NotTo(BeNil())
				Expect(r.Webhooks.WebhookVersion).To(Equal(v1))
			})

			It("should fail if Webhooks.Update fails", func() {
				r = Resource{
					GVK:      gvk,
					Webhooks: &Webhooks{WebhookVersion: v1},
				}
				other = Resource{
					GVK:      gvk,
					Webhooks: &Webhooks{WebhookVersion: v1beta1},
				}
				Expect(r.Update(other)).NotTo(Succeed())
			})

			// The rest of the cases are tested in Webhooks.Update
		})
	})

	Context("Replacer", func() {
		replacer := res.Replacer()

		DescribeTable("should replace the following strings",
			func(pattern, result string) { Expect(replacer.Replace(pattern)).To(Equal(result)) },
			Entry("no pattern", "version", "version"),
			Entry("pattern `%[group]`", "%[group]", res.Group),
			Entry("pattern `%[version]`", "%[version]", res.Version),
			Entry("pattern `%[kind]`", "%[kind]", "kind"),
			Entry("pattern `%[plural]`", "%[plural]", res.Plural),
			Entry("pattern `%[package-name]`", "%[package-name]", res.PackageName()),
		)
	})
})
