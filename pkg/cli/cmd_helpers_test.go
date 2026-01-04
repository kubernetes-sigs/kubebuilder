/*
Copyright 2025 The Kubernetes Authors.

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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ = Describe("cmd_helpers", func() {
	Context("error types", func() {
		It("noResolvedPluginError should return correct message", func() {
			err := noResolvedPluginError{}
			Expect(err.Error()).To(ContainSubstring("no resolved plugin"))
			Expect(err.Error()).To(ContainSubstring("verify the project version and plugins"))
		})

		It("noAvailablePluginError should return correct message with subcommand", func() {
			err := noAvailablePluginError{subcommand: "init"}
			Expect(err.Error()).To(ContainSubstring("init"))
			Expect(err.Error()).To(ContainSubstring("do not provide any"))
		})
	})

	Context("cmdErr", func() {
		It("should update command with error information", func() {
			cmd := &cobra.Command{
				Long: "Original description",
				RunE: func(*cobra.Command, []string) error {
					return nil
				},
			}
			testError := errors.New("test error")

			cmdErr(cmd, testError)

			Expect(cmd.Long).To(ContainSubstring("Original description"))
			Expect(cmd.Long).To(ContainSubstring("test error"))
			Expect(cmd.RunE).NotTo(BeNil())

			err := cmd.RunE(cmd, []string{})
			Expect(err).To(Equal(testError))
		})
	})

	Context("errCmdFunc", func() {
		It("should return a function that returns the provided error", func() {
			testError := errors.New("test error")
			runE := errCmdFunc(testError)

			err := runE(nil, nil)
			Expect(err).To(Equal(testError))
		})
	})

	Context("moveKeyToFront", func() {
		It("should handle empty chain", func() {
			result := moveKeyToFront([]string{}, "key1")
			Expect(result).To(Equal([]string{"key1"}))
		})

		It("should not change chain when key is already at front", func() {
			chain := []string{"key1", "key2", "key3"}
			result := moveKeyToFront(chain, "key1")
			Expect(result).To(Equal(chain))
		})

		It("should move key to front when it exists in chain", func() {
			chain := []string{"key1", "key2", "key3"}
			result := moveKeyToFront(chain, "key2")
			Expect(result).To(Equal([]string{"key2", "key1", "key3"}))
		})

		It("should move key to front from end of chain", func() {
			chain := []string{"key1", "key2", "key3"}
			result := moveKeyToFront(chain, "key3")
			Expect(result).To(Equal([]string{"key3", "key1", "key2"}))
		})

		It("should add key to front when not in chain", func() {
			chain := []string{"key1", "key2"}
			result := moveKeyToFront(chain, "key3")
			Expect(result).To(Equal([]string{"key3", "key1", "key2"}))
		})

		It("should remove duplicate when moving key to front", func() {
			chain := []string{"key1", "key2", "key2"}
			result := moveKeyToFront(chain, "key2")
			Expect(result).To(Equal([]string{"key2", "key1"}))
		})
	})

	Context("equalStringSlices", func() {
		It("should return true for equal slices", func() {
			a := []string{"a", "b", "c"}
			b := []string{"a", "b", "c"}
			Expect(equalStringSlices(a, b)).To(BeTrue())
		})

		It("should return true for empty slices", func() {
			Expect(equalStringSlices([]string{}, []string{})).To(BeTrue())
		})

		It("should return false for different lengths", func() {
			a := []string{"a", "b"}
			b := []string{"a", "b", "c"}
			Expect(equalStringSlices(a, b)).To(BeFalse())
		})

		It("should return false for different content", func() {
			a := []string{"a", "b", "c"}
			b := []string{"a", "x", "c"}
			Expect(equalStringSlices(a, b)).To(BeFalse())
		})

		It("should return false for different order", func() {
			a := []string{"a", "b", "c"}
			b := []string{"c", "b", "a"}
			Expect(equalStringSlices(a, b)).To(BeFalse())
		})

		It("should handle nil slices", func() {
			var a, b []string
			Expect(equalStringSlices(a, b)).To(BeTrue())
		})
	})

	Context("collectSubcommands", func() {
		var (
			testPlugin       *mockPluginWithSubcommand
			testSubcommand   *mockTestSubcommand
			testBundle       *mockPluginBundle
			testNestedPlugin *mockPluginWithSubcommand
		)

		BeforeEach(func() {
			testSubcommand = &mockTestSubcommand{}
			testPlugin = newMockPluginWithSubcommand(
				"test.plugin", []config.Version{{Number: 1}}, testSubcommand)
			testNestedPlugin = newMockPluginWithSubcommand(
				"nested.plugin", []config.Version{{Number: 1}}, &mockTestSubcommand{})
			testBundle = newMockPluginBundle(
				"test.bundle", []config.Version{{Number: 1}}, []plugin.Plugin{testNestedPlugin})
		})

		It("should return nil when filter returns false", func() {
			filter := func(plugin.Plugin) bool { return false }
			extract := func(plugin.Plugin) plugin.Subcommand { return testSubcommand }

			result := collectSubcommands(testPlugin, "config.key", filter, extract)
			Expect(result).To(BeNil())
		})

		It("should collect subcommand from single plugin", func() {
			filter := func(plugin.Plugin) bool { return true }
			extract := func(p plugin.Plugin) plugin.Subcommand {
				if mp, ok := p.(*mockPluginWithSubcommand); ok {
					return mp.subcommand
				}
				return nil
			}

			result := collectSubcommands(testPlugin, "config.key", filter, extract)
			Expect(result).To(HaveLen(1))
			Expect(result[0].key).To(Equal("test.plugin/v1"))
			Expect(result[0].configKey).To(Equal("config.key"))
			Expect(result[0].subcommand).To(Equal(testSubcommand))
		})

		It("should collect subcommands from bundle", func() {
			filter := func(plugin.Plugin) bool { return true }
			extract := func(p plugin.Plugin) plugin.Subcommand {
				if mp, ok := p.(*mockPluginWithSubcommand); ok {
					return mp.subcommand
				}
				return nil
			}

			result := collectSubcommands(testBundle, "bundle.key", filter, extract)
			Expect(result).To(HaveLen(1))
			Expect(result[0].key).To(Equal("nested.plugin/v1"))
			Expect(result[0].configKey).To(Equal("bundle.key"))
		})
	})

	Context("filterSubcommands", func() {
		var (
			cli            *CLI
			testPlugin1    *mockPluginWithSubcommand
			testPlugin2    *mockPluginWithSubcommand
			testSubcommand *mockTestSubcommand
		)

		BeforeEach(func() {
			testSubcommand = &mockTestSubcommand{}
			testPlugin1 = newMockPluginWithSubcommand("plugin1", []config.Version{{Number: 1}}, testSubcommand)
			testPlugin2 = newMockPluginWithSubcommand("plugin2", []config.Version{{Number: 1}}, testSubcommand)

			cli = &CLI{
				resolvedPlugins: []plugin.Plugin{testPlugin1, testPlugin2},
			}
		})

		It("should filter and extract subcommands from all plugins", func() {
			filter := func(p plugin.Plugin) bool {
				return p.Name() == "plugin1"
			}
			extract := func(p plugin.Plugin) plugin.Subcommand {
				if mp, ok := p.(*mockPluginWithSubcommand); ok {
					return mp.subcommand
				}
				return nil
			}

			result := cli.filterSubcommands(filter, extract)
			Expect(result).To(HaveLen(1))
			Expect(result[0].key).To(Equal("plugin1/v1"))
		})

		It("should return all subcommands when filter allows all", func() {
			filter := func(plugin.Plugin) bool { return true }
			extract := func(p plugin.Plugin) plugin.Subcommand {
				if mp, ok := p.(*mockPluginWithSubcommand); ok {
					return mp.subcommand
				}
				return nil
			}

			result := cli.filterSubcommands(filter, extract)
			Expect(result).To(HaveLen(2))
		})

		It("should return empty when filter rejects all", func() {
			filter := func(plugin.Plugin) bool { return false }
			extract := func(plugin.Plugin) plugin.Subcommand { return testSubcommand }

			result := cli.filterSubcommands(filter, extract)
			Expect(result).To(BeEmpty())
		})
	})
})

type mockTestSubcommand struct{}

func (m *mockTestSubcommand) Scaffold(machinery.Filesystem) error {
	return nil
}

type mockPluginWithSubcommand struct {
	name                     string
	supportedProjectVersions []config.Version
	subcommand               plugin.Subcommand
}

func newMockPluginWithSubcommand(
	name string,
	versions []config.Version,
	subcommand plugin.Subcommand,
) *mockPluginWithSubcommand {
	return &mockPluginWithSubcommand{
		name:                     name,
		supportedProjectVersions: versions,
		subcommand:               subcommand,
	}
}

func (m *mockPluginWithSubcommand) Name() string {
	return m.name
}

func (m *mockPluginWithSubcommand) Version() plugin.Version {
	return plugin.Version{Number: 1}
}

func (m *mockPluginWithSubcommand) SupportedProjectVersions() []config.Version {
	return m.supportedProjectVersions
}

type mockPluginBundle struct {
	name                     string
	supportedProjectVersions []config.Version
	plugins                  []plugin.Plugin
}

func newMockPluginBundle(name string, versions []config.Version, plugins []plugin.Plugin) *mockPluginBundle {
	return &mockPluginBundle{
		name:                     name,
		supportedProjectVersions: versions,
		plugins:                  plugins,
	}
}

func (m *mockPluginBundle) Name() string {
	return m.name
}

func (m *mockPluginBundle) Version() plugin.Version {
	return plugin.Version{Number: 1}
}

func (m *mockPluginBundle) SupportedProjectVersions() []config.Version {
	return m.supportedProjectVersions
}

func (m *mockPluginBundle) Plugins() []plugin.Plugin {
	return m.plugins
}
