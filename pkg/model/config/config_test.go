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

var _ = Describe("Config", func() {
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

		By("Using config version 1")
		config = Config{Version: Version1}
		pluginConfig = PluginConfig{}
		Expect(config.EncodePluginConfig(key, pluginConfig)).To(HaveOccurred())

		By("Using config version 1 with extra fields")
		config = Config{Version: Version1}
		pluginConfig = PluginConfig{
			Data1: "single plugin datum",
		}
		Expect(config.EncodePluginConfig(key, pluginConfig)).To(HaveOccurred())

		By("Using config version 2")
		config = Config{Version: Version2}
		pluginConfig = PluginConfig{}
		expectedConfig = Config{
			Version: Version2,
			Plugins: map[string]interface{}{
				"plugin-x": map[string]interface{}{
					"data-1": "",
					"data-2": "",
				},
			},
		}
		Expect(config.EncodePluginConfig(key, pluginConfig)).NotTo(HaveOccurred())
		Expect(config).To(Equal(expectedConfig))

		By("Using config version 2 with extra fields as struct")
		config = Config{Version: Version2}
		pluginConfig = PluginConfig{
			"plugin value 1",
			"plugin value 2",
		}
		expectedConfig = Config{
			Version: Version2,
			Plugins: map[string]interface{}{
				"plugin-x": map[string]interface{}{
					"data-1": "plugin value 1",
					"data-2": "plugin value 2",
				},
			},
		}
		Expect(config.EncodePluginConfig(key, pluginConfig)).NotTo(HaveOccurred())
		Expect(config).To(Equal(expectedConfig))
	})

	It("should decode correctly", func() {
		var (
			key                  = "plugin-x"
			config               Config
			pluginConfig         PluginConfig
			expectedPluginConfig PluginConfig
		)

		By("Using config version 1")
		config = Config{Version: Version1}
		pluginConfig = PluginConfig{}
		Expect(config.DecodePluginConfig(key, &pluginConfig)).To(HaveOccurred())

		By("Using config version 1 with extra fields")
		config = Config{Version: Version1}
		pluginConfig = PluginConfig{
			Data1: "single plugin datum",
		}
		Expect(config.DecodePluginConfig(key, &pluginConfig)).To(HaveOccurred())

		By("Using empty config version 2")
		config = Config{
			Version: Version2,
			Plugins: map[string]interface{}{
				"plugin-x": map[string]interface{}{},
			},
		}
		pluginConfig = PluginConfig{}
		expectedPluginConfig = PluginConfig{}
		Expect(config.DecodePluginConfig(key, &pluginConfig)).NotTo(HaveOccurred())
		Expect(pluginConfig).To(Equal(expectedPluginConfig))

		By("Using config version 2")
		config = Config{Version: Version2}
		pluginConfig = PluginConfig{}
		expectedPluginConfig = PluginConfig{}
		Expect(config.DecodePluginConfig(key, &pluginConfig)).NotTo(HaveOccurred())
		Expect(pluginConfig).To(Equal(expectedPluginConfig))

		By("Using config version 2 with extra fields as struct")
		config = Config{
			Version: Version2,
			Plugins: map[string]interface{}{
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
		Expect(config.DecodePluginConfig(key, &pluginConfig)).NotTo(HaveOccurred())
		Expect(pluginConfig).To(Equal(expectedPluginConfig))
	})
})
