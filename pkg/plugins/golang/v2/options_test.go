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

package v2

import (
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
)

var _ = Describe("Options", func() {
	Context("Validate", func() {
		DescribeTable("should succeed for valid options",
			func(options Options) { Expect(options.Validate()).To(Succeed()) },
			Entry("full GVK", Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: "FirstMate"}),
			Entry("missing domain", Options{Group: "crew", Version: "v1", Kind: "FirstMate"}),
		)

		DescribeTable("should fail for invalid options",
			func(options Options) { Expect(options.Validate()).NotTo(Succeed()) },
			Entry("group flag captured another flag", Options{Group: "--version"}),
			Entry("version flag captured another flag", Options{Version: "--kind"}),
			Entry("kind flag captured another flag", Options{Kind: "--group"}),
			Entry("missing group", Options{Domain: "test.io", Version: "v1", Kind: "FirstMate"}),
			Entry("missing version", Options{Group: "crew", Domain: "test.io", Kind: "FirstMate"}),
			Entry("missing kind", Options{Group: "crew", Domain: "test.io", Version: "v1"}),
		)
	})

	Context("NewResource", func() {
		var cfg config.Config

		BeforeEach(func() {
			cfg = cfgv2.New()
			_ = cfg.SetRepository("test")
		})

		DescribeTable("should succeed if the Resource is valid",
			func(options Options) {
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					} else {
						Expect(cfg.ClearMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
					Expect(resource.Validate()).To(Succeed())
					Expect(resource.Group).To(Equal(options.Group))
					Expect(resource.Domain).To(Equal(options.Domain))
					Expect(resource.Version).To(Equal(options.Version))
					Expect(resource.Kind).To(Equal(options.Kind))
					Expect(resource.API).NotTo(BeNil())
					if options.DoAPI || options.DoDefaulting || options.DoValidation || options.DoConversion {
						if multiGroup {
							Expect(resource.Path).To(Equal(
								path.Join(cfg.GetRepository(), "apis", options.Group, options.Version)))
						} else {
							Expect(resource.Path).To(Equal(path.Join(cfg.GetRepository(), "api", options.Version)))
						}
					} else {
						// Core-resources have a path despite not having an API/Webhook but they are not tested here
						Expect(resource.Path).To(Equal(""))
					}
					if options.DoAPI {
						Expect(resource.API.CRDVersion).To(Equal(options.CRDVersion))
						Expect(resource.API.Namespaced).To(Equal(options.Namespaced))
						Expect(resource.API.IsEmpty()).To(BeFalse())
					} else {
						Expect(resource.API.IsEmpty()).To(BeTrue())
					}
					Expect(resource.Controller).To(Equal(options.DoController))
					Expect(resource.Webhooks).NotTo(BeNil())
					if options.DoDefaulting || options.DoValidation || options.DoConversion {
						Expect(resource.Webhooks.WebhookVersion).To(Equal(options.WebhookVersion))
						Expect(resource.Webhooks.Defaulting).To(Equal(options.DoDefaulting))
						Expect(resource.Webhooks.Validation).To(Equal(options.DoValidation))
						Expect(resource.Webhooks.Conversion).To(Equal(options.DoConversion))
						Expect(resource.Webhooks.IsEmpty()).To(BeFalse())
					} else {
						Expect(resource.Webhooks.IsEmpty()).To(BeTrue())
					}
					Expect(resource.QualifiedGroup()).To(Equal(options.Group + "." + options.Domain))
					Expect(resource.PackageName()).To(Equal(options.Group))
					Expect(resource.ImportAlias()).To(Equal(options.Group + options.Version))
				}
			},
			Entry("basic", Options{
				Group:   "crew",
				Domain:  "test.io",
				Version: "v1",
				Kind:    "FirstMate",
			}),
			Entry("API", Options{
				Group:      "crew",
				Domain:     "test.io",
				Version:    "v1",
				Kind:       "FirstMate",
				DoAPI:      true,
				CRDVersion: "v1beta1",
				Namespaced: true,
			}),
			Entry("Controller", Options{
				Group:        "crew",
				Domain:       "test.io",
				Version:      "v1",
				Kind:         "FirstMate",
				DoController: true,
			}),
			Entry("Webhooks", Options{
				Group:          "crew",
				Domain:         "test.io",
				Version:        "v1",
				Kind:           "FirstMate",
				DoDefaulting:   true,
				DoValidation:   true,
				DoConversion:   true,
				WebhookVersion: "v1beta1",
			}),
		)

		DescribeTable("should default the Plural by pluralizing the Kind",
			func(kind, plural string) {
				options := Options{Group: "crew", Version: "v1", Kind: kind}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					} else {
						Expect(cfg.ClearMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
					Expect(resource.Validate()).To(Succeed())
					Expect(resource.Plural).To(Equal(plural))
				}
			},
			Entry("for `FirstMate`", "FirstMate", "firstmates"),
			Entry("for `Fish`", "Fish", "fish"),
			Entry("for `Helmswoman`", "Helmswoman", "helmswomen"),
		)

		DescribeTable("should keep the Plural if specified",
			func(kind, plural string) {
				options := Options{Group: "crew", Version: "v1", Kind: kind, Plural: plural}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					} else {
						Expect(cfg.ClearMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
					Expect(resource.Validate()).To(Succeed())
					Expect(resource.Plural).To(Equal(plural))
				}
			},
			Entry("for `FirstMate`", "FirstMate", "mates"),
			Entry("for `Fish`", "Fish", "shoal"),
		)

		DescribeTable("should allow hyphens and dots in group names",
			func(group, safeGroup string) {
				options := Options{
					Group:   group,
					Domain:  "test.io",
					Version: "v1",
					Kind:    "FirstMate",
					DoAPI:   true, // Scaffold the API so that the path is saved
				}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					} else {
						Expect(cfg.ClearMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
					Expect(resource.Validate()).To(Succeed())
					Expect(resource.Group).To(Equal(options.Group))
					if multiGroup {
						Expect(resource.Path).To(Equal(
							path.Join(cfg.GetRepository(), "apis", options.Group, options.Version)))
					} else {
						Expect(resource.Path).To(Equal(path.Join(cfg.GetRepository(), "api", options.Version)))
					}
					Expect(resource.QualifiedGroup()).To(Equal(options.Group + "." + options.Domain))
					Expect(resource.PackageName()).To(Equal(safeGroup))
					Expect(resource.ImportAlias()).To(Equal(safeGroup + options.Version))
				}
			},
			Entry("for hyphen-containing group", "my-project", "myproject"),
			Entry("for dot-containing group", "my.project", "myproject"),
		)

		It("should not append '.' if provided an empty domain", func() {
			options := Options{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			for _, multiGroup := range []bool{false, true} {
				if multiGroup {
					Expect(cfg.SetMultiGroup()).To(Succeed())
				} else {
					Expect(cfg.ClearMultiGroup()).To(Succeed())
				}

				resource := options.NewResource(cfg)
				Expect(resource.Validate()).To(Succeed())
				Expect(resource.QualifiedGroup()).To(Equal(options.Group))
			}
		})

		DescribeTable("should use core apis",
			func(group, qualified string) {
				options := Options{
					Group:   group,
					Domain:  "test.io",
					Version: "v1",
					Kind:    "FirstMate",
				}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					} else {
						Expect(cfg.ClearMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
					Expect(resource.Validate()).To(Succeed())
					Expect(resource.Path).To(Equal(path.Join("k8s.io", "api", options.Group, options.Version)))
					Expect(resource.API).NotTo(BeNil())
					Expect(resource.API.IsEmpty()).To(BeTrue())
					Expect(resource.QualifiedGroup()).To(Equal(qualified))
				}
			},
			Entry("for `apps`", "apps", "apps"),
			Entry("for `authentication`", "authentication", "authentication.k8s.io"),
		)
	})
})
