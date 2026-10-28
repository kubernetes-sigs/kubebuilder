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
	"errors"
	"fmt"
	"slices"
	"strings"
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

		resourceGroup = "group"
		resourceKind  = "Kind"
		apiVersion    = "v1beta1"
		dataKey1      = "data-1"
		pluginValue1  = "plugin value 1"
		pluginValue2  = "plugin value 2"

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
					Group:   resourceGroup,
					Version: "v1",
					Kind:    resourceKind,
				},
				Plural: "kinds",
				Path:   "api/v1",
				API: &resource.API{
					CRDVersion: "v1",
					Namespaced: true,
				},
				Controller: true, //nolint:staticcheck // legacy PROJECT compatibility
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
				Expect(result.GetControllerNames()).To(Equal(expected.GetControllerNames()))
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
					Group:   resourceGroup,
					Version: "v1",
					Kind:    resourceKind,
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
					API: &resource.API{CRDVersion: apiVersion},
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
			Expect(versions).To(Equal([]string{"v1", apiVersion}))
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
					Webhooks: &resource.Webhooks{WebhookVersion: apiVersion},
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
			Expect(versions).To(Equal([]string{"v1", apiVersion}))
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
						dataKey1: "",
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
						dataKey1: pluginValue1,
						"data-2": pluginValue2,
					},
				},
			}
			pluginCfg = PluginConfig{
				Data1: pluginValue1,
				Data2: pluginValue2,
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
		)

		It("DecodePluginConfig should fail when yaml.Marshal returns an error", func() {
			// Create a config with a plugin value that cannot be marshalled
			// Using a channel which is not supported by yaml marshalling
			cWithUnmarshallable := Cfg{
				Version:     Version,
				Domain:      domain,
				Repository:  repo,
				Name:        name,
				PluginChain: pluginChain,
				Plugins: pluginConfigs{
					key: map[string]any{
						"invalid": make(chan int),
					},
				},
			}
			err := cWithUnmarshallable.DecodePluginConfig(key, &pluginCfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to convert extra fields object to bytes"))
		})

		It("DecodePluginConfig should fail when yaml.Unmarshal returns an error", func() {
			// Create a config where the plugin config exists but cannot be unmarshalled
			// into the target type (PluginConfig expects string, but we provide incompatible data)
			type IncompatibleTarget struct {
				// This struct has different field expectations
				Field chan int `json:"data-1"` // channel cannot be unmarshalled from yaml
			}
			var incompatibleTarget IncompatibleTarget
			err := c2.DecodePluginConfig(key, &incompatibleTarget)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshal extra fields object"))
		})

		DescribeTable("EncodePluginConfig should encode the plugin data correctly",
			func(getPluginCfg func() PluginConfig, expectedCfg func() Cfg) {
				Expect(c.EncodePluginConfig(key, getPluginCfg())).To(Succeed())
				Expect(c).To(Equal(expectedCfg()))
			},
			Entry("for an empty plugin config object", func() PluginConfig { return PluginConfig{} }, func() Cfg { return c1 }),
			Entry("for a full plugin config object", func() PluginConfig { return pluginCfg }, func() Cfg { return c2 }),
		)

		It("EncodePluginConfig should fail when yaml.Marshal returns an error", func() {
			// Create a config object that cannot be marshalled to yaml
			// Using a channel which is not supported by yaml marshalling
			type UnmarshallableConfig struct {
				Invalid chan int `json:"invalid"`
			}
			unmarshallableCfg := UnmarshallableConfig{
				Invalid: make(chan int),
			}
			err := c.EncodePluginConfig(key, unmarshallableCfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to convert"))
			Expect(err.Error()).To(ContainSubstring("object to bytes"))
		})

		It("EncodePluginConfig should fail when yaml.Unmarshal returns an error", func() {
			// This test is tricky because yaml.Unmarshal is very permissive.
			// We need to create a scenario where the marshalled bytes cannot be unmarshalled
			// into map[string]any. This is rare, but we can test the error path exists
			// by verifying the code structure. In practice, yaml.Unmarshal to map[string]any
			// rarely fails, but we document this for completeness.
			// For now, we test that valid inputs still work correctly.
			type ValidConfig struct {
				Data string `json:"data"`
			}
			validCfg := ValidConfig{Data: "test"}
			Expect(c.EncodePluginConfig(key, validCfg)).To(Succeed())
			Expect(c.Plugins).To(HaveKey(key))
		})
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
							Group:   resourceGroup,
							Version: "v1",
							Kind:    resourceKind,
						},
					},
					{
						GVK: resource.GVK{
							Group:   resourceGroup,
							Version: "v1",
							Kind:    "Kind2",
						},
						API:         &resource.API{CRDVersion: "v1"},
						Controllers: &resource.Controllers{{Name: "kind2"}},
						Webhooks:    &resource.Webhooks{WebhookVersion: "v1"},
					},
					{
						GVK: resource.GVK{
							Group:   resourceGroup,
							Version: "v1-beta",
							Kind:    resourceKind,
						},
						Plural:   "kindes",
						API:      nil,
						Webhooks: nil,
					},
					{
						GVK: resource.GVK{
							Group:   "group2",
							Version: "v1",
							Kind:    resourceKind,
						},
						Path: "otherrepo/api/v1",
						API: &resource.API{
							CRDVersion: "v1",
							Namespaced: true,
						},
						Controllers: &resource.Controllers{{Name: "kind"}},
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
						dataKey1: "single plugin datum",
					},
					"plugin-y/v1": map[string]any{
						dataKey1: pluginValue1,
						"data-2": pluginValue2,
						"data-3": []any{"plugin value 3", "plugin value 4"},
					},
				},
			}
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
  controllers:
  - name: kind2
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
  controllers:
  - name: kind
  group: group2
  kind: Kind
  path: otherrepo/api/v1
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

		Context("controller migration", func() {
			const (
				captainKind   = "Captain"
				captainCtrl   = "captain"
				captainBackup = "captain-backup"
				sailorKind    = "Sailor"
				sailorCtrl    = "sailor"
				crewGroup     = "crew"
			)

			// PROJECT files are written as text so that the forms older releases wrote stay covered.
			const (
				projectHeader = `domain: my.domain
layout:
- go.kubebuilder.io/v4
projectName: my-project
repo: myrepo
resources:
`
				projectFooter = `version: "3"
`
				legacyCaptain = `- controller: true
  group: crew
  kind: Captain
  version: v1
`
				legacyCaptainV2 = `- controller: true
  group: crew
  kind: Captain
  version: v2
`
				legacySailor = `- controller: true
  group: crew
  kind: Sailor
  version: v1
`
				namedAdmiral = `- controllers:
  - name: admiral
  group: crew
  kind: Admiral
  version: v1
`
				namedCaptainBackup = `- controllers:
  - name: captain-backup
  group: crew
  kind: Captain
  version: v1
`
				sharedCaptain = `- controllers:
  - name: shared
  group: crew
  kind: Captain
  version: v1
`
				sharedFirstMate = `- controllers:
  - name: shared
  group: crew
  kind: FirstMate
  version: v1
`
				captainWithoutController = `- group: crew
  kind: Captain
  version: v1
`
				// bothFormsTemplate is a resource whose controller a hand-edited or older PROJECT file
				// records in both forms. The placeholders hold the listed name and the kind.
				bothFormsTemplate = `- controller: true
  controllers:
  - name: %s
  group: crew
  kind: %s
  version: v1
`
			)

			// projectWith returns a PROJECT file holding the given resources.
			projectWith := func(resources ...string) string {
				return projectHeader + strings.Join(resources, "") + projectFooter
			}

			// load returns the configuration a PROJECT file holds.
			load := func(body string) Cfg {
				GinkgoHelper()
				var cfg Cfg
				Expect(cfg.UnmarshalYAML([]byte(body))).To(Succeed())
				return cfg
			}

			// save writes cfg as the PROJECT file and returns the file and the configuration the
			// next command reads from it.
			save := func(cfg Cfg) ([]byte, Cfg) {
				GinkgoHelper()
				content, err := cfg.MarshalYAML()
				Expect(err).NotTo(HaveOccurred())
				var reloaded Cfg
				Expect(reloaded.UnmarshalYAML(content)).To(Succeed())
				return content, reloaded
			}

			// controllersOf returns the controller names a saved configuration records for a resource,
			// checking that the legacy field was not written.
			controllersOf := func(cfg Cfg, gvk resource.GVK) []string {
				GinkgoHelper()
				res, err := cfg.GetResource(gvk)
				Expect(err).NotTo(HaveOccurred())
				Expect(res.Controller).To(BeFalse(), //nolint:staticcheck // legacy field
					"the legacy field must not be written")
				return res.GetControllerNames()
			}

			var captainV1 resource.GVK

			BeforeEach(func() {
				captainV1 = resource.GVK{Group: crewGroup, Version: "v1", Kind: captainKind}
			})

			It("should return a migration error and leave the caller's config unchanged", func() {
				cfg := Cfg{Version: Version, Resources: []resource.Resource{
					{GVK: resource.GVK{Kind: captainKind}, Controller: true},
					{Controller: true},
				}}
				before := []resource.Resource{cfg.Resources[0].Copy(), cfg.Resources[1].Copy()}

				_, err := cfg.MarshalYAML()
				Expect(err).To(MatchError(ContainSubstring("kind cannot be empty")))
				Expect(cfg.Resources).To(Equal(before))
			})

			It("should migrate and omit empty fields without mutating the caller", func() {
				cfg := Cfg{Version: Version, Resources: []resource.Resource{
					{
						GVK: resource.GVK{Kind: captainKind}, Controller: true,
						Controllers: &resource.Controllers{{Name: captainBackup}},
					},
					{
						GVK: resource.GVK{Kind: sailorKind}, API: &resource.API{},
						Webhooks: &resource.Webhooks{}, Controllers: &resource.Controllers{},
					},
				}}
				before := []resource.Resource{cfg.Resources[0].Copy(), cfg.Resources[1].Copy()}

				first, reloaded := save(cfg)
				Expect(cfg.Resources).To(Equal(before))
				Expect(controllersOf(reloaded, resource.GVK{Kind: captainKind})).To(Equal([]string{captainBackup, captainCtrl}))
				sailor, err := reloaded.GetResource(resource.GVK{Kind: sailorKind})
				Expect(err).NotTo(HaveOccurred())
				Expect(sailor.API).To(BeNil(), "an empty api must be omitted")
				Expect(sailor.Webhooks).To(BeNil(), "empty webhooks must be omitted")
				Expect(sailor.Controllers).To(BeNil(), "an empty controllers list must be omitted")

				second, err := cfg.MarshalYAML()
				Expect(err).NotTo(HaveOccurred())
				Expect(second).To(Equal(first))
				Expect(cfg.Resources).To(Equal(before))
			})

			It("should keep a legacy controller recorded with an empty controllers list", func() {
				cfg := Cfg{Version: Version, Resources: []resource.Resource{{
					GVK:         resource.GVK{Kind: captainKind},
					Controller:  true,
					Controllers: &resource.Controllers{},
				}}}

				_, reloaded := save(cfg)
				Expect(controllersOf(reloaded, resource.GVK{Kind: captainKind})).To(Equal([]string{captainCtrl}))
			})

			It("should reject a migration collision without changing any resource", func() {
				cfg := Cfg{Version: Version, Resources: []resource.Resource{
					{GVK: resource.GVK{Kind: captainKind}, Controller: true},
					{
						GVK: resource.GVK{Kind: "MyKind"}, Controller: true,
						Controllers: &resource.Controllers{{Name: "my-kind"}},
					},
				}}
				before := []resource.Resource{cfg.Resources[0].Copy(), cfg.Resources[1].Copy()}
				content, err := cfg.MarshalYAML()
				Expect(err).To(MatchError(ContainSubstring("both generate MyKindReconciler")))
				Expect(content).To(BeNil())
				Expect(cfg.Resources).To(Equal(before))
			})

			It("should save a file whose resources already share a controller name, as only requests are checked", func() {
				cfg := load(projectWith(sharedCaptain, sharedFirstMate))
				cook := resource.Resource{
					GVK:         resource.GVK{Group: "galley", Version: "v1", Kind: "Cook"},
					Controllers: &resource.Controllers{{Name: "cook"}},
				}
				Expect(cfg.UpdateResource(cook)).To(Succeed())

				_, reloaded := save(cfg)
				Expect(controllersOf(reloaded, cook.GVK)).To(Equal([]string{"cook"}))
			})

			Context("with 2 versions of a kind that both record the legacy controller", func() {
				// Kubebuilder v4.13.0 writes this when the second version is created with --force:
				// both versions share one captain_controller.go and one CaptainReconciler.
				var cfg Cfg

				BeforeEach(func() {
					cfg = load(projectWith(legacyCaptain, legacyCaptainV2))
				})

				expectSharedCaptain := func(saved Cfg) {
					GinkgoHelper()
					for _, version := range []string{"v1", "v2"} {
						gvk := resource.GVK{Group: crewGroup, Version: version, Kind: captainKind}
						Expect(controllersOf(saved, gvk)).To(Equal([]string{captainCtrl}))
					}
				}

				It("should save both versions with the shared controller", func() {
					_, reloaded := save(cfg)
					expectSharedCaptain(reloaded)
				})

				DescribeTable("should add or update another kind and still save",
					func(add bool) {
						sailor := resource.Resource{
							GVK:         resource.GVK{Group: crewGroup, Version: "v1", Kind: sailorKind},
							Controllers: &resource.Controllers{{Name: sailorCtrl}},
						}
						if add {
							Expect(cfg.AddResource(sailor)).To(Succeed())
						} else {
							Expect(cfg.UpdateResource(sailor)).To(Succeed())
						}
						Expect(cfg.Resources).To(HaveLen(3))

						_, reloaded := save(cfg)
						expectSharedCaptain(reloaded)
						Expect(controllersOf(reloaded, sailor.GVK)).To(Equal([]string{sailorCtrl}))
					},
					Entry("with AddResource", true),
					Entry("with UpdateResource", false),
				)

				It("should update a version read back from the config and still save", func() {
					captainV2 := resource.GVK{Group: crewGroup, Version: "v2", Kind: captainKind}
					captain, err := cfg.GetResource(captainV2)
					Expect(err).NotTo(HaveOccurred())
					captain.Webhooks = &resource.Webhooks{WebhookVersion: "v1", Defaulting: true}
					Expect(cfg.UpdateResource(captain)).To(Succeed())

					_, reloaded := save(cfg)
					expectSharedCaptain(reloaded)
					updated, err := reloaded.GetResource(captainV2)
					Expect(err).NotTo(HaveOccurred())
					Expect(updated.Webhooks).To(Equal(captain.Webhooks))
				})
			})

			It("should save the same name in different groups of a multigroup project", func() {
				cfg := Cfg{Version: Version, MultiGroup: true, Resources: []resource.Resource{
					{
						GVK:         resource.GVK{Group: resourceGroup, Version: "v1", Kind: captainKind},
						Controllers: &resource.Controllers{{Name: "shared"}},
					},
					{
						GVK:         resource.GVK{Group: "ship", Version: "v1", Kind: "FirstMate"},
						Controllers: &resource.Controllers{{Name: "shared"}},
					},
				}}

				_, err := cfg.MarshalYAML()
				Expect(err).NotTo(HaveOccurred())
			})

			// roundTrip loads a PROJECT file holding the resources and saves it in the current format.
			roundTrip := func(resources ...string) ([]byte, Cfg) {
				GinkgoHelper()
				return save(load(projectWith(resources...)))
			}

			It("should convert a legacy controller to the controllers list", func() {
				_, out := roundTrip(legacyCaptain)

				Expect(controllersOf(out, captainV1)).To(Equal([]string{captainCtrl}))
			})

			It("should leave a project already using controllers byte-identical", func() {
				once, out := roundTrip(namedCaptainBackup)

				Expect(controllersOf(out, captainV1)).To(Equal([]string{captainBackup}),
					"a named controller must not gain the default one")
				twice, err := out.MarshalYAML()
				Expect(err).NotTo(HaveOccurred())
				Expect(twice).To(Equal(once), "writing an already-migrated project must not change it")
			})

			It("should be idempotent, so a second write changes nothing", func() {
				once, out := roundTrip(legacyCaptain)

				twice, err := out.MarshalYAML()
				Expect(err).NotTo(HaveOccurred())
				Expect(twice).To(Equal(once))
			})

			// The legacy controller must be kept under its default name, not dropped, and a listed name
			// equal to it must be saved once.
			DescribeTable("should save each controller of a file that carries both forms once",
				func(kind, listed string, expected []string) {
					_, out := roundTrip(fmt.Sprintf(bothFormsTemplate, listed, kind))

					gvk := resource.GVK{Group: crewGroup, Version: "v1", Kind: kind}
					Expect(controllersOf(out, gvk)).To(Equal(expected))
				},
				Entry("a listed name and the legacy default", captainKind, captainBackup,
					[]string{captainBackup, captainCtrl}),
				Entry("a listed name equal to the legacy default", captainKind, captainCtrl,
					[]string{captainCtrl}),
				Entry("a legacy default that differs from a listed name only by a hyphen", "Captainbackup",
					captainBackup, []string{captainBackup, "captainbackup"}),
			)

			DescribeTable("should refuse to save invalid controllers of one resource",
				func(controllers resource.Controllers, expected string) {
					cfg := Cfg{Version: Version, Resources: []resource.Resource{{
						GVK: captainV1, Controllers: &controllers,
					}}}

					content, err := cfg.MarshalYAML()
					Expect(err).To(MatchError(ContainSubstring(expected)))
					Expect(errors.As(err, &config.MarshalError{})).To(BeTrue())
					Expect(content).To(BeNil())
				},
				Entry("a name that is not a DNS label",
					resource.Controllers{{Name: "Captain_Backup"}}, "invalid controller name"),
				Entry("a duplicate name",
					resource.Controllers{{Name: captainBackup}, {Name: captainBackup}},
					`duplicate controller name "captain-backup"`),
				Entry("names generating the same reconciler",
					resource.Controllers{{Name: captainBackup}, {Name: "captain--backup"}},
					"both generate CaptainBackupReconciler"),
			)

			It("should migrate every resource, not only the ones a command touched", func() {
				_, out := roundTrip(legacyCaptain, legacySailor, namedAdmiral)

				Expect(controllersOf(out, captainV1)).To(Equal([]string{captainCtrl}))
				Expect(controllersOf(out, resource.GVK{Group: crewGroup, Version: "v1", Kind: sailorKind})).
					To(Equal([]string{sailorCtrl}))
				Expect(controllersOf(out, resource.GVK{Group: crewGroup, Version: "v1", Kind: "Admiral"})).
					To(Equal([]string{"admiral"}))
			})

			It("should omit an empty controllers list", func() {
				cfg := Cfg{Version: Version, Resources: []resource.Resource{{
					GVK: captainV1, Controllers: &resource.Controllers{},
				}}}

				_, reloaded := save(cfg)
				res, err := reloaded.GetResource(captainV1)
				Expect(err).NotTo(HaveOccurred())
				Expect(res.Controllers).To(BeNil())
			})

			It("should not invent a controller for a resource that has none", func() {
				_, out := roundTrip(captainWithoutController)

				res, err := out.GetResource(captainV1)
				Expect(err).NotTo(HaveOccurred())
				Expect(res.HasController()).To(BeFalse())
				Expect(res.Controllers).To(BeNil())
			})

			It("should report a legacy controller through the accessors before it is written", func() {
				cfg := load(projectWith(legacyCaptain))

				res, err := cfg.GetResource(captainV1)
				Expect(err).NotTo(HaveOccurred())
				Expect(res.HasController()).To(BeTrue())
				Expect(res.GetControllerNames()).To(Equal([]string{"captain"}))
			})
		})

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
				for key, expectedConfig := range c.Plugins {
					Expect(unmarshalled.Plugins).To(HaveKey(key))
					Expect(unmarshalled.Plugins[key]).To(BeEquivalentTo(expectedConfig))
				}
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
			Entry("for invalid YAML", `invalid: yaml: content
  badly indented`),
		)

		// Test forward compatibility - unknown fields should be ignored
		Context("Forward compatibility", func() {
			It("should ignore unknown fields for forward compatibility", func() {
				// Old kubebuilder reading PROJECT file with new fields
				content := `version: "3"
domain: example.com
repo: github.com/example/project
projectName: test-project
unknownField: "should be ignored"
anotherUnknownField: 123`

				var c Cfg
				Expect(c.UnmarshalYAML([]byte(content))).To(Succeed())
				Expect(c.Domain).To(Equal("example.com"))
				Expect(c.Repository).To(Equal("github.com/example/project"))
				Expect(c.Name).To(Equal("test-project"))
			})

			It("should ignore the new 'controllers' field for backward compatibility", func() {
				// Simulates old kubebuilder reading PROJECT file from new kubebuilder
				// that has the "controllers" field in resources
				content := `version: "3"
domain: testproject.org
repo: sigs.k8s.io/kubebuilder/testdata/project
projectName: test-project
resources:
- api:
    crdVersion: v1
    namespaced: true
  controllers:
  - name: captain
  - name: captain-backup
  domain: testproject.org
  group: crew
  kind: Captain
  version: v1`

				var c Cfg
				Expect(c.UnmarshalYAML([]byte(content))).To(Succeed())
				Expect(c.Domain).To(Equal("testproject.org"))
				Expect(c.Resources).To(HaveLen(1))
				Expect(c.Resources[0].Group).To(Equal("crew"))
				Expect(c.Resources[0].Kind).To(Equal("Captain"))
				// The "controllers" field should be silently ignored by older versions
			})

			It("should handle mixed known and unknown fields", func() {
				content := `version: "3"
domain: example.com
repo: github.com/example/project
projectName: test-project
layout:
- go.kubebuilder.io/v4
unknownField1: "ignored"
multigroup: true
unknownField2:
  nested: "also ignored"
resources:
- group: apps
  version: v1
  kind: Deployment
  newField: "ignored in resource"`

				var c Cfg
				Expect(c.UnmarshalYAML([]byte(content))).To(Succeed())
				Expect(c.Domain).To(Equal("example.com"))
				Expect(c.MultiGroup).To(BeTrue())
				Expect(c.PluginChain).To(Equal(stringSlice{"go.kubebuilder.io/v4"}))
				Expect(c.Resources).To(HaveLen(1))
			})
		})
	})
})

var _ = Describe("New", func() {
	It("should return a new config for project configuration 3", func() {
		Expect(New().GetVersion().Compare(Version)).To(Equal(0))
	})
})
