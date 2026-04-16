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

var _ = Describe("DeleteAPI", func() {
	Context("InjectResource", func() {
		var (
			subcmd deleteAPISubcommand
			cfg    config.Config
		)

		BeforeEach(func() {
			cfg = cfgv3.New()
			subcmd = deleteAPISubcommand{
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

		It("should fail if resource has webhooks", func() {
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
				},
			}

			Expect(cfg.AddResource(res)).To(Succeed())

			err := subcmd.InjectResource(&res)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("webhooks are configured"))
			Expect(err.Error()).To(ContainSubstring("delete the webhooks first"))
		})

		It("should succeed if resource exists without webhooks", func() {
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
			Expect(err).NotTo(HaveOccurred())
			Expect(subcmd.resource).NotTo(BeNil())
			Expect(subcmd.resource.Kind).To(Equal("Captain"))
		})
	})
})
