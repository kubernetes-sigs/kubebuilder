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
	"github.com/spf13/pflag"

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

	Context("duplicate flag handling (mergeFlagSetInto, syncDuplicateFlags)", func() {
		It("should not panic when merging two FlagSets that define the same flag name (same type)", func() {
			dest := pflag.NewFlagSet("dest", pflag.ExitOnError)
			src := pflag.NewFlagSet("src", pflag.ExitOnError)
			duplicateValues := make(map[string][]pflag.Value)
			firstPluginByFlag := make(map[string]string)

			var destBool bool
			var srcBool bool
			dest.BoolVar(&destBool, "force", false, "overwrite files (plugin A)")
			src.BoolVar(&srcBool, "force", false, "regenerate all files (plugin B)")

			err := mergeFlagSetInto(dest, src, duplicateValues, "pluginB/v1", firstPluginByFlag)
			Expect(err).NotTo(HaveOccurred())
			Expect(dest.Lookup("force")).NotTo(BeNil())
			Expect(duplicateValues["force"]).To(HaveLen(1))
		})

		It("should aggregate help text as For plugin (key): desc AND for plugin (key): desc", func() {
			dest := pflag.NewFlagSet("dest", pflag.ExitOnError)
			src := pflag.NewFlagSet("src", pflag.ExitOnError)
			duplicateValues := make(map[string][]pflag.Value)
			firstPluginByFlag := make(map[string]string)

			var a, b bool
			dest.BoolVar(&a, "force", false, "overwrite files (plugin A)")
			src.BoolVar(&b, "force", false, "regenerate all files (plugin B)")

			err := mergeFlagSetInto(dest, src, duplicateValues, "pluginB/v1", firstPluginByFlag)
			Expect(err).NotTo(HaveOccurred())

			flag := dest.Lookup("force")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).To(ContainSubstring("overwrite files (plugin A)"))
			Expect(flag.Usage).To(ContainSubstring("AND for plugin (pluginB/v1):"))
			Expect(flag.Usage).To(ContainSubstring("regenerate all files (plugin B)"))
		})

		It("should prefix first plugin with For plugin (key): when both flags merged via mergeFlagSetInto", func() {
			dest := pflag.NewFlagSet("dest", pflag.ExitOnError)
			pluginA := pflag.NewFlagSet("a", pflag.ExitOnError)
			pluginB := pflag.NewFlagSet("b", pflag.ExitOnError)
			duplicateValues := make(map[string][]pflag.Value)
			firstPluginByFlag := make(map[string]string)

			var a, b bool
			pluginA.BoolVar(&a, "force", false, "overwrite files (plugin A)")
			pluginB.BoolVar(&b, "force", false, "regenerate all files (plugin B)")

			Expect(mergeFlagSetInto(dest, pluginA, duplicateValues, "pluginA/v1", firstPluginByFlag)).NotTo(HaveOccurred())
			Expect(mergeFlagSetInto(dest, pluginB, duplicateValues, "pluginB/v1", firstPluginByFlag)).NotTo(HaveOccurred())

			flag := dest.Lookup("force")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Usage).To(Equal(
				"For plugin (pluginA/v1): overwrite files (plugin A) AND for plugin (pluginB/v1): regenerate all files (plugin B)"))
		})

		It("should show full plugin keys in aggregated usage", func() {
			dest := pflag.NewFlagSet("dest", pflag.ExitOnError)
			goPlugin := pflag.NewFlagSet("go", pflag.ExitOnError)
			helmPlugin := pflag.NewFlagSet("helm", pflag.ExitOnError)
			duplicateValues := make(map[string][]pflag.Value)
			firstPluginByFlag := make(map[string]string)

			var a, b bool
			goPlugin.BoolVar(&a, "force", false, "overwrite scaffolded files to apply changes (manual edits may be lost)")
			helmPlugin.BoolVar(&b, "force", false, "if true, regenerates all the files")

			Expect(mergeFlagSetInto(dest, goPlugin, duplicateValues, "base.go.kubebuilder.io/v4", firstPluginByFlag)).
				NotTo(HaveOccurred())
			Expect(mergeFlagSetInto(dest, helmPlugin, duplicateValues, "helm.kubebuilder.io/v2-alpha", firstPluginByFlag)).
				NotTo(HaveOccurred())

			flag := dest.Lookup("force")
			Expect(flag).NotTo(BeNil())
			expectedUsage := "For plugin (base.go.kubebuilder.io/v4): overwrite scaffolded files to apply changes " +
				"(manual edits may be lost) AND for plugin (helm.kubebuilder.io/v2-alpha): if true, regenerates all the files"
			Expect(flag.Usage).To(Equal(expectedUsage))
		})

		It("should return error when same flag name is bound with different value types", func() {
			dest := pflag.NewFlagSet("dest", pflag.ExitOnError)
			src := pflag.NewFlagSet("src", pflag.ExitOnError)
			duplicateValues := make(map[string][]pflag.Value)
			firstPluginByFlag := make(map[string]string)
			firstPluginByFlag["flag"] = "pluginA/v1" // dest already has this flag from a previous plugin

			var a bool
			var b string
			dest.BoolVar(&a, "flag", false, "bool usage (plugin A)")
			src.StringVar(&b, "flag", "", "string usage (plugin B)")

			err := mergeFlagSetInto(dest, src, duplicateValues, "pluginB/v1", firstPluginByFlag)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("same flag name"))
			Expect(err.Error()).To(ContainSubstring("different value types"))
			Expect(err.Error()).To(ContainSubstring("flag"))
			Expect(err.Error()).To(ContainSubstring("bool"))
			Expect(err.Error()).To(ContainSubstring("string"))
			Expect(err.Error()).To(ContainSubstring("pluginA/v1"))
			Expect(err.Error()).To(ContainSubstring("pluginB/v1"))
		})

		It("should sync parsed value to duplicate Values after syncDuplicateFlags", func() {
			flags := pflag.NewFlagSet("cmd", pflag.ExitOnError)
			var mainVal, dupVal bool
			flags.BoolVar(&mainVal, "force", false, "usage")
			tmpFS := pflag.NewFlagSet("", pflag.ExitOnError)
			tmpFS.BoolVar(&dupVal, "force", false, "")
			duplicateValues := map[string][]pflag.Value{
				"force": {tmpFS.Lookup("force").Value},
			}

			Expect(flags.Parse([]string{"--force", "true"})).NotTo(HaveOccurred())
			Expect(mainVal).To(BeTrue())
			Expect(dupVal).To(BeFalse())

			syncDuplicateFlags(flags, duplicateValues)
			Expect(dupVal).To(BeTrue())
		})

		It("should give all plugins in the chain the same value for a shared flag (e.g. --force)", func() {
			cmdFlags := pflag.NewFlagSet("edit", pflag.ExitOnError)
			pluginA := pflag.NewFlagSet("pluginA", pflag.ExitOnError)
			pluginB := pflag.NewFlagSet("pluginB", pflag.ExitOnError)
			var forceA, forceB bool
			pluginA.BoolVar(&forceA, "force", false, "plugin A force")
			pluginB.BoolVar(&forceB, "force", false, "plugin B force")

			duplicateValues := make(map[string][]pflag.Value)
			firstPluginByFlag := make(map[string]string)
			Expect(mergeFlagSetInto(cmdFlags, pluginA, duplicateValues, "pluginA/v1", firstPluginByFlag)).NotTo(HaveOccurred())
			Expect(mergeFlagSetInto(cmdFlags, pluginB, duplicateValues, "pluginB/v1", firstPluginByFlag)).NotTo(HaveOccurred())

			Expect(cmdFlags.Parse([]string{"--force", "true"})).NotTo(HaveOccurred())
			syncDuplicateFlags(cmdFlags, duplicateValues)
			Expect(forceA).To(BeTrue(), "plugin A must receive the value passed by the user")
			Expect(forceB).To(BeTrue(), "plugin B must receive the same value as the command")
		})

		It("should sync string flag value to duplicate Values", func() {
			flags := pflag.NewFlagSet("cmd", pflag.ExitOnError)
			var mainVal, dupVal string
			flags.StringVar(&mainVal, "name", "", "name usage")
			tmpFS := pflag.NewFlagSet("", pflag.ExitOnError)
			tmpFS.StringVar(&dupVal, "name", "", "")
			duplicateValues := map[string][]pflag.Value{
				"name": {tmpFS.Lookup("name").Value},
			}

			Expect(flags.Parse([]string{"--name", "foo"})).NotTo(HaveOccurred())
			syncDuplicateFlags(flags, duplicateValues)
			Expect(dupVal).To(Equal("foo"))
		})

		It("applies merge and sync for any subcommand (init, api, webhook, edit), not only edit", func() {
			cmd := &cobra.Command{Use: "api"}
			pluginA := &mockSubcommandWithForceFlag{}
			pluginB := &mockSubcommandWithForceFlag{}
			tuples := []keySubcommandTuple{
				{key: "pluginA.kubebuilder.io/v1", subcommand: pluginA},
				{key: "pluginB.kubebuilder.io/v1", subcommand: pluginB},
			}
			meta := plugin.CLIMetadata{}

			result, err := initializationHooks(cmd, tuples, meta)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.duplicateFlagValues["force"]).To(HaveLen(1), "second plugin's Value recorded as duplicate")

			Expect(cmd.ParseFlags([]string{"--force", "true"})).NotTo(HaveOccurred())
			syncDuplicateFlags(cmd.Flags(), result.duplicateFlagValues)
			Expect(pluginA.Force).To(BeTrue(), "first plugin (flag on command) receives value")
			Expect(pluginB.Force).To(BeTrue(), "second plugin (duplicate) receives same value after sync")
		})
	})
})

type mockTestSubcommand struct{}

func (m *mockTestSubcommand) Scaffold(machinery.Filesystem) error {
	return nil
}

// mockSubcommandWithForceFlag implements Subcommand and HasFlags for tests with a shared flag.
type mockSubcommandWithForceFlag struct {
	Force bool
}

func (m *mockSubcommandWithForceFlag) Scaffold(machinery.Filesystem) error {
	return nil
}

func (m *mockSubcommandWithForceFlag) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&m.Force, "force", false, "force usage")
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
