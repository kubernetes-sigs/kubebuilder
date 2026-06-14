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
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	goPlugin "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

const (
	crewGroup   = "crew"
	testIO      = "test.io"
	captainKind = "Captain"
	captains    = "captains"
	shipGroup   = "ship"
	frigateKind = "Frigate"

	externalAPIModuleWithVersion = "github.com/external/api@v1.0.0"
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

	It("should reject external API options when creating API in project", func() {
		subCmd.options.DoAPI = true
		subCmd.options.ExternalAPIPath = "github.com/external/api"

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot use '--external-api-path'"))
	})

	It("should require external-api-path when using external-api-module", func() {
		subCmd.options.DoAPI = false
		subCmd.options.ExternalAPIModule = externalAPIModuleWithVersion
		subCmd.options.ExternalAPIPath = ""

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("requires '--external-api-path'"))
	})

	It("should reject external-api-path with module version guidance", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = externalAPIModuleWithVersion

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid '--external-api-path'"))
		Expect(err.Error()).To(ContainSubstring("must be a valid Go package import path"))
		Expect(err.Error()).To(ContainSubstring("Use '--external-api-module'"))
	})

	It("should reject bare relative external-api-path", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = "api/v1"

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("external API import path"))
		Expect(err.Error()).To(ContainSubstring("must be domain-qualified"))
	})

	It("should reject leading-dot pseudo-domain external-api-path", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = ".com/org/repo/api/v1"

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid '--external-api-path'"))
		Expect(err.Error()).To(ContainSubstring("must be domain-qualified"))
	})

	It("should reject malformed external-api-path", func() {
		subCmd.options.DoAPI = false
		subCmd.options.DoController = true
		subCmd.options.ExternalAPIPath = "a//b"

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid '--external-api-path'"))
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
