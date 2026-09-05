/*
Copyright 2026 The Kubernetes Authors.

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

package v4

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

// coreAPIPath is the Go import path recorded for a core (Kubernetes built-in) resource in tests.
const coreAPIPath = "k8s.io/api/core/v1"

var _ = Describe("createWebhookSubcommand", func() {
	var (
		subCmd *createWebhookSubcommand
		cfg    config.Config
		res    *resource.Resource
	)

	BeforeEach(func() {
		subCmd = &createWebhookSubcommand{}
		cfg = cfgv3.New()
		_ = cfg.SetRepository("github.com/example/test")

		subCmd.options = &goPlugin.Options{}
		res = &resource.Resource{
			GVK: resource.GVK{
				Group:   crewGroup,
				Domain:  testIO,
				Version: "v1",
				Kind:    captainKind,
			},
			Plural:   captains,
			Webhooks: &resource.Webhooks{},
		}
	})

	Context("UpdateMetadata", func() {
		It("should provide webhook examples", func() {
			meta := &plugin.SubcommandMetadata{}

			subCmd.UpdateMetadata(plugin.CLIMetadata{CommandName: testCommandName}, meta)

			Expect(meta.Examples).To(ContainSubstring("--defaulting --programmatic-validation"))
			Expect(meta.Examples).To(ContainSubstring("--conversion --spoke v1"))
			Expect(meta.Examples).To(ContainSubstring("--defaulting-path=/my-custom-mutate-path"))
			Expect(meta.Examples).To(ContainSubstring("--validation-path=/my-custom-validate-path"))
		})
	})

	It("should reject defaulting-path without --defaulting", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		subCmd.options.DefaultingPath = "/custom-path"
		subCmd.options.DoDefaulting = false

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--defaulting-path can only be used with --defaulting"))
	})

	It("should reject validation-path without --programmatic-validation", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		subCmd.options.ValidationPath = "/custom-path"
		subCmd.options.DoValidation = false

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--validation-path can only be used with --programmatic-validation"))
	})

	It("should require external-api-path when using external-api-module", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		subCmd.options.ExternalAPIModule = externalAPIModuleWithVersion
		subCmd.options.ExternalAPIPath = ""
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("requires '--external-api-path'"))
	})

	It("should reject external-api-path with module version", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		subCmd.options.ExternalAPIPath = externalAPIModuleWithVersion
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("version specifiers belong in the module field"))
	})

	It("should reject bare relative external-api-path", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		subCmd.options.ExternalAPIPath = relativeAPIPath
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("must be a fully-qualified Go import path"))
	})

	It("should reject leading-dot pseudo-domain external-api-path", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		subCmd.options.ExternalAPIPath = ".com/org/repo/api/v1"
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("must be a fully-qualified Go import path"))
	})

	It("should reject malformed external-api-path", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		subCmd.options.ExternalAPIPath = "a//b"
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("malformed import path"))
		Expect(err.Error()).To(ContainSubstring("double slash"))
	})

	It("should allow creating a webhook for an external resource with a valid path", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		subCmd.options.ExternalAPIPath = "github.com/example/external/api/v1"
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).NotTo(HaveOccurred())
		Expect(res.External).To(BeTrue())
		Expect(res.Path).To(Equal("github.com/example/external/api/v1"))
	})

	It("should retain path for existing external resource without re-providing --external-api-path", func() {
		const externalPath = "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

		Expect(subCmd.InjectConfig(cfg)).To(Succeed())

		// Simulate an existing external resource stored in PROJECT.
		// GVK domain stays as testIO because the lookup key is constructed from the CLI flags
		// (--group, --version, --kind) with the project domain — not the external-api-domain.
		storedRes := *res
		storedRes.External = true
		storedRes.Path = externalPath
		Expect(cfg.AddResource(storedRes)).To(Succeed())

		// User runs: kubebuilder create webhook --group crew --version v1 --kind Captain --defaulting
		// without re-supplying --external-api-path
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).NotTo(HaveOccurred())
		Expect(res.External).To(BeTrue())
		Expect(res.Path).To(Equal(externalPath))
		Expect(res.Domain).To(Equal(testIO))
	})

	It("should reuse external API configuration from PROJECT without external flags", func() {
		const externalPath = "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
		const externalDomain = "cert-manager.io"

		Expect(subCmd.InjectConfig(cfg)).To(Succeed())

		// Simulate an external API in the PROJECT file
		storedRes := *res
		storedRes.External = true
		storedRes.Path = externalPath
		storedRes.Domain = externalDomain
		Expect(cfg.AddResource(storedRes)).To(Succeed())

		// Simulate user running create webhook without any external flags.
		// res.Domain will be the default project domain initially.
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		// Expected: success, and fields are correctly reused.
		Expect(err).NotTo(HaveOccurred())
		Expect(res.External).To(BeTrue())
		Expect(res.Path).To(Equal(externalPath))
		Expect(res.Domain).To(Equal(externalDomain))
	})

	It("should return improved error message when API is missing", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no API found for"))
		Expect(err.Error()).To(ContainSubstring("run 'create api' first"))
		Expect(err.Error()).To(ContainSubstring("or pass --external-api-path for an external type"))
	})

	// updateResourceFromConfig resolves which recorded resource a create-webhook command refers to.
	// The specs below cover the full decision, by number of recorded resources sharing the
	// Group/Version/Kind and the flags given:
	//
	//	#  | matches | --external-api-domain | matches a record? | --external-api-path | outcome
	//	---+---------+-----------------------+-------------------+---------------------+-------------------------
	//	1  |   0     | any                   | -                 | any                 | keep res from the flags
	//	2  |   1     | unset                 | -                 | -                   | use the record
	//	3  |   1     | set                   | yes               | -                   | use the record
	//	4  |   1     | set                   | no                | set                 | create a new resource
	//	5  |   1     | set                   | no                | unset               | error (domain not found)
	//	6  |  >=2    | set                   | yes               | -                   | use the record
	//	7  |  >=2    | set                   | no                | set                 | create a new resource
	//	8  |  >=2    | set                   | no                | unset               | error (domain not found)
	//	9  |  >=2    | unset                 | -                 | -                   | error (all external)
	//	10 |  >=2    | unset                 | - (1 non-external)| -                   | use the non-external one
	//	11 |  >=2    | unset                 | - (project+core)  | -                   | use the project one
	Context("when several external resources share the same Group/Version/Kind", func() {
		const (
			pathIO    = "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
			pathK8sIO = "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
			domainIO  = "cert-manager.io"
			domainK8s = "cert-manager.k8s.io"
		)

		// storeExternal records an external resource sharing crewGroup/v1/captainKind but under
		// the given domain. Recorded resources are unique by full GVK, so distinct domains produce
		// distinct entries that collide only on Group/Version/Kind.
		storeExternal := func(domain, path string) {
			Expect(cfg.AddResource(resource.Resource{
				GVK: resource.GVK{
					Group:   crewGroup,
					Domain:  domain,
					Version: "v1",
					Kind:    captainKind,
				},
				External: true,
				Path:     path,
				Webhooks: &resource.Webhooks{},
			})).To(Succeed())
		}

		// storedByDomain re-reads the recorded resources keyed by domain, to assert entries are
		// left untouched.
		storedByDomain := func() map[string]resource.Resource {
			stored, err := cfg.GetResources()
			Expect(err).NotTo(HaveOccurred())
			byDomain := make(map[string]resource.Resource, len(stored))
			for _, s := range stored {
				byDomain[s.Domain] = s
			}
			return byDomain
		}

		It("should refuse and leave the PROJECT untouched when no domain is given", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			storeExternal(domainIO, pathIO)
			storeExternal(domainK8s, pathK8sIO)
			subCmd.options.DoDefaulting = true

			err := subCmd.InjectResource(res)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("match more than one resource"))
			Expect(err.Error()).To(ContainSubstring("pass --external-api-domain"))
			Expect(err.Error()).To(ContainSubstring(domainIO))
			Expect(err.Error()).To(ContainSubstring(domainK8s))

			// Both entries keep their path, domain and (empty) webhooks: #5931 is about the
			// webhook being silently bound to the first entry, so none may have gained one.
			byDomain := storedByDomain()
			Expect(byDomain).To(HaveLen(2))
			Expect(byDomain[domainIO].Path).To(Equal(pathIO))
			Expect(byDomain[domainIO].Domain).To(Equal(domainIO))
			Expect(byDomain[domainIO].Webhooks == nil || byDomain[domainIO].Webhooks.IsEmpty()).To(BeTrue())
			Expect(byDomain[domainK8s].Path).To(Equal(pathK8sIO))
			Expect(byDomain[domainK8s].Domain).To(Equal(domainK8s))
			Expect(byDomain[domainK8s].Webhooks == nil || byDomain[domainK8s].Webhooks.IsEmpty()).To(BeTrue())
		})

		It("should refuse regardless of the order the resources are recorded", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			// Reverse order from the previous test: nothing may depend on file order.
			storeExternal(domainK8s, pathK8sIO)
			storeExternal(domainIO, pathIO)
			subCmd.options.DoDefaulting = true

			err := subCmd.InjectResource(res)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("match more than one resource"))
		})

		It("should work on the resource named by --external-api-domain and leave the other alone", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			storeExternal(domainIO, pathIO)
			storeExternal(domainK8s, pathK8sIO)
			subCmd.options.DoDefaulting = true
			subCmd.options.ExternalAPIDomain = domainK8s

			err := subCmd.InjectResource(res)

			Expect(err).NotTo(HaveOccurred())
			Expect(res.External).To(BeTrue())
			Expect(res.Domain).To(Equal(domainK8s))
			Expect(res.Path).To(Equal(pathK8sIO))

			// The unnamed entry is untouched.
			Expect(storedByDomain()[domainIO].Path).To(Equal(pathIO))
		})

		It("should refuse when the given domain matches none of them", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			storeExternal(domainIO, pathIO)
			storeExternal(domainK8s, pathK8sIO)
			subCmd.options.DoDefaulting = true
			subCmd.options.ExternalAPIDomain = "does-not-exist.io"

			err := subCmd.InjectResource(res)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no resource matches --external-api-domain"))
			Expect(err.Error()).To(ContainSubstring("does-not-exist.io"))
			Expect(err.Error()).To(ContainSubstring(domainIO))
			Expect(err.Error()).To(ContainSubstring(domainK8s))
		})

		It("should refuse without a domain even when one entry has an empty domain", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			// An empty flag must not silently select the empty-domain entry.
			storeExternal("", pathK8sIO)
			storeExternal(domainIO, pathIO)
			subCmd.options.DoDefaulting = true

			err := subCmd.InjectResource(res)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("match more than one resource"))
			// The empty domain is rendered explicitly so the list is not "(domains: , cert-manager.io)".
			Expect(err.Error()).To(ContainSubstring("<empty>"))
			Expect(err.Error()).To(ContainSubstring(domainIO))
		})

		It("should prefer the non-external entry when no domain is given", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			// A project (non-external) entry shares the GVK with an external one.
			Expect(cfg.AddResource(resource.Resource{
				GVK:      resource.GVK{Group: crewGroup, Domain: testIO, Version: "v1", Kind: captainKind},
				Path:     "github.com/example/test/api/v1",
				Webhooks: &resource.Webhooks{},
			})).To(Succeed())
			storeExternal(domainK8s, pathK8sIO)
			subCmd.options.DoDefaulting = true

			err := subCmd.InjectResource(res)

			// Resolves to the non-external entry, not the external one, without a domain.
			Expect(err).NotTo(HaveOccurred())
			Expect(res.External).To(BeFalse())
		})

		It("should create a new external variant when the domain is new and a path is given", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			storeExternal(domainIO, pathIO)
			storeExternal(domainK8s, pathK8sIO)
			subCmd.options.DoDefaulting = true
			// A new domain plus a path fully describes a third external resource.
			subCmd.options.ExternalAPIDomain = "cert-manager.new.io"
			subCmd.options.ExternalAPIPath = "github.com/cert-manager/cert-manager/pkg/apis/new/v1"

			err := subCmd.InjectResource(res)

			Expect(err).NotTo(HaveOccurred())
			Expect(res.External).To(BeTrue())
			Expect(res.Domain).To(Equal("cert-manager.new.io"))
		})

		It("should resolve to the core entry over an external one when no domain is given", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			storeExternal(domainK8s, pathK8sIO)
			// A core entry shares the GVK with the external one.
			Expect(cfg.AddResource(resource.Resource{
				GVK:      resource.GVK{Group: crewGroup, Domain: "k8s.io", Version: "v1", Kind: captainKind},
				Core:     true,
				Path:     coreAPIPath,
				Webhooks: &resource.Webhooks{},
			})).To(Succeed())
			subCmd.options.DoDefaulting = true

			err := subCmd.InjectResource(res)

			// The lone non-external (core) entry wins; the external one is left out.
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Core).To(BeTrue())
			Expect(res.External).To(BeFalse())
		})

		It("should keep the project entry when a project and core entry collide", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			// A project entry (res's own domain) and a core entry share the GVK.
			Expect(cfg.AddResource(resource.Resource{
				GVK:      resource.GVK{Group: crewGroup, Domain: testIO, Version: "v1", Kind: captainKind},
				Path:     "github.com/example/test/api/v1",
				Webhooks: &resource.Webhooks{},
			})).To(Succeed())
			Expect(cfg.AddResource(resource.Resource{
				GVK:      resource.GVK{Group: crewGroup, Domain: "k8s.io", Version: "v1", Kind: captainKind},
				Core:     true,
				Path:     coreAPIPath,
				Webhooks: &resource.Webhooks{},
			})).To(Succeed())
			subCmd.options.DoDefaulting = true

			err := subCmd.InjectResource(res)

			// The project entry (domain == res.Domain) is kept, not the core one.
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Core).To(BeFalse())
			Expect(res.External).To(BeFalse())
		})
	})

	Context("when a single external resource shares the Group/Version/Kind", func() {
		const (
			pathIO   = "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
			domainIO = "cert-manager.io"
		)

		storeExternal := func(domain, path string) {
			Expect(cfg.AddResource(resource.Resource{
				GVK:      resource.GVK{Group: crewGroup, Domain: domain, Version: "v1", Kind: captainKind},
				External: true,
				Path:     path,
				Webhooks: &resource.Webhooks{},
			})).To(Succeed())
		}

		It("should use the recorded resource when --external-api-domain matches it", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			storeExternal(domainIO, pathIO)
			subCmd.options.DoDefaulting = true
			subCmd.options.ExternalAPIDomain = domainIO

			err := subCmd.InjectResource(res)

			Expect(err).NotTo(HaveOccurred())
			Expect(res.External).To(BeTrue())
			Expect(res.Path).To(Equal(pathIO))
			Expect(res.Domain).To(Equal(domainIO))
		})

		It("should refuse when --external-api-domain does not match and no path is given", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			storeExternal(domainIO, pathIO)
			subCmd.options.DoDefaulting = true
			subCmd.options.ExternalAPIDomain = "cert-manager.k8s.io"

			err := subCmd.InjectResource(res)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no resource matches --external-api-domain"))
			Expect(err.Error()).To(ContainSubstring(domainIO))
		})

		It("should create a new external resource when the domain is new and a path is given", func() {
			Expect(subCmd.InjectConfig(cfg)).To(Succeed())
			storeExternal(domainIO, pathIO)
			subCmd.options.DoDefaulting = true
			subCmd.options.ExternalAPIDomain = "cert-manager.k8s.io"
			subCmd.options.ExternalAPIPath = "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"

			err := subCmd.InjectResource(res)

			Expect(err).NotTo(HaveOccurred())
			Expect(res.External).To(BeTrue())
			Expect(res.Domain).To(Equal("cert-manager.k8s.io"))
		})
	})

	It("should resolve a core type without --external-api-domain", func() {
		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
		// A single recorded core type must resolve without --external-api-domain.
		Expect(cfg.AddResource(resource.Resource{
			GVK:      res.GVK,
			Core:     true,
			Path:     coreAPIPath,
			Webhooks: &resource.Webhooks{},
		})).To(Succeed())
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).NotTo(HaveOccurred())
		Expect(res.Core).To(BeTrue())
	})

	Context("isValidVersion", func() {
		BeforeEach(func() {
			res = &resource.Resource{
				GVK: resource.GVK{
					Group:   crewGroup,
					Domain:  testIO,
					Version: "v1",
					Kind:    captainKind,
				},
			}

			for _, version := range []string{"v1", "v2", "v1beta1"} {
				r := resource.Resource{
					GVK: resource.GVK{
						Group:   crewGroup,
						Domain:  testIO,
						Version: version,
						Kind:    captainKind,
					},
					API: &resource.API{CRDVersion: "v1"},
				}
				Expect(cfg.AddResource(r)).To(Succeed())
			}
		})

		It("should return true for existing version with same group and kind", func() {
			Expect(isValidVersion("v2", res, cfg)).To(BeTrue())
			Expect(isValidVersion("v1beta1", res, cfg)).To(BeTrue())
		})

		It("should return false for non-existing version", func() {
			Expect(isValidVersion("v3", res, cfg)).To(BeFalse())
		})

		It("should return false for different group", func() {
			differentRes := resource.Resource{
				GVK: resource.GVK{
					Group:   shipGroup,
					Domain:  testIO,
					Version: "v1",
					Kind:    frigateKind,
				},
				API: &resource.API{CRDVersion: "v1"},
			}
			Expect(cfg.AddResource(differentRes)).To(Succeed())

			otherRes := &resource.Resource{GVK: differentRes.GVK}
			Expect(isValidVersion("v2", otherRes, cfg)).To(BeFalse())
		})

		It("should return false for different kind", func() {
			differentRes := resource.Resource{
				GVK: resource.GVK{
					Group:   crewGroup,
					Domain:  testIO,
					Version: "v1",
					Kind:    "Pirate",
				},
				API: &resource.API{CRDVersion: "v1"},
			}
			Expect(cfg.AddResource(differentRes)).To(Succeed())

			otherRes := &resource.Resource{GVK: differentRes.GVK}
			Expect(isValidVersion("v2", otherRes, cfg)).To(BeFalse())
		})
	})
})
