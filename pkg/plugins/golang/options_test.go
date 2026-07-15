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

package golang

import (
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ = Describe("Options", func() {
	Context("UpdateResource", func() {
		const (
			group   = "crew"
			domain  = "test.io"
			version = "v1"
			kind    = "FirstMate"
			plural  = "firstmates"
		)

		var (
			gvk resource.GVK
			cfg config.Config
		)

		BeforeEach(func() {
			gvk = resource.GVK{
				Group:   group,
				Domain:  domain,
				Version: version,
				Kind:    kind,
			}

			cfg = cfgv3.New()
			_ = cfg.SetRepository("test")
		})

		DescribeTable("should succeed",
			func(options Options) {
				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					} else {
						Expect(cfg.ClearMultiGroup()).To(Succeed())
					}

					res := resource.Resource{
						GVK:      gvk,
						Plural:   plural,
						API:      &resource.API{},
						Webhooks: &resource.Webhooks{},
					}

					options.UpdateResource(&res, cfg)
					Expect(res.Validate()).To(Succeed())
					Expect(res.GVK.IsEqualTo(gvk)).To(BeTrue())
					if options.Plural != "" {
						Expect(res.Plural).To(Equal(options.Plural))
					}
					if options.DoAPI || options.DoDefaulting || options.DoValidation || options.DoConversion {
						if multiGroup {
							Expect(res.Path).To(Equal(
								path.Join(cfg.GetRepository(), "api", gvk.Group, gvk.Version)))
						} else {
							Expect(res.Path).To(Equal(path.Join(cfg.GetRepository(), "api", gvk.Version)))
						}
					} else if len(options.ExternalAPIPath) > 0 {
						Expect(res.Path).To(Equal("github.com/example/external/api/v1"))
					} else {
						// Core-resources have a path despite not having an API/Webhook but they are not tested here
						Expect(res.Path).To(Equal(""))
					}
					Expect(res.API).NotTo(BeNil())
					if options.DoAPI {
						Expect(res.API.Namespaced).To(Equal(options.Namespaced))
						Expect(res.API.SSA).To(Equal(options.SSA))
						Expect(res.API.IsEmpty()).To(BeFalse())
					} else {
						Expect(res.API.IsEmpty()).To(BeTrue())
					}
					Expect(res.Controller).To(Equal(options.DoController))
					Expect(res.Webhooks).NotTo(BeNil())
					if options.DoDefaulting || options.DoValidation || options.DoConversion {
						Expect(res.Webhooks.Defaulting).To(Equal(options.DoDefaulting))
						Expect(res.Webhooks.Validation).To(Equal(options.DoValidation))
						Expect(res.Webhooks.Conversion).To(Equal(options.DoConversion))
						Expect(res.Webhooks.Spoke).To(Equal(options.Spoke))
						Expect(res.Webhooks.IsEmpty()).To(BeFalse())
					} else {
						Expect(res.Webhooks.IsEmpty()).To(BeTrue())
					}

					if len(options.ExternalAPIPath) > 0 {
						Expect(res.External).To(BeTrue())
						Expect(res.Domain).To(Equal("test.io"))
					}

					Expect(res.QualifiedGroup()).To(Equal(gvk.Group + "." + gvk.Domain))
					Expect(res.PackageName()).To(Equal(gvk.Group))
					Expect(res.ImportAlias()).To(Equal(gvk.Group + gvk.Version))
				}
			},
			Entry("when updating nothing", Options{}),
			Entry("when updating the plural", Options{Plural: "mates"}),
			Entry("when updating the Controller", Options{DoController: true}),
			Entry("when updating with External API Path",
				Options{ExternalAPIPath: "github.com/example/external/api/v1", ExternalAPIDomain: "test.io"}),
			Entry("when updating the API with setting webhooks params",
				Options{DoAPI: true, DoDefaulting: true, DoValidation: true, DoConversion: true}),
			Entry("when updating the API with SSA enabled", Options{DoAPI: true, SSA: true}),
		)

		It("should retain path and external flag when ExternalAPIPath is not provided but resource is already external",
			func() {
				const externalPath = "github.com/example/external/api/v1"

				res := resource.Resource{
					GVK:      gvk,
					Plural:   plural,
					External: true,
					Path:     externalPath,
					API:      &resource.API{},
					Webhooks: &resource.Webhooks{},
				}

				Options{DoController: true}.UpdateResource(&res, cfg)

				Expect(res.External).To(BeTrue())
				Expect(res.Path).To(Equal(externalPath))
				Expect(res.Validate()).To(Succeed())
			})

		It("should preserve res.Domain when ExternalAPIDomain is not supplied in !alreadyHasAPI block",
			func() {
				const externalPath = "github.com/example/external/api/v1"
				const externalDomain = "example.com"

				res := resource.Resource{
					GVK: resource.GVK{
						Group:   group,
						Domain:  externalDomain,
						Version: version,
						Kind:    kind,
					},
					Plural:   plural,
					External: true,
					Path:     externalPath,
					API:      &resource.API{},
					Webhooks: &resource.Webhooks{},
				}

				// DoDefaulting without ExternalAPIPath — simulates webhook flow without re-providing the flag
				Options{DoDefaulting: true}.UpdateResource(&res, cfg)

				Expect(res.External).To(BeTrue())
				Expect(res.GVK.Domain).To(Equal(externalDomain),
					"domain must not be zeroed out when ExternalAPIDomain is empty")
				Expect(res.Path).To(Equal(externalPath),
					"path must not be clobbered by webhook block for external resource")
				Expect(res.Validate()).To(Succeed())
			})

		DescribeTable("should use core apis",
			func(group, qualified string) {
				options := Options{}
				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					} else {
						Expect(cfg.ClearMultiGroup()).To(Succeed())
					}

					res := resource.Resource{
						GVK: resource.GVK{
							Group:   group,
							Domain:  domain,
							Version: version,
							Kind:    kind,
						},
						Plural:   plural,
						API:      &resource.API{},
						Webhooks: &resource.Webhooks{},
					}

					options.UpdateResource(&res, cfg)
					Expect(res.Validate()).To(Succeed())

					Expect(res.Path).To(Equal(path.Join("k8s.io", "api", group, version)))
					Expect(res.HasAPI()).To(BeFalse())
					Expect(res.QualifiedGroup()).To(Equal(qualified))
				}
			},
			Entry("for `apps`", "apps", "apps"),
			Entry("for `authentication`", "authentication", "authentication.k8s.io"),
		)

		DescribeTable("should use core apis with project version 2",
			// This needs a separate test because project version 2 didn't store API and therefore
			// the `HasAPI` method of the resource obtained with `GetResource` will always return false.
			// Instead, the existence of a resource in the list means the API was scaffolded.
			func(group, qualified string) {
				cfg = cfgv3.New()
				_ = cfg.SetRepository("test")

				options := Options{}
				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					} else {
						Expect(cfg.ClearMultiGroup()).To(Succeed())
					}

					res := resource.Resource{
						GVK: resource.GVK{
							Group:   group,
							Domain:  domain,
							Version: version,
							Kind:    kind,
						},
						Plural:   plural,
						API:      &resource.API{},
						Webhooks: &resource.Webhooks{},
					}

					options.UpdateResource(&res, cfg)
					Expect(res.Validate()).To(Succeed())

					Expect(res.Path).To(Equal(path.Join("k8s.io", "api", group, version)))
					Expect(res.HasAPI()).To(BeFalse())
					Expect(res.QualifiedGroup()).To(Equal(qualified))
				}
			},
			Entry("for `apps`", "apps", "apps"),
			Entry("for `authentication`", "authentication", "authentication.k8s.io"),
		)
	})

	Context("ReconcileDomainAlias", func() {
		const (
			projectDomain  = "test.io"
			externalDomain = "cert-manager.io"
		)

		var cfg config.Config

		newRes := func(domain string) *resource.Resource {
			return &resource.Resource{
				GVK: resource.GVK{Group: "cert-manager", Domain: domain, Version: "v1", Kind: "Issuer"},
			}
		}

		BeforeEach(func() {
			cfg = cfgv3.New()
			Expect(cfg.SetDomain(projectDomain)).To(Succeed())
		})

		It("is a no-op when the deprecated flag is unset", func() {
			res := newRes(projectDomain)

			Expect(Options{}.ReconcileDomainAlias(res, cfg)).To(Succeed())
			Expect(res.Domain).To(Equal(projectDomain))
		})

		It("is a no-op when both flags agree", func() {
			res := newRes(externalDomain)
			opts := Options{ExternalAPIDomain: externalDomain}

			Expect(opts.ReconcileDomainAlias(res, cfg)).To(Succeed())
			Expect(res.Domain).To(Equal(externalDomain))
		})

		It("keeps working as the alias when the ambiguity went unresolved", func() {
			res := newRes("")
			opts := Options{ExternalAPIDomain: externalDomain}

			Expect(opts.ReconcileDomainAlias(res, cfg)).To(Succeed())
			Expect(res.Domain).To(Equal(externalDomain))
		})

		It("keeps working as the alias when the resource is untracked", func() {
			res := newRes(projectDomain)
			opts := Options{ExternalAPIDomain: externalDomain}

			Expect(opts.ReconcileDomainAlias(res, cfg)).To(Succeed())
			Expect(res.Domain).To(Equal(externalDomain))
		})

		It("refuses two different values rather than letting one win", func() {
			res := newRes("other.io")
			opts := Options{ExternalAPIDomain: externalDomain}

			err := opts.ReconcileDomainAlias(res, cfg)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("conflicting values"))
			Expect(err.Error()).To(ContainSubstring("other.io"))
			Expect(err.Error()).To(ContainSubstring(externalDomain))
			Expect(res.Domain).To(Equal("other.io"))
		})
	})
})
