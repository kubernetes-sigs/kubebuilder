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

package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ = Describe("init", func() {
	Context("getInitHelpExamples", func() {
		It("should return help examples for available project versions", func() {
			plugin1 := newMockPlugin("test.plugin", "3", config.Version{Number: 3})
			plugin2 := newMockPlugin("another.plugin", "3", config.Version{Number: 3}, config.Version{Number: 4})

			cli := CLI{
				commandName: "kubebuilder",
				plugins: map[string]plugin.Plugin{
					plugin.KeyFor(plugin1): plugin1,
					plugin.KeyFor(plugin2): plugin2,
				},
			}

			examples := cli.getInitHelpExamples()

			Expect(examples).To(ContainSubstring("kubebuilder init"))
			Expect(examples).To(ContainSubstring("project-version"))
			Expect(examples).To(ContainSubstring("-h"))
		})

		It("should handle multiple versions", func() {
			plugin1 := newMockPlugin("plugin1", "3", config.Version{Number: 2})
			plugin2 := newMockPlugin("plugin2", "3", config.Version{Number: 3})

			cli := CLI{
				commandName: "kb",
				plugins: map[string]plugin.Plugin{
					plugin.KeyFor(plugin1): plugin1,
					plugin.KeyFor(plugin2): plugin2,
				},
			}

			examples := cli.getInitHelpExamples()

			Expect(examples).To(ContainSubstring("kb init"))
			Expect(examples).NotTo(HaveSuffix("\n\n"))
		})

		It("should return empty string when no plugins", func() {
			cli := CLI{
				commandName: "kubebuilder",
				plugins:     map[string]plugin.Plugin{},
			}

			examples := cli.getInitHelpExamples()
			Expect(examples).To(BeEmpty())
		})
	})

	Context("getAvailableProjectVersions", func() {
		It("should return unique project versions", func() {
			plugin1 := newMockPlugin("plugin1", "3", config.Version{Number: 3})
			plugin2 := newMockPlugin("plugin2", "3", config.Version{Number: 3})
			plugin3 := newMockPlugin("plugin3", "3", config.Version{Number: 4})

			cli := CLI{
				plugins: map[string]plugin.Plugin{
					plugin.KeyFor(plugin1): plugin1,
					plugin.KeyFor(plugin2): plugin2,
					plugin.KeyFor(plugin3): plugin3,
				},
			}

			versions := cli.getAvailableProjectVersions()

			Expect(versions).To(ContainElement("\"3\""))
			Expect(versions).To(ContainElement("\"4\""))
			Expect(versions).To(HaveLen(2))
		})

		It("should exclude deprecated plugins", func() {
			deprecatedPlugin := newMockDeprecatedPlugin("deprecated", "3", "use v4 instead", config.Version{Number: 2})
			regularPlugin := newMockPlugin("regular", "3", config.Version{Number: 3})

			cli := CLI{
				plugins: map[string]plugin.Plugin{
					plugin.KeyFor(deprecatedPlugin): deprecatedPlugin,
					plugin.KeyFor(regularPlugin):    regularPlugin,
				},
			}

			versions := cli.getAvailableProjectVersions()

			Expect(versions).NotTo(ContainElement("\"2\""))
			Expect(versions).To(ContainElement("\"3\""))
		})

		It("should return sorted versions", func() {
			plugin1 := newMockPlugin("plugin1", "3", config.Version{Number: 4})
			plugin2 := newMockPlugin("plugin2", "3", config.Version{Number: 2})
			plugin3 := newMockPlugin("plugin3", "3", config.Version{Number: 3})

			cli := CLI{
				plugins: map[string]plugin.Plugin{
					plugin.KeyFor(plugin1): plugin1,
					plugin.KeyFor(plugin2): plugin2,
					plugin.KeyFor(plugin3): plugin3,
				},
			}

			versions := cli.getAvailableProjectVersions()

			Expect(versions).To(Equal([]string{"\"2\"", "\"3\"", "\"4\""}))
		})

		It("should return empty slice when no plugins", func() {
			cli := CLI{
				plugins: map[string]plugin.Plugin{},
			}

			versions := cli.getAvailableProjectVersions()
			Expect(versions).To(BeEmpty())
		})

		It("should handle plugin with multiple supported versions", func() {
			multiPlugin := newMockPlugin("multi", "3",
				config.Version{Number: 2},
				config.Version{Number: 3},
				config.Version{Number: 4})

			cli := CLI{
				plugins: map[string]plugin.Plugin{
					plugin.KeyFor(multiPlugin): multiPlugin,
				},
			}

			versions := cli.getAvailableProjectVersions()

			Expect(versions).To(HaveLen(3))
			Expect(versions).To(ContainElement("\"2\""))
			Expect(versions).To(ContainElement("\"3\""))
			Expect(versions).To(ContainElement("\"4\""))
		})
	})
})
