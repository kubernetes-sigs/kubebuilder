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
						Plural:   "firstmates",
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
						Expect(res.Path).To(Equal("testPath"))
					} else {
						// Core-resources have a path despite not having an API/Webhook but they are not tested here
						Expect(res.Path).To(Equal(""))
					}
					Expect(res.API).NotTo(BeNil())
					if options.DoAPI {
						Expect(res.API.Namespaced).To(Equal(options.Namespaced))
						Expect(res.API.IsEmpty()).To(BeFalse())
					} else {
						Expect(res.API.IsEmpty()).To(BeTrue())
					}
					Expect(res.HasController()).To(Equal(options.DoController))
					if options.DoController {
						expectedControllerName := options.Controller.Name
						if expectedControllerName == "" {
							expectedControllerName = "firstmate"
						}
						Expect(res.Controllers).To(Equal([]resource.Controller{{Name: expectedControllerName}}))
					}
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
			Entry("when updating the Controller with custom name",
				Options{DoController: true, Controller: resource.Controller{Name: "firstmate-backup"}}),
			Entry("when updating with External API Path", Options{ExternalAPIPath: "testPath", ExternalAPIDomain: "test.io"}),
			Entry("when updating the API with setting webhooks params",
				Options{DoAPI: true, DoDefaulting: true, DoValidation: true, DoConversion: true}),
		)

		It("should reuse existing API path when creating an internal controller-only resource", func() {
			existing := resource.Resource{
				GVK: gvk,
				API: &resource.API{
					CRDVersion: "v1",
					Namespaced: true,
				},
				Path: "existing/api/path",
			}
			Expect(cfg.UpdateResource(existing)).To(Succeed())

			res := resource.Resource{
				GVK:      gvk,
				Plural:   "firstmates",
				API:      &resource.API{},
				Webhooks: &resource.Webhooks{},
			}

			options := Options{
				DoAPI:        false,
				DoController: true,
			}
			options.UpdateResource(&res, cfg)

			Expect(res.Path).To(Equal("existing/api/path"))
			Expect(res.Controllers).To(Equal([]resource.Controller{{Name: "firstmate"}}))
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
						Plural:   "firstmates",
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
						Plural:   "firstmates",
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
})
