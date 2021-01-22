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

package v3alpha

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

func TestConfigV3Alpha(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config V3-Alpha Suite")
}

var _ = Describe("cfg", func() {
	const (
		domain = "my.domain"
		repo   = "myrepo"
		name   = "ProjectName"
		layout = "go.kubebuilder.io/v2"

		otherDomain = "other.domain"
		otherRepo   = "otherrepo"
		otherName   = "OtherProjectName"
		otherLayout = "go.kubebuilder.io/v3-alpha"
	)

	var c cfg

	BeforeEach(func() {
		c = cfg{
			Version:    Version,
			Domain:     domain,
			Repository: repo,
			Name:       name,
			Layout:     layout,
		}
	})

	Context("Version", func() {
		It("GetVersion should return version 3-alpha", func() {
			Expect(c.GetVersion().Compare(Version)).To(Equal(0))
		})
	})

	Context("Domain", func() {
		It("GetDomain should return the domain", func() {
			Expect(c.GetDomain()).To(Equal(domain))
		})

		It("SetDomain should set the domain", func() {
			Expect(c.SetDomain(otherDomain)).To(Succeed())
			Expect(c.Domain).To(Equal(otherDomain))
		})
	})

	Context("Repository", func() {
		It("GetRepository should return the repository", func() {
			Expect(c.GetRepository()).To(Equal(repo))
		})

		It("SetRepository should set the repository", func() {
			Expect(c.SetRepository(otherRepo)).To(Succeed())
			Expect(c.Repository).To(Equal(otherRepo))
		})
	})

	Context("Name", func() {
		It("GetProjectName should return the name", func() {
			Expect(c.GetProjectName()).To(Equal(name))
		})

		It("SetProjectName should set the name", func() {
			Expect(c.SetProjectName(otherName)).To(Succeed())
			Expect(c.Name).To(Equal(otherName))
		})
	})

	Context("Layout", func() {
		It("GetLayout should return the layout", func() {
			Expect(c.GetLayout()).To(Equal(layout))
		})

		It("SetLayout should set the layout", func() {
			Expect(c.SetLayout(otherLayout)).To(Succeed())
			Expect(c.Layout).To(Equal(otherLayout))
		})
	})

	Context("Multi group", func() {
		It("IsMultiGroup should return false if not set", func() {
			Expect(c.IsMultiGroup()).To(BeFalse())
		})

		It("IsMultiGroup should return true if set", func() {
			c.MultiGroup = true
			Expect(c.IsMultiGroup()).To(BeTrue())
		})

		It("SetMultiGroup should enable multi-group support", func() {
			Expect(c.SetMultiGroup()).To(Succeed())
			Expect(c.MultiGroup).To(BeTrue())
		})

		It("ClearMultiGroup should disable multi-group support", func() {
			c.MultiGroup = true
			Expect(c.ClearMultiGroup()).To(Succeed())
			Expect(c.MultiGroup).To(BeFalse())
		})
	})

	Context("Component config", func() {
		It("IsComponentConfig should return false if not set", func() {
			Expect(c.IsComponentConfig()).To(BeFalse())
		})

		It("IsComponentConfig should return true if set", func() {
			c.ComponentConfig = true
			Expect(c.IsComponentConfig()).To(BeTrue())
		})

		It("SetComponentConfig should fail to enable component config support", func() {
			Expect(c.SetComponentConfig()).To(Succeed())
			Expect(c.ComponentConfig).To(BeTrue())
		})

		It("ClearComponentConfig should fail to disable component config support", func() {
			c.ComponentConfig = false
			Expect(c.ClearComponentConfig()).To(Succeed())
			Expect(c.ComponentConfig).To(BeFalse())
		})
	})

	Context("Resources", func() {
		var res = resource.Resource{
			GVK: resource.GVK{
				Group:   "group",
				Version: "v1",
				Kind:    "Kind",
			},
			API: &resource.API{
				CRDVersion: "v1",
				Namespaced: true,
			},
			Controller: true,
			Webhooks: &resource.Webhooks{
				WebhookVersion: "v1",
				Defaulting:     true,
				Validation:     true,
				Conversion:     true,
			},
		}

		DescribeTable("ResourcesLength should return the number of resources",
			func(n int) {
				for i := 0; i < n; i++ {
					c.Resources = append(c.Resources, res)
				}
				Expect(c.ResourcesLength()).To(Equal(n))
			},
			Entry("for no resources", 0),
			Entry("for one resource", 1),
			Entry("for several resources", 3),
		)

		It("HasResource should return false for a non-existent resource", func() {
			Expect(c.HasResource(res.GVK)).To(BeFalse())
		})

		It("HasResource should return true for an existent resource", func() {
			c.Resources = append(c.Resources, res)
			Expect(c.HasResource(res.GVK)).To(BeTrue())
		})

		It("GetResource should fail for a non-existent resource", func() {
			_, err := c.GetResource(res.GVK)
			Expect(err).To(HaveOccurred())
		})

		It("GetResource should return an existent resource", func() {
			c.Resources = append(c.Resources, res)
			r, err := c.GetResource(res.GVK)
			Expect(err).NotTo(HaveOccurred())
			Expect(r.GVK.IsEqualTo(res.GVK)).To(BeTrue())
			Expect(r.API).NotTo(BeNil())
			Expect(r.API.CRDVersion).To(Equal(res.API.CRDVersion))
			Expect(r.Webhooks).NotTo(BeNil())
			Expect(r.Webhooks.WebhookVersion).To(Equal(res.Webhooks.WebhookVersion))
		})

		It("GetResources should return a slice of the tracked resources", func() {
			c.Resources = append(c.Resources, res, res, res)
			resources, err := c.GetResources()
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(Equal([]resource.Resource{res, res, res}))
		})

		// Auxiliary function for AddResource and UpdateResource tests
		checkResource := func(result, expected resource.Resource) {
			Expect(result.GVK.IsEqualTo(expected.GVK)).To(BeTrue())
			Expect(result.API).NotTo(BeNil())
			Expect(result.API.CRDVersion).To(Equal(expected.API.CRDVersion))
			Expect(result.API.Namespaced).To(BeFalse())
			Expect(result.Controller).To(BeFalse())
			Expect(result.Webhooks).NotTo(BeNil())
			Expect(result.Webhooks.WebhookVersion).To(Equal(expected.Webhooks.WebhookVersion))
			Expect(result.Webhooks.Defaulting).To(BeFalse())
			Expect(result.Webhooks.Validation).To(BeFalse())
			Expect(result.Webhooks.Conversion).To(BeFalse())
		}

		It("AddResource should add the provided resource if non-existent", func() {
			l := len(c.Resources)
			Expect(c.AddResource(res)).To(Succeed())
			Expect(len(c.Resources)).To(Equal(l + 1))

			checkResource(c.Resources[0], res)
		})

		It("AddResource should do nothing if the resource already exists", func() {
			c.Resources = append(c.Resources, res)
			l := len(c.Resources)
			Expect(c.AddResource(res)).To(Succeed())
			Expect(len(c.Resources)).To(Equal(l))
		})

		It("UpdateResource should add the provided resource if non-existent", func() {
			l := len(c.Resources)
			Expect(c.UpdateResource(res)).To(Succeed())
			Expect(len(c.Resources)).To(Equal(l + 1))

			checkResource(c.Resources[0], res)
		})

		It("UpdateResource should update it if the resource already exists", func() {
			c.Resources = append(c.Resources, resource.Resource{
				GVK: resource.GVK{
					Group:   "group",
					Version: "v1",
					Kind:    "Kind",
				},
			})
			l := len(c.Resources)
			Expect(c.Resources[0].GVK.IsEqualTo(res.GVK)).To(BeTrue())
			Expect(c.Resources[0].API).To(BeNil())
			Expect(c.Resources[0].Controller).To(BeFalse())
			Expect(c.Resources[0].Webhooks).To(BeNil())

			Expect(c.UpdateResource(res)).To(Succeed())
			Expect(len(c.Resources)).To(Equal(l))

			r := c.Resources[0]
			Expect(r.GVK.IsEqualTo(res.GVK)).To(BeTrue())
			Expect(r.API).NotTo(BeNil())
			Expect(r.API.CRDVersion).To(Equal(res.API.CRDVersion))
			Expect(r.API.Namespaced).To(BeFalse())
			Expect(r.Controller).To(BeFalse())
			Expect(r.Webhooks).NotTo(BeNil())
			Expect(r.Webhooks.WebhookVersion).To(Equal(res.Webhooks.WebhookVersion))
			Expect(r.Webhooks.Defaulting).To(BeFalse())
			Expect(r.Webhooks.Validation).To(BeFalse())
			Expect(r.Webhooks.Conversion).To(BeFalse())
		})

		It("HasGroup should return false with no tracked resources", func() {
			Expect(c.HasGroup(res.Group)).To(BeFalse())
		})

		It("HasGroup should return true with tracked resources in the same group", func() {
			c.Resources = append(c.Resources, res)
			Expect(c.HasGroup(res.Group)).To(BeTrue())
		})

		It("HasGroup should return false with tracked resources in other group", func() {
			c.Resources = append(c.Resources, res)
			Expect(c.HasGroup("other-group")).To(BeFalse())
		})

		It("IsCRDVersionCompatible should return true with no tracked resources", func() {
			Expect(c.IsCRDVersionCompatible("v1beta1")).To(BeTrue())
			Expect(c.IsCRDVersionCompatible("v1")).To(BeTrue())
		})

		It("IsCRDVersionCompatible should return true only for matching CRD versions of tracked resources", func() {
			c.Resources = append(c.Resources, resource.Resource{
				GVK: resource.GVK{
					Group:   res.Group,
					Version: res.Version,
					Kind:    res.Kind,
				},
				API: &resource.API{CRDVersion: "v1beta1"},
			})
			Expect(c.IsCRDVersionCompatible("v1beta1")).To(BeTrue())
			Expect(c.IsCRDVersionCompatible("v1")).To(BeFalse())
			Expect(c.IsCRDVersionCompatible("v2")).To(BeFalse())
		})

		It("IsWebhookVersionCompatible should return true with no tracked resources", func() {
			Expect(c.IsWebhookVersionCompatible("v1beta1")).To(BeTrue())
			Expect(c.IsWebhookVersionCompatible("v1")).To(BeTrue())
		})

		It("IsWebhookVersionCompatible should return true only for matching webhook versions of tracked resources", func() {
			c.Resources = append(c.Resources, resource.Resource{
				GVK: resource.GVK{
					Group:   res.Group,
					Version: res.Version,
					Kind:    res.Kind,
				},
				Webhooks: &resource.Webhooks{WebhookVersion: "v1beta1"},
			})
			Expect(c.IsWebhookVersionCompatible("v1beta1")).To(BeTrue())
			Expect(c.IsWebhookVersionCompatible("v1")).To(BeFalse())
			Expect(c.IsWebhookVersionCompatible("v2")).To(BeFalse())
		})
	})

	Context("Plugins", func() {
		// Test plugin config. Don't want to export this config, but need it to
		// be accessible by test.
		type PluginConfig struct {
			Data1 string `json:"data-1"`
			Data2 string `json:"data-2,omitempty"`
		}

		const (
			key = "plugin-x"
		)

		var (
			c0 = cfg{
				Version:    Version,
				Domain:     domain,
				Repository: repo,
				Name:       name,
				Layout:     layout,
			}
			c1 = cfg{
				Version:    Version,
				Domain:     domain,
				Repository: repo,
				Name:       name,
				Layout:     layout,
				Plugins: PluginConfigs{
					"plugin-x": map[string]interface{}{
						"data-1": "",
					},
				},
			}
			c2 = cfg{
				Version:    Version,
				Domain:     domain,
				Repository: repo,
				Name:       name,
				Layout:     layout,
				Plugins: PluginConfigs{
					"plugin-x": map[string]interface{}{
						"data-1": "plugin value 1",
						"data-2": "plugin value 2",
					},
				},
			}
			pluginConfig = PluginConfig{
				Data1: "plugin value 1",
				Data2: "plugin value 2",
			}
		)

		DescribeTable("DecodePluginConfig should retrieve the plugin data correctly",
			func(inputConfig cfg, expectedPluginConfig PluginConfig) {
				var pluginConfig PluginConfig
				Expect(inputConfig.DecodePluginConfig(key, &pluginConfig)).To(Succeed())
				Expect(pluginConfig).To(Equal(expectedPluginConfig))
			},
			Entry("for no plugin config object", c0, nil),
			Entry("for an empty plugin config object", c1, PluginConfig{}),
			Entry("for a full plugin config object", c2, pluginConfig),
			// TODO (coverage): add cases where yaml.Marshal returns an error
			// TODO (coverage): add cases where yaml.Unmarshal returns an error
		)

		DescribeTable("EncodePluginConfig should encode the plugin data correctly",
			func(pluginConfig PluginConfig, expectedConfig cfg) {
				Expect(c.EncodePluginConfig(key, pluginConfig)).To(Succeed())
				Expect(c).To(Equal(expectedConfig))
			},
			Entry("for an empty plugin config object", PluginConfig{}, c1),
			Entry("for a full plugin config object", pluginConfig, c2),
			// TODO (coverage): add cases where yaml.Marshal returns an error
			// TODO (coverage): add cases where yaml.Unmarshal returns an error
		)
	})

	Context("Persistence", func() {
		var (
			// BeforeEach is called after the entries are evaluated, and therefore, c is not available
			c1 = cfg{
				Version:    Version,
				Domain:     domain,
				Repository: repo,
				Name:       name,
				Layout:     layout,
			}
			c2 = cfg{
				Version:         Version,
				Domain:          otherDomain,
				Repository:      otherRepo,
				Name:            otherName,
				Layout:          otherLayout,
				MultiGroup:      true,
				ComponentConfig: true,
				Resources: []resource.Resource{
					{
						GVK: resource.GVK{
							Group:   "group",
							Version: "v1",
							Kind:    "Kind",
						},
					},
					{
						GVK: resource.GVK{
							Group:   "group",
							Version: "v1",
							Kind:    "Kind2",
						},
						API:      &resource.API{CRDVersion: "v1"},
						Webhooks: &resource.Webhooks{WebhookVersion: "v1"},
					},
					{
						GVK: resource.GVK{
							Group:   "group",
							Version: "v1-beta",
							Kind:    "Kind",
						},
						API:      &resource.API{},
						Webhooks: &resource.Webhooks{},
					},
					{
						GVK: resource.GVK{
							Group:   "group2",
							Version: "v1",
							Kind:    "Kind",
						},
					},
				},
				Plugins: PluginConfigs{
					"plugin-x": map[string]interface{}{
						"data-1": "single plugin datum",
					},
					"plugin-y/v1": map[string]interface{}{
						"data-1": "plugin value 1",
						"data-2": "plugin value 2",
						"data-3": []string{"plugin value 3", "plugin value 4"},
					},
				},
			}
			// TODO: include cases with Plural, Path, API.namespaced, Controller, Webhooks.Defaulting,
			//       Webhooks.Validation and Webhooks.Conversion when added
			s1 = `domain: my.domain
layout: go.kubebuilder.io/v2
projectName: ProjectName
repo: myrepo
version: 3-alpha
`
			s2 = `componentConfig: true
domain: other.domain
layout: go.kubebuilder.io/v3-alpha
multigroup: true
plugins:
  plugin-x:
    data-1: single plugin datum
  plugin-y/v1:
    data-1: plugin value 1
    data-2: plugin value 2
    data-3:
    - plugin value 3
    - plugin value 4
projectName: OtherProjectName
repo: otherrepo
resources:
- group: group
  kind: Kind
  version: v1
- api:
    crdVersion: v1
  group: group
  kind: Kind2
  version: v1
  webhooks:
    webhookVersion: v1
- group: group
  kind: Kind
  version: v1-beta
- group: group2
  kind: Kind
  version: v1
version: 3-alpha
`
		)

		DescribeTable("Marshal should succeed",
			func(c cfg, content string) {
				b, err := c.Marshal()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal(content))
			},
			Entry("for a basic configuration", c1, s1),
			Entry("for a full configuration", c2, s2),
		)

		DescribeTable("Marshal should fail",
			func(c cfg) {
				_, err := c.Marshal()
				Expect(err).To(HaveOccurred())
			},
			// TODO (coverage): add cases where yaml.Marshal returns an error
		)

		DescribeTable("Unmarshal should succeed",
			func(content string, c cfg) {
				var unmarshalled cfg
				Expect(unmarshalled.Unmarshal([]byte(content))).To(Succeed())
				Expect(unmarshalled.Version.Compare(c.Version)).To(Equal(0))
				Expect(unmarshalled.Domain).To(Equal(c.Domain))
				Expect(unmarshalled.Repository).To(Equal(c.Repository))
				Expect(unmarshalled.Name).To(Equal(c.Name))
				Expect(unmarshalled.Layout).To(Equal(c.Layout))
				Expect(unmarshalled.MultiGroup).To(Equal(c.MultiGroup))
				Expect(unmarshalled.ComponentConfig).To(Equal(c.ComponentConfig))
				Expect(unmarshalled.Resources).To(Equal(c.Resources))
				Expect(unmarshalled.Plugins).To(HaveLen(len(c.Plugins)))
				// TODO: fully test Plugins field and not on its length
			},
			Entry("basic", s1, c1),
			Entry("full", s2, c2),
		)

		DescribeTable("Unmarshal should fail",
			func(content string) {
				var c cfg
				Expect(c.Unmarshal([]byte(content))).NotTo(Succeed())
			},
			Entry("for unknown fields", `field: 1
version: 3-alpha`),
		)
	})
})

var _ = Describe("New", func() {
	It("should return a new config for project configuration 3-alpha", func() {
		Expect(New().GetVersion().Compare(Version)).To(Equal(0))
	})
})
