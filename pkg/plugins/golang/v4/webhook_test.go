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
				Group:   "crew",
				Domain:  "test.io",
				Version: "v1",
				Kind:    "Captain",
			},
			Plural:   "captains",
			Webhooks: &resource.Webhooks{},
		}
	})

	It("should reject defaulting-path without --defaulting", func() {
		subCmd.options.DefaultingPath = "/custom-path"
		subCmd.options.DoDefaulting = false

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--defaulting-path can only be used with --defaulting"))
	})

	It("should reject validation-path without --programmatic-validation", func() {
		subCmd.options.ValidationPath = "/custom-path"
		subCmd.options.DoValidation = false

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--validation-path can only be used with --programmatic-validation"))
	})

	It("should require external-api-path when using external-api-module", func() {
		subCmd.options.ExternalAPIModule = "github.com/external/api@v1.0.0"
		subCmd.options.ExternalAPIPath = ""
		subCmd.options.DoDefaulting = true

		err := subCmd.InjectResource(res)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("requires '--external-api-path'"))
	})

	Context("isValidVersion", func() {
		BeforeEach(func() {
			res = &resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Domain:  "test.io",
					Version: "v1",
					Kind:    "Captain",
				},
			}

			for _, version := range []string{"v1", "v2", "v1beta1"} {
				r := resource.Resource{
					GVK: resource.GVK{
						Group:   "crew",
						Domain:  "test.io",
						Version: version,
						Kind:    "Captain",
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
					Group:   "ship",
					Domain:  "test.io",
					Version: "v1",
					Kind:    "Frigate",
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
					Group:   "crew",
					Domain:  "test.io",
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
