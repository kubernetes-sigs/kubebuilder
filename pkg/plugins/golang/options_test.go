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

package golang

import (
	"path"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
)

var _ = Describe("Options", func() {
	Context("Validate", func() {
		DescribeTable("should succeed for valid options",
			func(options *Options) { Expect(options.Validate()).To(Succeed()) },
			Entry("full GVK",
				&Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: "FirstMate"}),
			Entry("missing domain",
				&Options{Group: "crew", Version: "v1", Kind: "FirstMate"}),
			Entry("missing group",
				&Options{Domain: "test.io", Version: "v1", Kind: "FirstMate"}),
			Entry("kind with multiple initial uppercase characters",
				&Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: "FIRSTMate"}),
		)

		DescribeTable("should fail for invalid options",
			func(options *Options) { Expect(options.Validate()).NotTo(Succeed()) },
			Entry("group flag captured another flag",
				&Options{Group: "--version"}),
			Entry("version flag captured another flag",
				&Options{Version: "--kind"}),
			Entry("kind flag captured another flag",
				&Options{Kind: "--group"}),
			Entry("missing group and domain",
				&Options{Version: "v1", Kind: "FirstMate"}),
			Entry("group with uppercase characters",
				&Options{Group: "Crew", Domain: "test.io", Version: "v1", Kind: "FirstMate"}),
			Entry("group with non-alpha characters",
				&Options{Group: "crew1*?", Domain: "test.io", Version: "v1", Kind: "FirstMate"}),
			Entry("missing version",
				&Options{Group: "crew", Domain: "test.io", Kind: "FirstMate"}),
			Entry("version without v prefix",
				&Options{Group: "crew", Domain: "test.io", Version: "1", Kind: "FirstMate"}),
			Entry("unstable version without v prefix",
				&Options{Group: "crew", Domain: "test.io", Version: "1beta1", Kind: "FirstMate"}),
			Entry("unstable version with wrong prefix",
				&Options{Group: "crew", Domain: "test.io", Version: "a1beta1", Kind: "FirstMate"}),
			Entry("unstable version without alpha/beta number",
				&Options{Group: "crew", Domain: "test.io", Version: "v1beta", Kind: "FirstMate"}),
			Entry("multiple unstable version",
				&Options{Group: "crew", Domain: "test.io", Version: "v1beta1alpha1", Kind: "FirstMate"}),
			Entry("missing kind",
				&Options{Group: "crew", Domain: "test.io", Version: "v1"}),
			Entry("kind is too long",
				&Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: strings.Repeat("a", 64)}),
			Entry("kind with whitespaces",
				&Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: "First Mate"}),
			Entry("kind ends with `-`",
				&Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: "FirstMate-"}),
			Entry("kind starts with a decimal character",
				&Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: "1FirstMate"}),
			Entry("kind starts with a lowercase character",
				&Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: "firstMate"}),
			Entry("Invalid CRD version",
				&Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: "FirstMate", CRDVersion: "a"}),
			Entry("Invalid webhook version",
				&Options{Group: "crew", Domain: "test.io", Version: "v1", Kind: "FirstMate", WebhookVersion: "a"}),
		)
	})

	Context("NewResource", func() {
		var cfg config.Config

		BeforeEach(func() {
			cfg = cfgv3.New()
			_ = cfg.SetRepository("test")
		})

		DescribeTable("should succeed if the Resource is valid",
			func(options *Options) {
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
					Expect(resource.Group).To(Equal(options.Group))
					Expect(resource.Domain).To(Equal(options.Domain))
					Expect(resource.Version).To(Equal(options.Version))
					Expect(resource.Kind).To(Equal(options.Kind))
					if multiGroup {
						Expect(resource.Path).To(Equal(
							path.Join(cfg.GetRepository(), "apis", options.Group, options.Version)))
					} else {
						Expect(resource.Path).To(Equal(path.Join(cfg.GetRepository(), "api", options.Version)))
					}
					Expect(resource.API.CRDVersion).To(Equal(options.CRDVersion))
					Expect(resource.API.Namespaced).To(Equal(options.Namespaced))
					Expect(resource.Controller).To(Equal(options.DoController))
					Expect(resource.Webhooks.WebhookVersion).To(Equal(options.WebhookVersion))
					Expect(resource.Webhooks.Defaulting).To(Equal(options.DoDefaulting))
					Expect(resource.Webhooks.Validation).To(Equal(options.DoValidation))
					Expect(resource.Webhooks.Conversion).To(Equal(options.DoConversion))
					Expect(resource.QualifiedGroup()).To(Equal(options.Group + "." + options.Domain))
					Expect(resource.PackageName()).To(Equal(options.Group))
					Expect(resource.ImportAlias()).To(Equal(options.Group + options.Version))
				}
			},
			Entry("basic", &Options{
				Group:   "crew",
				Domain:  "test.io",
				Version: "v1",
				Kind:    "FirstMate",
			}),
			Entry("API", &Options{
				Group:      "crew",
				Domain:     "test.io",
				Version:    "v1",
				Kind:       "FirstMate",
				DoAPI:      true,
				CRDVersion: "v1",
				Namespaced: true,
			}),
			Entry("Controller", &Options{
				Group:        "crew",
				Domain:       "test.io",
				Version:      "v1",
				Kind:         "FirstMate",
				DoController: true,
			}),
			Entry("Webhooks", &Options{
				Group:          "crew",
				Domain:         "test.io",
				Version:        "v1",
				Kind:           "FirstMate",
				DoDefaulting:   true,
				DoValidation:   true,
				DoConversion:   true,
				WebhookVersion: "v1",
			}),
		)

		DescribeTable("should default the Plural by pluralizing the Kind",
			func(kind, plural string) {
				options := &Options{Group: "crew", Version: "v1", Kind: kind}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
					Expect(resource.Plural).To(Equal(plural))
				}
			},
			Entry("for `FirstMate`", "FirstMate", "firstmates"),
			Entry("for `Fish`", "Fish", "fish"),
			Entry("for `Helmswoman`", "Helmswoman", "helmswomen"),
		)

		DescribeTable("should keep the Plural if specified",
			func(kind, plural string) {
				options := &Options{Group: "crew", Version: "v1", Kind: kind, Plural: plural}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
					Expect(resource.Plural).To(Equal(plural))
				}
			},
			Entry("for `FirstMate`", "FirstMate", "mates"),
			Entry("for `Fish`", "Fish", "shoal"),
		)

		DescribeTable("should allow hyphens and dots in group names",
			func(group, safeGroup string) {
				options := &Options{
					Group:   group,
					Domain:  "test.io",
					Version: "v1",
					Kind:    "FirstMate",
				}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
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
			options := &Options{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			for _, multiGroup := range []bool{false, true} {
				if multiGroup {
					Expect(cfg.SetMultiGroup()).To(Succeed())
				}

				resource := options.NewResource(cfg)
				Expect(resource.QualifiedGroup()).To(Equal(options.Group))
			}
		})

		DescribeTable("should use core apis",
			func(group, qualified string) {
				options := &Options{
					Group:   group,
					Domain:  "test.io",
					Version: "v1",
					Kind:    "FirstMate",
				}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					if multiGroup {
						Expect(cfg.SetMultiGroup()).To(Succeed())
					}

					resource := options.NewResource(cfg)
					Expect(resource.Path).To(Equal(path.Join("k8s.io", "api", options.Group, options.Version)))
					Expect(resource.API.CRDVersion).To(Equal(""))
					Expect(resource.QualifiedGroup()).To(Equal(qualified))
				}
			},
			Entry("for `apps`", "apps", "apps"),
			Entry("for `authentication`", "authentication", "authentication.k8s.io"),
		)

		It("should use domain if the group is empty", func() {
			safeDomain := "testio"

			options := &Options{
				Domain:  "test.io",
				Version: "v1",
				Kind:    "FirstMate",
			}
			Expect(options.Validate()).To(Succeed())

			for _, multiGroup := range []bool{false, true} {
				if multiGroup {
					Expect(cfg.SetMultiGroup()).To(Succeed())
				}

				resource := options.NewResource(cfg)
				Expect(resource.Group).To(Equal(""))
				if multiGroup {
					Expect(resource.Path).To(Equal(path.Join(cfg.GetRepository(), "apis", options.Version)))
				} else {
					Expect(resource.Path).To(Equal(path.Join(cfg.GetRepository(), "api", options.Version)))
				}
				Expect(resource.QualifiedGroup()).To(Equal(options.Domain))
				Expect(resource.PackageName()).To(Equal(safeDomain))
				Expect(resource.ImportAlias()).To(Equal(safeDomain + options.Version))
			}
		})
	})
})
