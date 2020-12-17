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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

//nolint:dupl
var _ = Describe("Resource", func() {
	const (
		group   = "group"
		domain  = "test.io"
		version = "v1"
		kind    = "Kind"
	)

	var (
		res1 = Resource{
			GVK: GVK{
				Group:   group,
				Domain:  domain,
				Version: version,
				Kind:    kind,
			},
		}
		res2 = Resource{
			GVK: GVK{
				// Empty group
				Domain:  domain,
				Version: version,
				Kind:    kind,
			},
		}
		res3 = Resource{
			GVK: GVK{
				Group: group,
				// Empty domain
				Version: version,
				Kind:    kind,
			},
		}
	)

	Context("compound field", func() {
		const (
			safeDomain    = "testio"
			groupVersion  = group + version
			domainVersion = safeDomain + version
		)

		DescribeTable("PackageName should return the correct string",
			func(res Resource, packageName string) { Expect(res.PackageName()).To(Equal(packageName)) },
			Entry("fully qualified resource", res1, group),
			Entry("empty group name", res2, safeDomain),
			Entry("empty domain", res3, group),
		)

		DescribeTable("ImportAlias",
			func(res Resource, importAlias string) { Expect(res.ImportAlias()).To(Equal(importAlias)) },
			Entry("fully qualified resource", res1, groupVersion),
			Entry("empty group name", res2, domainVersion),
			Entry("empty domain", res3, groupVersion),
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
	})

	Context("Copy", func() {
		const (
			plural         = "kinds"
			path           = "api/v1"
			crdVersion     = "v1"
			webhookVersion = "v1"
		)

		res := Resource{
			GVK: GVK{
				Group:   group,
				Domain:  domain,
				Version: version,
				Kind:    kind,
			},
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
		})

		It("modifying the copy should not affect the original", func() {
			other := res.Copy()
			other.Group = "group2"
			other.Domain = "other.domain"
			other.Version = "v2"
			other.Kind = "kind2"
			other.Plural = "kind2s"
			other.Path = "api/v2"
			other.API.CRDVersion = "v1beta1"
			other.API.Namespaced = false
			other.API = nil // Change fields before changing pointer
			other.Controller = false
			other.Webhooks.WebhookVersion = "v1beta1"
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
			r = Resource{
				GVK: GVK{
					Group:   group,
					Version: version,
					Kind:    kind,
				},
			}
			other = Resource{
				GVK: GVK{
					Group:   group,
					Version: version,
					Kind:    "OtherKind",
				},
			}
			Expect(r.Update(other)).NotTo(Succeed())
		})

		Context("API", func() {
			It("should work with nil APIs", func() {
				r = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
				}
				other = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
					API: &API{CRDVersion: v1},
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.API).NotTo(BeNil())
				Expect(r.API.CRDVersion).To(Equal(v1))
			})

			It("should fail if API.Update fails", func() {
				r = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
					API: &API{CRDVersion: v1},
				}
				other = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
					API: &API{CRDVersion: "v1beta1"},
				}
				Expect(r.Update(other)).NotTo(Succeed())
			})

			// The rest of the cases are tested in API.Update
		})

		Context("Controller", func() {
			It("should set the controller flag if provided and not previously set", func() {
				r = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
				}
				other = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
					Controller: true,
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Controller).To(BeTrue())
			})

			It("should keep the controller flag if previously set", func() {
				r = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
					Controller: true,
				}

				By("not providing it")
				other = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Controller).To(BeTrue())

				By("providing it")
				other = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
					Controller: true,
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Controller).To(BeTrue())
			})

			It("should not set the controller flag if not provided and not previously set", func() {
				r = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
				}
				other = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Controller).To(BeFalse())
			})
		})

		Context("Webhooks", func() {
			It("should work with nil Webhooks", func() {
				r = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
				}
				other = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
					Webhooks: &Webhooks{WebhookVersion: v1},
				}
				Expect(r.Update(other)).To(Succeed())
				Expect(r.Webhooks).NotTo(BeNil())
				Expect(r.Webhooks.WebhookVersion).To(Equal(v1))
			})

			It("should fail if Webhooks.Update fails", func() {
				r = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
					Webhooks: &Webhooks{WebhookVersion: v1},
				}
				other = Resource{
					GVK: GVK{
						Group:   group,
						Version: version,
						Kind:    kind,
					},
					Webhooks: &Webhooks{WebhookVersion: "v1beta1"},
				}
				Expect(r.Update(other)).NotTo(Succeed())
			})

			// The rest of the cases are tested in Webhooks.Update
		})
	})

	Context("Replacer", func() {
		res := Resource{
			GVK: GVK{
				Group:   group,
				Domain:  domain,
				Version: version,
				Kind:    kind,
			},
			Plural: "kinds",
		}
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
