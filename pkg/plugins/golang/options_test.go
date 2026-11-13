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
	sigsyaml "sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ = Describe("Options", func() {
	Context("ValidateControllerName", func() {
		// An older release let a named controller take the file of another kind's default
		// controller. Such a project is loaded, not built through the config API.
		const olderProject = `version: "3"
repo: test
resources:
- controller: true
  domain: test.io
  group: crew
  kind: Backup
  version: v1
- controllers:
  - name: backup
  domain: test.io
  group: crew
  kind: Captain
  version: v1
`

		It("should refuse the default controller of a kind when another kind records the same name", func() {
			cfg := cfgv3.New()
			Expect(cfg.UnmarshalYAML([]byte(olderProject))).To(Succeed())
			res := &resource.Resource{GVK: resource.GVK{Group: "crew", Domain: "test.io", Version: "v1", Kind: "Backup"}}

			err := Options{DoController: true}.ValidateControllerName(res, cfg)

			Expect(err).To(MatchError(ContainSubstring(`conflicts with controller "backup" of crew/v1/Captain`)))
		})

		// Kubebuilder v4.13.0 records this when the second version is created with --force: both
		// versions share one captain_controller.go and one CaptainReconciler.
		const sharedLegacyProject = `version: "3"
repo: test
resources:
- controller: true
  domain: test.io
  group: crew
  kind: Captain
  version: v1
- controller: true
  domain: test.io
  group: crew
  kind: Captain
  version: v2
`
		const sharedNamedProject = `version: "3"
repo: test
resources:
- controllers:
  - name: captain
  domain: test.io
  group: crew
  kind: Captain
  version: v1
- controllers:
  - name: captain
  domain: test.io
  group: crew
  kind: Captain
  version: v2
`

		DescribeTable("should refuse a name generating the reconciler of another controller the resource records",
			func(recorded, requested string) {
				cfg := cfgv3.New()
				gvk := resource.GVK{Group: "ship", Domain: "example.com", Version: "v1", Kind: "MyKind"}
				Expect(cfg.AddResource(resource.Resource{
					GVK: gvk, Plural: "mykinds", Controllers: &resource.Controllers{{Name: recorded}},
				})).To(Succeed())

				err := Options{DoController: true, ControllerName: requested}.
					ValidateControllerName(&resource.Resource{GVK: gvk}, cfg)

				Expect(err).To(MatchError(ContainSubstring("MyKindReconciler")))
			},
			Entry("for the default name beside a named controller", "my-kind", ""),
			Entry("for a name beside the default controller", "mykind", "my-kind"),
		)

		It("should accept the name the resource already records", func() {
			cfg := cfgv3.New()
			gvk := resource.GVK{Group: "ship", Domain: "example.com", Version: "v1", Kind: "MyKind"}
			Expect(cfg.AddResource(resource.Resource{
				GVK: gvk, Plural: "mykinds", Controllers: &resource.Controllers{{Name: "my-kind"}},
			})).To(Succeed())

			Expect(Options{DoController: true, ControllerName: "my-kind"}.
				ValidateControllerName(&resource.Resource{GVK: gvk}, cfg)).To(Succeed())
		})

		DescribeTable("should accept the default controller again for a version that shares it with another version",
			func(project, version string) {
				cfg := cfgv3.New()
				Expect(cfg.UnmarshalYAML([]byte(project))).To(Succeed())
				recorded, err := cfg.GetResources()
				Expect(err).NotTo(HaveOccurred())
				res := &resource.Resource{}
				for _, r := range recorded {
					if r.Version == version {
						res.GVK = r.GVK
					}
				}
				Expect(res.Version).To(Equal(version))

				Expect(Options{DoController: true}.ValidateControllerName(res, cfg)).To(Succeed())
			},
			Entry("for the first version, in the legacy form", sharedLegacyProject, "v1"),
			Entry("for the second version, in the legacy form", sharedLegacyProject, "v2"),
			Entry("for the first version, by name", sharedNamedProject, "v1"),
			Entry("for the second version, by name", sharedNamedProject, "v2"),
		)
	})

	Context("UpdateResource", func() {
		const (
			group         = "crew"
			domain        = "test.io"
			version       = "v1"
			kind          = "FirstMate"
			plural        = "firstmates"
			controller    = "firstmate"
			backupControl = "firstmate-backup"
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
					Expect(sigsyaml.Marshal(res)).NotTo(MatchRegexp(`(?m)^controller:`), "the legacy field must not be written")
					Expect(res.HasController()).To(Equal(options.DoController))
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

		Context("recording controllers", func() {
			var res resource.Resource

			BeforeEach(func() {
				res = resource.Resource{
					GVK:      gvk,
					Plural:   plural,
					API:      &resource.API{},
					Webhooks: &resource.Webhooks{},
				}
			})

			It("should record the default kind-based name when none is given", func() {
				Options{DoController: true}.UpdateResource(&res, cfg)

				Expect(res.GetControllerNames()).To(Equal([]string{controller}))
			})

			It("should record the given controller name", func() {
				Options{DoController: true, ControllerName: backupControl}.UpdateResource(&res, cfg)

				Expect(res.GetControllerNames()).To(Equal([]string{backupControl}))
			})

			It("should carry a legacy controller into the list before adding a new one", func() {
				// An old PROJECT file records the controller with the deprecated key.
				Expect(sigsyaml.Unmarshal([]byte("controller: true"), &res)).To(Succeed())

				Options{DoController: true, ControllerName: backupControl}.UpdateResource(&res, cfg)

				Expect(sigsyaml.Marshal(res)).NotTo(MatchRegexp(`(?m)^controller:`), "the legacy field must not be written")
				Expect(res.GetControllerNames()).To(Equal([]string{controller, backupControl}))
			})

			It("should not duplicate a controller that is already recorded", func() {
				Options{DoController: true}.UpdateResource(&res, cfg)
				Options{DoController: true}.UpdateResource(&res, cfg)

				Expect(res.GetControllerNames()).To(Equal([]string{controller}))
			})

			It("should record nothing when no controller is requested", func() {
				Options{}.UpdateResource(&res, cfg)

				Expect(res.HasController()).To(BeFalse())
			})
		})

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
})
