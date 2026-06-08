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
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ = Describe("init", func() {
	Context("getInitHelpExamples", func() {
		It("should return help examples for available project versions", func() {
			plugin1 := newMockPlugin("test.plugin", "3", config.Version{Number: 3})
			plugin2 := newMockPlugin("another.plugin", "3", config.Version{Number: 3}, config.Version{Number: 4})

			cli := CLI{
				commandName: kubebuilderCommandName,
				plugins: map[string]plugin.Plugin{
					plugin.KeyFor(plugin1): plugin1,
					plugin.KeyFor(plugin2): plugin2,
				},
			}

			examples := cli.getInitHelpExamples()

			Expect(examples).To(ContainSubstring("kubebuilder init"))
			Expect(examples).To(ContainSubstring("project-version"))
			Expect(examples).To(ContainSubstring("--plugins go/v4,helm/v2-alpha"))
		})

		It("should handle multiple versions", func() {
			plugin1 := newMockPlugin("plugin1", "3", config.Version{Number: 10})
			plugin2 := newMockPlugin("plugin2", "3", config.Version{Number: 3, Stage: stage.Alpha})

			cli := CLI{
				commandName: "kb",
				plugins: map[string]plugin.Plugin{
					plugin.KeyFor(plugin1): plugin1,
					plugin.KeyFor(plugin2): plugin2,
				},
			}

			examples := cli.getInitHelpExamples()

			Expect(examples).To(ContainSubstring("kb init"))
			Expect(examples).To(ContainSubstring("--project-version 10"))
			Expect(examples).NotTo(HaveSuffix("\n\n"))
		})

		It("should omit project version example when no version is available", func() {
			cli := CLI{
				commandName: kubebuilderCommandName,
				plugins:     map[string]plugin.Plugin{},
			}

			examples := cli.getInitHelpExamples()
			Expect(examples).To(ContainSubstring("kubebuilder init --domain example.org"))
			Expect(examples).NotTo(ContainSubstring("--project-version 0"))
		})
	})

	Context("flag descriptions", func() {
		It("should use the same project version description for root and init", func() {
			cli := CLI{
				commandName:           "kubebuilder",
				defaultProjectVersion: config.Version{Number: 3},
				plugins:               map[string]plugin.Plugin{},
			}

			rootCmd := cli.newRootCmd()
			initCmd := cli.newInitCmd()

			Expect(rootCmd.Flags().Lookup(projectVersionFlag).Usage).To(Equal(projectVersionFlagDescription))
			Expect(initCmd.Flags().Lookup(projectVersionFlag).Usage).To(Equal(projectVersionFlagDescription))
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

			Expect(versions).To(ContainElement(config.Version{Number: 3}))
			Expect(versions).To(ContainElement(config.Version{Number: 4}))
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

			Expect(versions).NotTo(ContainElement(config.Version{Number: 2}))
			Expect(versions).To(ContainElement(config.Version{Number: 3}))
		})

		It("should return versions sorted by project version order", func() {
			plugin1 := newMockPlugin("plugin1", "3", config.Version{Number: 10})
			plugin2 := newMockPlugin("plugin2", "3", config.Version{Number: 2})
			plugin3 := newMockPlugin("plugin3", "3", config.Version{Number: 3})
			plugin4 := newMockPlugin("plugin4", "3", config.Version{Number: 3, Stage: stage.Alpha})

			cli := CLI{
				plugins: map[string]plugin.Plugin{
					plugin.KeyFor(plugin1): plugin1,
					plugin.KeyFor(plugin2): plugin2,
					plugin.KeyFor(plugin3): plugin3,
					plugin.KeyFor(plugin4): plugin4,
				},
			}

			versions := cli.getAvailableProjectVersions()

			Expect(versions).To(Equal([]config.Version{
				{Number: 2},
				{Number: 3, Stage: stage.Alpha},
				{Number: 3},
				{Number: 10},
			}))
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
			Expect(versions).To(ContainElement(config.Version{Number: 2}))
			Expect(versions).To(ContainElement(config.Version{Number: 3}))
			Expect(versions).To(ContainElement(config.Version{Number: 4}))
		})
	})
})
