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
			gvk = resource.GVK{
				Group:   group,
				Domain:  domain,
				Version: version,
				Kind:    kind,
			}

			cfg config.Config
		)

		BeforeEach(func() {
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
					Expect(res.QualifiedGroup()).To(Equal(gvk.Group + "." + gvk.Domain))
					Expect(res.PackageName()).To(Equal(gvk.Group))
					Expect(res.ImportAlias()).To(Equal(gvk.Group + gvk.Version))
				}
			},
			Entry("when updating nothing", Options{}),
			Entry("when updating the plural", Options{Plural: "mates"}),
			Entry("when updating the Controller", Options{DoController: true}),
		)

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
