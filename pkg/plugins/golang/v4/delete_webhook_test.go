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
)

var _ = Describe("DeleteWebhook", func() {
	Context("InjectResource", func() {
		var (
			subcmd deleteWebhookSubcommand
			cfg    config.Config
		)

		BeforeEach(func() {
			cfg = cfgv3.New()
			subcmd = deleteWebhookSubcommand{
				config: cfg,
			}
		})

		It("should fail if resource does not exist in config", func() {
			res := &resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
			}

			err := subcmd.InjectResource(res)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})

		It("should fail if resource has no webhooks", func() {
			res := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				API: &resource.API{
					CRDVersion: "v1",
				},
			}

			Expect(cfg.AddResource(res)).To(Succeed())

			err := subcmd.InjectResource(&res)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not have any webhooks"))
		})

		It("should default to all webhooks when no flags specified", func() {
			res := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				API: &resource.API{
					CRDVersion: "v1",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Defaulting:     true,
					Validation:     true,
				},
			}

			Expect(cfg.AddResource(res)).To(Succeed())

			err := subcmd.InjectResource(&res)
			Expect(err).NotTo(HaveOccurred())
			Expect(subcmd.resource).NotTo(BeNil())
			Expect(subcmd.doDefaulting).To(BeTrue())
			Expect(subcmd.doValidation).To(BeTrue())
			Expect(subcmd.doConversion).To(BeFalse())
		})

		It("should fail if specified webhook type doesn't exist", func() {
			res := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				API: &resource.API{
					CRDVersion: "v1",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Validation:     true,
				},
			}

			Expect(cfg.AddResource(res)).To(Succeed())
			subcmd.doDefaulting = true // Request to delete defaulting, but it doesn't exist

			err := subcmd.InjectResource(&res)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not have a defaulting webhook"))
		})

		It("should succeed when deleting specific webhook type", func() {
			res := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				API: &resource.API{
					CRDVersion: "v1",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Defaulting:     true,
					Validation:     true,
				},
			}

			Expect(cfg.AddResource(res)).To(Succeed())
			subcmd.doValidation = true // Only delete validation webhook

			err := subcmd.InjectResource(&res)
			Expect(err).NotTo(HaveOccurred())
			Expect(subcmd.resource).NotTo(BeNil())
			Expect(subcmd.doValidation).To(BeTrue())
			Expect(subcmd.doDefaulting).To(BeFalse())
		})
	})

	Context("willBeLastWebhookAfterDeletion", func() {
		var (
			subcmd deleteWebhookSubcommand
			cfg    config.Config
		)

		BeforeEach(func() {
			cfg = cfgv3.New()
			subcmd = deleteWebhookSubcommand{
				config: cfg,
			}
		})

		It("should return true when deleting the only webhook", func() {
			res := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Defaulting:     true,
				},
			}

			Expect(cfg.AddResource(res)).To(Succeed())
			subcmd.resource = &res
			subcmd.doDefaulting = true

			isLast, err := subcmd.willBeLastWebhookAfterDeletion()
			Expect(err).NotTo(HaveOccurred())
			Expect(isLast).To(BeTrue())
		})

		It("should return false when deleting one webhook type but others remain", func() {
			res := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Defaulting:     true,
					Validation:     true,
				},
			}

			Expect(cfg.AddResource(res)).To(Succeed())
			subcmd.resource = &res
			subcmd.doDefaulting = true // Only deleting defaulting, validation remains

			isLast, err := subcmd.willBeLastWebhookAfterDeletion()
			Expect(err).NotTo(HaveOccurred())
			Expect(isLast).To(BeFalse())
		})

		It("should return false when multiple resources have webhooks", func() {
			res1 := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Defaulting:     true,
				},
			}

			res2 := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "FirstMate",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Validation:     true,
				},
			}

			Expect(cfg.AddResource(res1)).To(Succeed())
			Expect(cfg.AddResource(res2)).To(Succeed())
			subcmd.resource = &res1
			subcmd.doDefaulting = true

			isLast, err := subcmd.willBeLastWebhookAfterDeletion()
			Expect(err).NotTo(HaveOccurred())
			Expect(isLast).To(BeFalse())
		})

		It("should return true when deleting all webhook types and no other resources have webhooks", func() {
			res1 := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Defaulting:     true,
					Validation:     true,
				},
			}

			res2 := resource.Resource{
				GVK: resource.GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "FirstMate",
				},
				API: &resource.API{
					CRDVersion: "v1",
				},
			}

			Expect(cfg.AddResource(res1)).To(Succeed())
			Expect(cfg.AddResource(res2)).To(Succeed())
			subcmd.resource = &res1
			subcmd.doDefaulting = true
			subcmd.doValidation = true

			isLast, err := subcmd.willBeLastWebhookAfterDeletion()
			Expect(err).NotTo(HaveOccurred())
			Expect(isLast).To(BeTrue())
		})
	})
})
