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
		// GVK domain stays as testIO because the stored resource carries the project domain,
		// so that is what the single tracked group/version/kind match lends back.
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

		// Resource was originally scaffolded with --domain=cert-manager.io, so PROJECT stores
		// Domain="cert-manager.io". A fresh CLI invocation without --domain has the single
		// tracked group/version/kind match lend its domain up in cmd_helpers before the GVK
		// reaches InjectResource, so by this point res.Domain is already the stored external
		// domain — mirror that here.
		storedRes := *res
		storedRes.External = true
		storedRes.Path = externalPath
		storedRes.Domain = externalDomain
		Expect(cfg.AddResource(storedRes)).To(Succeed())

		// Simulate user running create webhook without any other external flags.
		subCmd.options.DoDefaulting = true
		res.Domain = externalDomain

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

	// The stored resource is found by its full GVK, so the deprecated alias has to reach the
	// resource before the lookup runs; reconciling it afterwards would miss the entry and leave
	// the external path unset.
	It("should find the existing external resource when only the deprecated alias names its domain", func() {
		const externalPath = "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
		const externalDomain = "cert-manager.io"

		Expect(subCmd.InjectConfig(cfg)).To(Succeed())

		storedRes := *res
		storedRes.External = true
		storedRes.Path = externalPath
		storedRes.Domain = externalDomain
		Expect(cfg.AddResource(storedRes)).To(Succeed())

		// The ambiguity left the domain unresolved, so only --external-api-domain names it.
		subCmd.options.DoDefaulting = true
		subCmd.options.ExternalAPIDomain = externalDomain
		res.Domain = ""

		Expect(subCmd.InjectResource(res)).To(Succeed())
		Expect(res.Domain).To(Equal(externalDomain))
		Expect(res.External).To(BeTrue())
		Expect(res.Path).To(Equal(externalPath))
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
