/*
Copyright 2020 The Kubernetes Authors.

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

package config

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const v1beta1 = "v1beta1"

var _ = Describe("PluginConfig", func() {
	// Test plugin config. Don't want to export this config, but need it to
	// be accessible by test.
	type PluginConfig struct {
		Data1 string `json:"data-1"`
		Data2 string `json:"data-2"`
	}

	It("should encode correctly", func() {
		var (
			key            = "plugin-x"
			config         Config
			pluginConfig   PluginConfig
			expectedConfig Config
		)

		By("Using config version empty")
		config = Config{}
		pluginConfig = PluginConfig{}
		Expect(config.EncodePluginConfig(key, pluginConfig)).NotTo(Succeed())

		By("Using config version 2")
		config = Config{Version: Version2}
		pluginConfig = PluginConfig{}
		Expect(config.EncodePluginConfig(key, pluginConfig)).NotTo(Succeed())

		By("Using config version 2 with extra fields")
		config = Config{Version: Version2}
		pluginConfig = PluginConfig{
			Data1: "single plugin datum",
		}
		Expect(config.EncodePluginConfig(key, pluginConfig)).NotTo(Succeed())

		By("Using config version 3-alpha")
		config = Config{Version: Version3Alpha}
		pluginConfig = PluginConfig{}
		expectedConfig = Config{
			Version: Version3Alpha,
			Plugins: PluginConfigs{
				"plugin-x": map[string]interface{}{
					"data-1": "",
					"data-2": "",
				},
			},
		}
		Expect(config.EncodePluginConfig(key, pluginConfig)).To(Succeed())
		Expect(config).To(Equal(expectedConfig))

		By("Using config version 3-alpha with extra fields as struct")
		config = Config{Version: Version3Alpha}
		pluginConfig = PluginConfig{
			"plugin value 1",
			"plugin value 2",
		}
		expectedConfig = Config{
			Version: Version3Alpha,
			Plugins: PluginConfigs{
				"plugin-x": map[string]interface{}{
					"data-1": "plugin value 1",
					"data-2": "plugin value 2",
				},
			},
		}
		Expect(config.EncodePluginConfig(key, pluginConfig)).To(Succeed())
		Expect(config).To(Equal(expectedConfig))
	})

	It("should decode correctly", func() {
		var (
			key                  = "plugin-x"
			config               Config
			pluginConfig         PluginConfig
			expectedPluginConfig PluginConfig
		)

		By("Using config version 2")
		config = Config{Version: Version2}
		pluginConfig = PluginConfig{}
		Expect(config.DecodePluginConfig(key, &pluginConfig)).NotTo(Succeed())

		By("Using config version 2 with extra fields")
		config = Config{Version: Version2}
		pluginConfig = PluginConfig{
			Data1: "single plugin datum",
		}
		Expect(config.DecodePluginConfig(key, &pluginConfig)).NotTo(Succeed())

		By("Using empty config version 3-alpha")
		config = Config{
			Version: Version3Alpha,
			Plugins: PluginConfigs{
				"plugin-x": map[string]interface{}{},
			},
		}
		pluginConfig = PluginConfig{}
		expectedPluginConfig = PluginConfig{}
		Expect(config.DecodePluginConfig(key, &pluginConfig)).To(Succeed())
		Expect(pluginConfig).To(Equal(expectedPluginConfig))

		By("Using config version 3-alpha")
		config = Config{Version: Version3Alpha}
		pluginConfig = PluginConfig{}
		expectedPluginConfig = PluginConfig{}
		Expect(config.DecodePluginConfig(key, &pluginConfig)).To(Succeed())
		Expect(pluginConfig).To(Equal(expectedPluginConfig))

		By("Using config version 3-alpha with extra fields as struct")
		config = Config{
			Version: Version3Alpha,
			Plugins: PluginConfigs{
				"plugin-x": map[string]interface{}{
					"data-1": "plugin value 1",
					"data-2": "plugin value 2",
				},
			},
		}
		pluginConfig = PluginConfig{}
		expectedPluginConfig = PluginConfig{
			"plugin value 1",
			"plugin value 2",
		}
		Expect(config.DecodePluginConfig(key, &pluginConfig)).To(Succeed())
		Expect(pluginConfig).To(Equal(expectedPluginConfig))
	})

	It("should Marshal and Unmarshal a plugin", func() {
		By("Using config with extra fields as struct")
		cfg := Config{
			Version: Version3Alpha,
			Plugins: PluginConfigs{
				"plugin-x": map[string]interface{}{
					"data-1": "plugin value 1",
				},
			},
		}
		b, err := cfg.Marshal()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(b)).To(Equal("version: 3-alpha\nplugins:\n  plugin-x:\n    data-1: plugin value 1\n"))
		Expect(cfg.Unmarshal(b)).To(Succeed())
	})
})

var _ = Describe("Resource Version Compatibility", func() {

	var (
		c          *Config
		gvk1, gvk2 GVK

		defaultVersion = "v1"
	)

	BeforeEach(func() {
		c = &Config{}
		gvk1 = GVK{Group: "example", Version: "v1", Kind: "TestKind"}
		gvk2 = GVK{Group: "example", Version: "v1", Kind: "TestKind2"}
	})

	Context("resourceAPIVersionCompatible", func() {
		It("returns true for a list of empty resources", func() {
			Expect(c.resourceAPIVersionCompatible("crd", defaultVersion)).To(BeTrue())
		})
		It("returns true for one resource with an empty version", func() {
			c.Resources = []GVK{gvk1}
			Expect(c.resourceAPIVersionCompatible("crd", defaultVersion)).To(BeTrue())
		})
		It("returns true for one resource with matching version", func() {
			gvk1.CRDVersion = defaultVersion
			c.Resources = []GVK{gvk1}
			Expect(c.resourceAPIVersionCompatible("crd", defaultVersion)).To(BeTrue())
		})
		It("returns true for two resources with matching versions", func() {
			gvk1.CRDVersion = defaultVersion
			gvk2.CRDVersion = defaultVersion
			c.Resources = []GVK{gvk1, gvk2}
			Expect(c.resourceAPIVersionCompatible("crd", defaultVersion)).To(BeTrue())
		})
		It("returns false for one resource with a non-matching version", func() {
			gvk1.CRDVersion = v1beta1
			c.Resources = []GVK{gvk1}
			Expect(c.resourceAPIVersionCompatible("crd", defaultVersion)).To(BeFalse())
		})
		It("returns false for two resources containing a non-matching version", func() {
			gvk1.CRDVersion = v1beta1
			gvk2.CRDVersion = defaultVersion
			c.Resources = []GVK{gvk1, gvk2}
			Expect(c.resourceAPIVersionCompatible("crd", defaultVersion)).To(BeFalse())
		})

		It("returns false for two resources containing a non-matching version (webhooks)", func() {
			gvk1.WebhookVersion = v1beta1
			gvk2.WebhookVersion = defaultVersion
			c.Resources = []GVK{gvk1, gvk2}
			Expect(c.resourceAPIVersionCompatible("webhook", defaultVersion)).To(BeFalse())
		})
	})
})

var _ = Describe("Config", func() {
	var (
		c          *Config
		gvk1, gvk2 GVK
	)

	BeforeEach(func() {
		c = &Config{}
		gvk1 = GVK{Group: "example", Version: "v1", Kind: "TestKind"}
		gvk2 = GVK{Group: "example", Version: "v1", Kind: "TestKind2"}
	})

	Context("UpdateResource", func() {
		It("Adds a non-existing resource", func() {
			c.UpdateResources(gvk1)
			Expect(c.Resources).To(Equal([]GVK{gvk1}))
			// Update again to ensure idempotency.
			c.UpdateResources(gvk1)
			Expect(c.Resources).To(Equal([]GVK{gvk1}))
		})
		It("Updates an existing resource", func() {
			c.UpdateResources(gvk1)
			gvk := GVK{Group: gvk1.Group, Version: gvk1.Version, Kind: gvk1.Kind, CRDVersion: "v1"}
			c.UpdateResources(gvk)
			Expect(c.Resources).To(Equal([]GVK{gvk}))
		})
		It("Updates an existing resource with more than one resource present", func() {
			c.UpdateResources(gvk1)
			c.UpdateResources(gvk2)
			gvk := GVK{Group: gvk1.Group, Version: gvk1.Version, Kind: gvk1.Kind, CRDVersion: "v1"}
			c.UpdateResources(gvk)
			Expect(c.Resources).To(Equal([]GVK{gvk, gvk2}))
		})
	})
})
