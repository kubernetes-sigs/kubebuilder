package golang

import (
	"path"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
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
		DescribeTable("should succeed if the Resource is valid",
			func(options *Options) {
				cfg := &config.Config{Repo: "test"}

				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					cfg.MultiGroup = multiGroup
					resource := options.NewResource(cfg)

					Expect(resource.Group).To(Equal(options.Group))
					Expect(resource.Domain).To(Equal(options.Domain))
					Expect(resource.Version).To(Equal(options.Version))
					Expect(resource.Kind).To(Equal(options.Kind))
					if cfg.MultiGroup {
						Expect(resource.Path).To(Equal(path.Join(cfg.Repo, "apis", options.Group, options.Version)))
					} else {
						Expect(resource.Path).To(Equal(path.Join(cfg.Repo, "api", options.Version)))
					}
					Expect(resource.API.Version).To(Equal(options.CRDVersion))
					Expect(resource.API.Namespaced).To(Equal(options.Namespaced))
					Expect(resource.Controller).To(Equal(options.DoController))
					Expect(resource.Webhooks.Version).To(Equal(options.WebhookVersion))
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

		It("should default the Plural by pluralizing the Kind", func() {
			cfg := &config.Config{}

			for kind, plural := range map[string]string{
				"FirstMate":  "firstmates",
				"Fish":       "fish",
				"Helmswoman": "helmswomen",
			} {
				options := &Options{Group: "crew", Version: "v1", Kind: kind}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					cfg.MultiGroup = multiGroup

					resource := options.NewResource(cfg)
					Expect(resource.Plural).To(Equal(plural))
				}
			}
		})

		It("should keep the Plural if specified", func() {
			cfg := &config.Config{}

			for kind, plural := range map[string]string{
				"FirstMate": "mates",
				"Fish":      "shoal",
			} {
				options := &Options{Group: "crew", Version: "v1", Kind: kind, Plural: plural}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					cfg.MultiGroup = multiGroup

					resource := options.NewResource(cfg)
					Expect(resource.Plural).To(Equal(plural))
				}
			}
		})

		It("should allow hyphens and dots in group names", func() {
			cfg := &config.Config{Repo: "test"}

			for group, safeGroup := range map[string]string{
				"my-project": "myproject",
				"my.project": "myproject",
			} {
				options := &Options{
					Group:   group,
					Domain:  "test.io",
					Version: "v1",
					Kind:    "FirstMate",
				}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					cfg.MultiGroup = multiGroup

					resource := options.NewResource(cfg)
					Expect(resource.Group).To(Equal(options.Group))
					if cfg.MultiGroup {
						Expect(resource.Path).To(Equal(path.Join(cfg.Repo, "apis", options.Group, options.Version)))
					} else {
						Expect(resource.Path).To(Equal(path.Join(cfg.Repo, "api", options.Version)))
					}
					Expect(resource.QualifiedGroup()).To(Equal(options.Group + "." + options.Domain))
					Expect(resource.PackageName()).To(Equal(safeGroup))
					Expect(resource.ImportAlias()).To(Equal(safeGroup + options.Version))
				}
			}
		})

		It("should not append '.' if provided an empty domain", func() {
			cfg := &config.Config{}
			options := &Options{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			for _, multiGroup := range []bool{false, true} {
				cfg.MultiGroup = multiGroup

				resource := options.NewResource(cfg)
				Expect(resource.QualifiedGroup()).To(Equal(options.Group))
			}
		})

		It("should use core apis", func() {
			cfg := &config.Config{Repo: "test"}

			for group, qualified := range map[string]string{
				"apps":           "apps",
				"authentication": "authentication.k8s.io",
			} {
				options := &Options{
					Group:   group,
					Domain:  "test.io",
					Version: "v1",
					Kind:    "FirstMate",
				}
				Expect(options.Validate()).To(Succeed())

				for _, multiGroup := range []bool{false, true} {
					cfg.MultiGroup = multiGroup

					resource := options.NewResource(cfg)
					Expect(resource.Path).To(Equal(path.Join("k8s.io", "api", options.Group, options.Version)))
					Expect(resource.API.Version).To(Equal(""))
					Expect(resource.QualifiedGroup()).To(Equal(qualified))
				}
			}
		})

		It("should use domain if the group is empty", func() {
			cfg := &config.Config{Repo: "test"}
			safeDomain := "testio"

			options := &Options{
				Domain:  "test.io",
				Version: "v1",
				Kind:    "FirstMate",
			}
			Expect(options.Validate()).To(Succeed())

			for _, multiGroup := range []bool{false, true} {
				cfg.MultiGroup = multiGroup

				resource := options.NewResource(cfg)
				Expect(resource.Group).To(Equal(""))
				if cfg.MultiGroup {
					Expect(resource.Path).To(Equal(path.Join(cfg.Repo, "apis", options.Version)))
				} else {
					Expect(resource.Path).To(Equal(path.Join(cfg.Repo, "api", options.Version)))
				}
				Expect(resource.QualifiedGroup()).To(Equal(options.Domain))
				Expect(resource.PackageName()).To(Equal(safeDomain))
				Expect(resource.ImportAlias()).To(Equal(safeDomain + options.Version))
			}
		})
	})
})
