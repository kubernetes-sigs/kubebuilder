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

package v3

import (
	"slices"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

func TestConfigV3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config V3 Suite")
}

var _ = Describe("Cfg", func() {
	const (
		domain = "my.domain"
		repo   = "myrepo"
		name   = "ProjectName"

		otherDomain = "other.domain"
		otherRepo   = "otherrepo"
		otherName   = "OtherProjectName"
	)

	var (
		c                Cfg
		pluginChain      []string
		otherPluginChain []string
	)

	BeforeEach(func() {
		pluginChain = []string{"go.kubebuilder.io/v2"}
		otherPluginChain = []string{"go.kubebuilder.io/v3"}

		c = Cfg{
			Version:     Version,
			Domain:      domain,
			Repository:  repo,
			Name:        name,
			PluginChain: pluginChain,
		}
	})

	Context("Version", func() {
		It("GetVersion should return version 3", func() {
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

	Context("Project name", func() {
		It("GetProjectName should return the name", func() {
			Expect(c.GetProjectName()).To(Equal(name))
		})

		It("SetProjectName should set the name", func() {
			Expect(c.SetProjectName(otherName)).To(Succeed())
			Expect(c.Name).To(Equal(otherName))
		})
	})

	Context("Plugin chain", func() {
		It("GetPluginChain should return the plugin chain", func() {
			Expect(c.GetPluginChain()).To(Equal(pluginChain))
		})

		It("SetPluginChain should set the plugin chain", func() {
			Expect(c.SetPluginChain(otherPluginChain)).To(Succeed())
			Expect([]string(c.PluginChain)).To(Equal(otherPluginChain))
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

	Context("Resources", func() {
		var (
			res              resource.Resource
			resWithoutPlural resource.Resource
			checkResource    func(result, expected resource.Resource)
		)

		BeforeEach(func() {
			res = resource.Resource{
				GVK: resource.GVK{
					Group:   "group",
					Version: "v1",
					Kind:    "Kind",
				},
				Plural: "kinds",
				Path:   "api/v1",
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
			resWithoutPlural = res.Copy()

			// As some of the tests insert directly into the slice without using the interface methods,
			// regular plural forms should not be present in here. rsWithoutPlural is used for this purpose.
			resWithoutPlural.Plural = ""

			// Auxiliary function for GetResource, AddResource and UpdateResource tests
			checkResource = func(result, expected resource.Resource) {
				Expect(result.GVK.IsEqualTo(expected.GVK)).To(BeTrue())
				Expect(result.Plural).To(Equal(expected.Plural))
				Expect(result.Path).To(Equal(expected.Path))
				if expected.API == nil {
					Expect(result.API).To(BeNil())
				} else {
					Expect(result.API).NotTo(BeNil())
					Expect(result.API.CRDVersion).To(Equal(expected.API.CRDVersion))
					Expect(result.API.Namespaced).To(Equal(expected.API.Namespaced))
				}
				Expect(result.Controller).To(Equal(expected.Controller))
				if expected.Webhooks == nil {
					Expect(result.Webhooks).To(BeNil())
				} else {
					Expect(result.Webhooks).NotTo(BeNil())
					Expect(result.Webhooks.WebhookVersion).To(Equal(expected.Webhooks.WebhookVersion))
					Expect(result.Webhooks.Defaulting).To(Equal(expected.Webhooks.Defaulting))
					Expect(result.Webhooks.Validation).To(Equal(expected.Webhooks.Validation))
					Expect(result.Webhooks.Conversion).To(Equal(expected.Webhooks.Conversion))
				}
			}
		})

		DescribeTable("ResourcesLength should return the number of resources",
			func(n int) {
				for range n {
					c.Resources = append(c.Resources, resWithoutPlural)
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
			c.Resources = append(c.Resources, resWithoutPlural)
			Expect(c.HasResource(res.GVK)).To(BeTrue())
		})

		It("GetResource should fail for a non-existent resource", func() {
			_, err := c.GetResource(res.GVK)
			Expect(err).To(HaveOccurred())
		})

		It("GetResource should return an existent resource", func() {
			c.Resources = append(c.Resources, resWithoutPlural)
			r, err := c.GetResource(res.GVK)
			Expect(err).NotTo(HaveOccurred())

			checkResource(r, res)
		})

		It("GetResources should return a slice of the tracked resources", func() {
			c.Resources = append(c.Resources, resWithoutPlural, resWithoutPlural, resWithoutPlural)
			resources, err := c.GetResources()
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(Equal([]resource.Resource{res, res, res}))
		})

		It("AddResource should add the provided resource if non-existent", func() {
			l := len(c.Resources)
			Expect(c.AddResource(res)).To(Succeed())
			Expect(c.Resources).To(HaveLen(l + 1))

			checkResource(c.Resources[0], resWithoutPlural)
		})

		It("AddResource should do nothing if the resource already exists", func() {
			c.Resources = append(c.Resources, res)
			l := len(c.Resources)
			Expect(c.AddResource(res)).To(Succeed())
			Expect(c.Resources).To(HaveLen(l))
		})

		It("UpdateResource should add the provided resource if non-existent", func() {
			l := len(c.Resources)
			Expect(c.UpdateResource(res)).To(Succeed())
			Expect(c.Resources).To(HaveLen(l + 1))

			checkResource(c.Resources[0], resWithoutPlural)
		})

		It("UpdateResource should update it if the resource already exists", func() {
			r := resource.Resource{
				GVK: resource.GVK{
					Group:   "group",
					Version: "v1",
					Kind:    "Kind",
				},
				Path: "api/v1",
			}
			c.Resources = append(c.Resources, r)
			l := len(c.Resources)
			checkResource(c.Resources[0], r)

			Expect(c.UpdateResource(res)).To(Succeed())
			Expect(c.Resources).To(HaveLen(l))

			checkResource(c.Resources[0], resWithoutPlural)
		})

		It("RemoveResource should remove the resource if it exists", func() {
			c.Resources = append(c.Resources, res)
			l := len(c.Resources)
			Expect(c.RemoveResource(res.GVK)).To(Succeed())
			Expect(c.Resources).To(HaveLen(l - 1))
			Expect(c.HasResource(res.GVK)).To(BeFalse())
		})

		It("RemoveResource should return an error if the resource does not exist", func() {
			Expect(c.RemoveResource(res.GVK)).NotTo(Succeed())
		})

		It("RemoveResource should remove only the specified resource", func() {
			res2 := resource.Resource{
				GVK: resource.GVK{
					Group:   "group",
					Version: "v1",
					Kind:    "OtherKind",
				},
			}
			c.Resources = append(c.Resources, res, res2)
			Expect(c.Resources).To(HaveLen(2))

			Expect(c.RemoveResource(res.GVK)).To(Succeed())
			Expect(c.Resources).To(HaveLen(1))
			Expect(c.HasResource(res.GVK)).To(BeFalse())
			Expect(c.HasResource(res2.GVK)).To(BeTrue())
		})

		It("SetResourceWebhooks should update webhook configuration", func() {
			resWithWebhooks := resource.Resource{
				GVK: resource.GVK{
					Group:   "group",
					Version: "v1",
					Kind:    "Kind",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Defaulting:     true,
					Validation:     true,
				},
			}
			c.Resources = append(c.Resources, resWithWebhooks)

			// Update to remove defaulting webhook
			newWebhooks := &resource.Webhooks{
				WebhookVersion: "v1",
				Defaulting:     false,
				Validation:     true,
			}
			Expect(c.SetResourceWebhooks(resWithWebhooks.GVK, newWebhooks)).To(Succeed())

			updated, err := c.GetResource(resWithWebhooks.GVK)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Webhooks.Defaulting).To(BeFalse())
			Expect(updated.Webhooks.Validation).To(BeTrue())
		})

		It("SetResourceWebhooks should clear webhooks when nil provided", func() {
			resWithWebhooks := resource.Resource{
				GVK: resource.GVK{
					Group:   "group",
					Version: "v1",
					Kind:    "Kind",
				},
				Webhooks: &resource.Webhooks{
					WebhookVersion: "v1",
					Defaulting:     true,
				},
			}
			c.Resources = append(c.Resources, resWithWebhooks)

			Expect(c.SetResourceWebhooks(resWithWebhooks.GVK, nil)).To(Succeed())

			updated, err := c.GetResource(resWithWebhooks.GVK)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Webhooks).To(BeNil())
		})

		It("SetResourceWebhooks should fail if resource not found", func() {
			Expect(c.SetResourceWebhooks(res.GVK, nil)).NotTo(Succeed())
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

		It("ListCRDVersions should return an empty list with no tracked resources", func() {
			Expect(c.ListCRDVersions()).To(BeEmpty())
		})

		It("ListCRDVersions should return a list of tracked resources CRD versions", func() {
			c.Resources = append(c.Resources,
				resource.Resource{
					GVK: resource.GVK{
						Group:   res.Group,
						Version: res.Version,
						Kind:    res.Kind,
					},
					API: &resource.API{CRDVersion: "v1beta1"},
				},
				resource.Resource{
					GVK: resource.GVK{
						Group:   res.Group,
						Version: res.Version,
						Kind:    "OtherKind",
					},
					API: &resource.API{CRDVersion: "v1"},
				},
			)
			versions := c.ListCRDVersions()
			slices.Sort(versions) // ListCRDVersions has no order guarantee so sorting for reproducibility
			Expect(versions).To(Equal([]string{"v1", "v1beta1"}))
		})

		It("ListWebhookVersions should return an empty list with no tracked resources", func() {
			Expect(c.ListWebhookVersions()).To(BeEmpty())
		})

		It("ListWebhookVersions should return a list of tracked resources webhook versions", func() {
			c.Resources = append(c.Resources,
				resource.Resource{
					GVK: resource.GVK{
						Group:   res.Group,
						Version: res.Version,
						Kind:    res.Kind,
					},
					Webhooks: &resource.Webhooks{WebhookVersion: "v1beta1"},
				},
				resource.Resource{
					GVK: resource.GVK{
						Group:   res.Group,
						Version: res.Version,
						Kind:    "OtherKind",
					},
					Webhooks: &resource.Webhooks{WebhookVersion: "v1"},
				},
			)
			versions := c.ListWebhookVersions()
			slices.Sort(versions) // ListWebhookVersions has no order guarantee so sorting for reproducibility
			Expect(versions).To(Equal([]string{"v1", "v1beta1"}))
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
			c0, c1, c2 Cfg
			pluginCfg  PluginConfig
		)

		BeforeEach(func() {
			c0 = Cfg{
				Version:     Version,
				Domain:      domain,
				Repository:  repo,
				Name:        name,
				PluginChain: pluginChain,
			}
			c1 = Cfg{
				Version:     Version,
				Domain:      domain,
				Repository:  repo,
				Name:        name,
				PluginChain: pluginChain,
				Plugins: pluginConfigs{
					key: map[string]any{
						"data-1": "",
					},
				},
			}
			c2 = Cfg{
				Version:     Version,
				Domain:      domain,
				Repository:  repo,
				Name:        name,
				PluginChain: pluginChain,
				Plugins: pluginConfigs{
					key: map[string]any{
						"data-1": "plugin value 1",
						"data-2": "plugin value 2",
					},
				},
			}
			pluginCfg = PluginConfig{
				Data1: "plugin value 1",
				Data2: "plugin value 2",
			}
		})

		It("DecodePluginConfig should fail for no plugin config object", func() {
			err := c0.DecodePluginConfig(key, &pluginCfg)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(config.PluginKeyNotFoundError{Key: key}))
		})

		It("DecodePluginConfig should fail to retrieve data from a non-existent plugin", func() {
			err := c1.DecodePluginConfig("plugin-y", &pluginCfg)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(config.PluginKeyNotFoundError{Key: "plugin-y"}))
		})

		DescribeTable("DecodePluginConfig should retrieve the plugin data correctly",
			func(getCfg func() Cfg, expected func() PluginConfig) {
				pluginCfg = PluginConfig{} // reset to not reuse values
				Expect(getCfg().DecodePluginConfig(key, &pluginCfg)).To(Succeed())
				Expect(pluginCfg).To(Equal(expected()))
			},
			Entry("for an empty plugin config object", func() Cfg { return c1 }, func() PluginConfig { return PluginConfig{} }),
			Entry("for a full plugin config object", func() Cfg { return c2 }, func() PluginConfig { return pluginCfg }),
			// TODO (coverage): add cases where yaml.Marshal returns an error
			// TODO (coverage): add cases where yaml.Unmarshal returns an error
		)

		DescribeTable("EncodePluginConfig should encode the plugin data correctly",
			func(getPluginCfg func() PluginConfig, expectedCfg func() Cfg) {
				Expect(c.EncodePluginConfig(key, getPluginCfg())).To(Succeed())
				Expect(c).To(Equal(expectedCfg()))
			},
			Entry("for an empty plugin config object", func() PluginConfig { return PluginConfig{} }, func() Cfg { return c1 }),
			Entry("for a full plugin config object", func() PluginConfig { return pluginCfg }, func() Cfg { return c2 }),
			// TODO (coverage): add cases where yaml.Marshal returns an error
			// TODO (coverage): add cases where yaml.Unmarshal returns an error
		)
	})

	Context("Persistence", func() {
		var (
			c1, c2        Cfg
			s1, s1bis, s2 string
		)

		BeforeEach(func() {
			c1 = Cfg{
				Version:     Version,
				Domain:      domain,
				Repository:  repo,
				Name:        name,
				PluginChain: pluginChain,
			}
			c2 = Cfg{
				Version:     Version,
				Domain:      otherDomain,
				Repository:  otherRepo,
				Name:        otherName,
				PluginChain: otherPluginChain,
				MultiGroup:  true,
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
						API:        &resource.API{CRDVersion: "v1"},
						Controller: true,
						Webhooks:   &resource.Webhooks{WebhookVersion: "v1"},
					},
					{
						GVK: resource.GVK{
							Group:   "group",
							Version: "v1-beta",
							Kind:    "Kind",
						},
						Plural:   "kindes",
						API:      nil,
						Webhooks: nil,
					},
					{
						GVK: resource.GVK{
							Group:   "group2",
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
					},
				},
				Plugins: pluginConfigs{
					"plugin-x": map[string]any{
						"data-1": "single plugin datum",
					},
					"plugin-y/v1": map[string]any{
						"data-1": "plugin value 1",
						"data-2": "plugin value 2",
						"data-3": []string{"plugin value 3", "plugin value 4"},
					},
				},
			}
			// TODO: include cases with Path when added
			s1 = `domain: my.domain
layout:
- go.kubebuilder.io/v2
projectName: ProjectName
repo: myrepo
version: "3"
`
			s1bis = `domain: my.domain
layout: go.kubebuilder.io/v2
projectName: ProjectName
repo: myrepo
version: "3"
`
			s2 = `domain: other.domain
layout:
- go.kubebuilder.io/v3
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
  controller: true
  group: group
  kind: Kind2
  version: v1
  webhooks:
    webhookVersion: v1
- group: group
  kind: Kind
  plural: kindes
  version: v1-beta
- api:
    crdVersion: v1
    namespaced: true
  controller: true
  group: group2
  kind: Kind
  version: v1
  webhooks:
    conversion: true
    defaulting: true
    validation: true
    webhookVersion: v1
version: "3"
`
		})

		DescribeTable("MarshalYAML should succeed",
			func(getCfg func() Cfg, getContent func() string) {
				b, err := getCfg().MarshalYAML()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal(getContent()))
			},
			Entry("for a basic configuration", func() Cfg { return c1 }, func() string { return s1 }),
			Entry("for a full configuration", func() Cfg { return c2 }, func() string { return s2 }),
		)

		DescribeTable("UnmarshalYAML should succeed",
			func(getContent func() string, getCfg func() Cfg) {
				var unmarshalled Cfg
				Expect(unmarshalled.UnmarshalYAML([]byte(getContent()))).To(Succeed())
				c := getCfg()
				Expect(unmarshalled.Version.Compare(c.Version)).To(Equal(0))
				Expect(unmarshalled.Domain).To(Equal(c.Domain))
				Expect(unmarshalled.Repository).To(Equal(c.Repository))
				Expect(unmarshalled.Name).To(Equal(c.Name))
				Expect(unmarshalled.PluginChain).To(Equal(c.PluginChain))
				Expect(unmarshalled.MultiGroup).To(Equal(c.MultiGroup))
				Expect(unmarshalled.Resources).To(Equal(c.Resources))
				Expect(unmarshalled.Plugins).To(HaveLen(len(c.Plugins)))
				// TODO: fully test Plugins field and not on its length
			},
			Entry("basic", func() string { return s1 }, func() Cfg { return c1 }),
			Entry("full", func() string { return s2 }, func() Cfg { return c2 }),
			Entry("string layout", func() string { return s1bis }, func() Cfg { return c1 }),
		)

		DescribeTable("UnmarshalYAML should fail",
			func(content string) {
				var c Cfg
				Expect(c.UnmarshalYAML([]byte(content))).NotTo(Succeed())
			},
			Entry("for unknown fields", `field: 1
version: "3"`),
		)
	})
})

var _ = Describe("New", func() {
	It("should return a new config for project configuration 3", func() {
		Expect(New().GetVersion().Compare(Version)).To(Equal(0))
	})
})
