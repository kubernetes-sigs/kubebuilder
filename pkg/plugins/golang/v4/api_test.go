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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

const (
	crewGroup       = "crew"
	testIO          = "test.io"
	testCommandName = "kubebuilder"
	captainKind     = "Captain"
	captains        = "captains"
	shipGroup       = "ship"
	frigateKind     = "Frigate"

	externalAPIModuleWithVersion = "github.com/external/api@v1.0.0"
	relativeAPIPath              = "api/v1"
)

var _ = Describe("createAPISubcommand", func() {
	var (
		subCmd *createAPISubcommand
		cfg    config.Config
		res    *resource.Resource
	)

	BeforeEach(func() {
		subCmd = &createAPISubcommand{}
		cfg = cfgv3.New()
		_ = cfg.SetRepository("github.com/example/test")

		subCmd.options = &goPlugin.Options{}
		subCmd.resourceFlag = &pflag.Flag{Changed: true}
		subCmd.controllerFlag = &pflag.Flag{Changed: true}

		res = &resource.Resource{
			GVK: resource.GVK{
				Group:   crewGroup,
				Domain:  testIO,
				Version: "v1",
				Kind:    captainKind,
			},
			Plural:   captains,
			API:      &resource.API{},
			Webhooks: &resource.Webhooks{},
		}

		Expect(subCmd.InjectConfig(cfg)).To(Succeed())
	})

	Context("UpdateMetadata", func() {
		It("should provide concise API examples", func() {
			meta := &plugin.SubcommandMetadata{}

			subCmd.UpdateMetadata(plugin.CLIMetadata{CommandName: testCommandName}, meta)

			Expect(meta.Examples).To(ContainSubstring("kubebuilder create api --group crew --version v1 --kind Captain"))
			Expect(meta.Examples).To(ContainSubstring("--namespaced=false --controller=false"))
			Expect(meta.Examples).To(ContainSubstring("--external-api-path"))
			Expect(meta.Examples).NotTo(ContainSubstring("nano "))
		})
	})

	It("should reject external API options when creating API in project", func() {
		subCmd.options.DoAPI = true
		subCmd.options.ExternalAPIPath = "github.com/external/api"

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot use '--external-api-path'"))
	})

	It("should reject --ssa when not creating an API resource (--resource=false)", func() {
		subCmd.options.SSA = true
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"'--ssa' can only be used when creating an API resource ('--resource=true')"))
	})

	It("should allow --ssa when creating an API resource (--resource=true)", func() {
		subCmd.options.SSA = true
		subCmd.options.DoAPI = true
		subCmd.options.DoController = true

		Expect(subCmd.InjectResource(res)).To(Succeed())
		Expect(res.API.SSA).To(BeTrue())
	})

	It("should require external-api-path when using external-api-module", func() {
		subCmd.options.DoAPI = false
		subCmd.options.ExternalAPIModule = externalAPIModuleWithVersion
		subCmd.options.ExternalAPIPath = ""

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("requires '--external-api-path'"))
	})

	It("should reject external-api-path with module version", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = externalAPIModuleWithVersion

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("version specifiers belong in the module field"))
	})

	It("should reject bare relative external-api-path", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = relativeAPIPath

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("must be a fully-qualified Go import path"))
	})

	It("should reject bare domain external-api-path", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = "example.com"

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("must include a package sub-path"))
	})

	It("should reject leading-dot pseudo-domain external-api-path", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = ".com/org/repo/api/v1"

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("must be a fully-qualified Go import path"))
	})

	It("should reject malformed external-api-path", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = "a//b"

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("malformed import path"))
		Expect(err.Error()).To(ContainSubstring("double slash"))
	})

	It("should allow adding a controller to existing external resource without re-providing --external-api-path", func() {
		const externalPath = "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
		existingExternal := *res
		existingExternal.External = true
		existingExternal.Path = externalPath
		Expect(cfg.AddResource(existingExternal)).To(Succeed())

		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = ""
		res.External = true
		res.Path = externalPath

		err := subCmd.InjectResource(res)

		Expect(err).NotTo(HaveOccurred())
	})

	It("should find existing external resource when stored Domain differs from project domain", func() {
		const externalPath = "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
		const externalDomain = "cert-manager.io"

		// Simulate: resource originally scaffolded with --external-api-domain=cert-manager.io,
		// so PROJECT stores Domain="cert-manager.io". A fresh CLI invocation without
		// --external-api-domain runs resolveDomain up in cmd_helpers before the GVK reaches
		// InjectResource, so by this point res.Domain has already been reconciled to the
		// stored external domain — mirror that here.
		existingExternal := *res
		existingExternal.External = true
		existingExternal.Path = externalPath
		existingExternal.Domain = externalDomain
		Expect(cfg.AddResource(existingExternal)).To(Succeed())

		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = ""
		res.Domain = externalDomain

		err := subCmd.InjectResource(res)

		Expect(err).NotTo(HaveOccurred())
		Expect(res.External).To(BeTrue())
		Expect(res.Path).To(Equal(externalPath))
	})

	It("should return an actionable error for --resource=false on old project with relative external path", func() {
		// Simulate an old PROJECT file that stored a relative path instead of a Go import path
		existingExternal := *res
		existingExternal.External = true
		existingExternal.Path = relativeAPIPath
		Expect(cfg.AddResource(existingExternal)).To(Succeed())

		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = ""

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid Path"))
		Expect(err.Error()).To(ContainSubstring("must be a fully-qualified Go import path"))
		Expect(err.Error()).To(ContainSubstring("github.com/org/repo/api/v1"))
	})

	It("should prevent duplicate API without force flag", func() {
		subCmd.options.DoAPI = true
		subCmd.options.DoController = true

		resWithAPI := *res
		resWithAPI.API = &resource.API{CRDVersion: "v1"}
		Expect(cfg.AddResource(resWithAPI)).To(Succeed())

		subCmd.force = false
		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("API resource already exists"))
	})

	It("should allow duplicate API with force flag", func() {
		subCmd.options.DoAPI = true
		subCmd.options.DoController = true

		resWithAPI := *res
		resWithAPI.API = &resource.API{CRDVersion: "v1"}
		Expect(cfg.AddResource(resWithAPI)).To(Succeed())

		subCmd.force = true
		err := subCmd.InjectResource(res)

		Expect(err).NotTo(HaveOccurred())
	})

	It("should prevent multiple groups in single-group project", func() {
		subCmd.options.DoAPI = true
		subCmd.options.DoController = true

		firstRes := resource.Resource{
			GVK: resource.GVK{
				Group:   shipGroup,
				Domain:  testIO,
				Version: "v1",
				Kind:    frigateKind,
			},
			Plural: "frigates",
			API:    &resource.API{CRDVersion: "v1"},
		}
		Expect(cfg.AddResource(firstRes)).To(Succeed())

		res.Group = crewGroup
		res.Plural = captains

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("multiple groups are not allowed"))
	})

	It("should allow multiple groups when multigroup is enabled", func() {
		subCmd.options.DoAPI = true
		subCmd.options.DoController = true

		Expect(cfg.SetMultiGroup()).To(Succeed())

		firstRes := resource.Resource{
			GVK: resource.GVK{
				Group:   shipGroup,
				Domain:  testIO,
				Version: "v1",
				Kind:    frigateKind,
			},
			Plural: "frigates",
			API:    &resource.API{CRDVersion: "v1"},
		}
		Expect(cfg.AddResource(firstRes)).To(Succeed())

		res.Group = crewGroup

		Expect(subCmd.InjectResource(res)).To(Succeed())
	})
})

var _ = Describe("validateController", func() {
	const (
		captainCtrl   = "captain"
		captainBackup = "captain-backup"
	)

	var (
		cfg config.Config
		gvk resource.GVK
	)

	BeforeEach(func() {
		cfg = cfgv3.New()
		Expect(cfg.SetRepository("test")).To(Succeed())
		gvk = resource.GVK{Group: crewGroup, Domain: testIO, Version: "v1", Kind: captainKind}
	})

	// stored puts a resource carrying the given controllers into the config, the way a
	// PROJECT file on disk would present it.
	stored := func(controllers *resource.Controllers, legacy bool) {
		res := resource.Resource{GVK: gvk, Plural: captains, Controllers: controllers}
		res.Controller = legacy //nolint:staticcheck // simulating an unmigrated PROJECT file
		Expect(cfg.AddResource(res)).To(Succeed())
	}

	subcommand := func(name string, doAPI bool) *createAPISubcommand {
		return &createAPISubcommand{
			config:   cfg,
			resource: &resource.Resource{GVK: gvk, Plural: captains},
			options:  &goPlugin.Options{DoController: true, ControllerName: name, DoAPI: doAPI},
		}
	}

	It("should skip validation when no controller is requested", func() {
		stored(&resource.Controllers{{Name: captainCtrl}}, false)
		p := subcommand("", false)
		p.options.DoController = false

		Expect(p.validateController()).To(Succeed())
	})

	It("should reject a name that is not a valid DNS label", func() {
		Expect(subcommand("Not_A_Label", false).validateController()).NotTo(Succeed())
	})

	It("should accept any name when the resource does not exist yet", func() {
		Expect(subcommand(captainBackup, false).validateController()).To(Succeed())
	})

	It("should accept a new name on a resource that already has controllers", func() {
		stored(&resource.Controllers{{Name: captainCtrl}}, false)

		Expect(subcommand(captainBackup, false).validateController()).To(Succeed())
	})

	It("should reject a name already recorded", func() {
		stored(&resource.Controllers{{Name: captainCtrl}}, false)

		Expect(subcommand(captainCtrl, false).validateController()).NotTo(Succeed())
	})

	It("should reject a name already recorded through the legacy field", func() {
		stored(nil, true)

		Expect(subcommand(captainCtrl, false).validateController()).NotTo(Succeed())
	})

	It("should reject a name generating an existing reconciler", func() {
		stored(&resource.Controllers{{Name: captainBackup}}, false)

		Expect(subcommand("captain--backup", false).validateController()).NotTo(Succeed())
	})

	It("should allow re-scaffolding the single default controller without a name", func() {
		stored(&resource.Controllers{{Name: captainCtrl}}, false)

		Expect(subcommand("", false).validateController()).To(Succeed())
	})

	It("should require a name when the resource has several controllers", func() {
		stored(&resource.Controllers{{Name: captainCtrl}, {Name: captainBackup}}, false)

		Expect(subcommand("", false).validateController()).NotTo(Succeed())
	})

	It("should require a name when the only controller is not the default", func() {
		stored(&resource.Controllers{{Name: captainBackup}}, false)

		Expect(subcommand("", false).validateController()).NotTo(Succeed())
	})

	It("should allow omitting the name while recreating the API", func() {
		stored(&resource.Controllers{{Name: captainCtrl}, {Name: captainBackup}}, false)

		Expect(subcommand("", true).validateController()).To(Succeed())
	})
})

var _ = Describe("validateController across resources", func() {
	const (
		admiralCtrl   = "admiral"
		admiralKind   = "Admiral"
		admiralPlural = "admirals"
	)

	var (
		cfg        config.Config
		captainGVK resource.GVK
	)

	BeforeEach(func() {
		cfg = cfgv3.New()
		Expect(cfg.SetRepository("test")).To(Succeed())
		captainGVK = resource.GVK{Group: crewGroup, Domain: testIO, Version: "v1", Kind: captainKind}

		// Admiral already owns AdmiralReconciler under its default controller name.
		Expect(cfg.AddResource(resource.Resource{
			GVK:         resource.GVK{Group: crewGroup, Domain: testIO, Version: "v1", Kind: admiralKind},
			Plural:      admiralPlural,
			Controllers: &resource.Controllers{{Name: admiralCtrl}},
		})).To(Succeed())
	})

	subcommand := func(name string) *createAPISubcommand {
		return &createAPISubcommand{
			config:   cfg,
			resource: &resource.Resource{GVK: captainGVK, Plural: captains},
			options:  &goPlugin.Options{DoController: true, ControllerName: name},
		}
	}

	It("should reject a name generating another resource's reconciler", func() {
		err := subcommand(admiralCtrl).validateController()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("AdmiralReconciler"))
	})

	It("should accept a name that does not collide", func() {
		Expect(subcommand("captain-backup").validateController()).To(Succeed())
	})

	// A second version of an existing kind resolves to the same default controller name,
	// so it collides even though no --controller-name was given.
	It("should reject the default name when another version of the kind already has it", func() {
		p := subcommand("")
		p.resource = &resource.Resource{
			GVK:    resource.GVK{Group: crewGroup, Domain: testIO, Version: "v2", Kind: admiralKind},
			Plural: admiralPlural,
		}

		err := p.validateController()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("AdmiralReconciler"))
	})

	It("should accept a second version of a kind that names its controller", func() {
		p := subcommand("admiral-v2")
		p.resource = &resource.Resource{
			GVK:    resource.GVK{Group: crewGroup, Domain: testIO, Version: "v2", Kind: admiralKind},
			Plural: admiralPlural,
		}

		Expect(p.validateController()).To(Succeed())
	})

	It("should allow the same name in another group of a multigroup project", func() {
		Expect(cfg.SetMultiGroup()).To(Succeed())
		p := subcommand(admiralCtrl)
		p.resource.Group = shipGroup

		Expect(p.validateController()).To(Succeed())
	})

	It("should still reject it within the same group of a multigroup project", func() {
		Expect(cfg.SetMultiGroup()).To(Succeed())

		Expect(subcommand(admiralCtrl).validateController()).NotTo(Succeed())
	})
})

var _ = Describe("default controller name", func() {
	// Migrate derives the default name with strings.ToLower(Kind) and Controller.Validate
	// requires a DNS-1035 label. GVK.Validate applies the same rule to the kind, so any
	// resource kubebuilder accepts can always be migrated.
	DescribeTable("should be valid for every kind GVK.Validate accepts",
		func(kind string) {
			gvk := resource.GVK{Group: crewGroup, Domain: testIO, Version: "v1", Kind: kind}
			Expect(gvk.Validate()).To(Succeed())

			name := resource.DefaultControllerName(kind)
			Expect(resource.Controller{Name: name}.Validate()).To(Succeed())
		},
		Entry("a short kind", captainKind),
		Entry("a kind with digits", "Captain2"),
		Entry("a kind at the 63 character limit", "C"+strings.Repeat("a", 62)),
	)
})
