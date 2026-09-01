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
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
)

var _ = Describe("Root command utilities", func() {
	Describe("isHelpFlagArg", func() {
		DescribeTable("should return true for help flags",
			func(arg string) {
				Expect(isHelpFlagArg(arg)).To(BeTrue())
			},
			Entry("--help", "--help"),
			Entry("-h", "-h"),
			Entry("--help=true", "--help=true"),
			Entry("--help=1", "--help=1"),
			Entry("-h=true", "-h=true"),
			Entry("-h=1", "-h=1"),
		)

		DescribeTable("should return false for non-help flags",
			func(arg string) {
				Expect(isHelpFlagArg(arg)).To(BeFalse())
			},
			Entry("--help=false", "--help=false"),
			Entry("--help=0", "--help=0"),
			Entry("-h=false", "-h=false"),
			Entry("-h=0", "-h=0"),
			Entry("--help=invalid", "--help=invalid"),
			Entry("--domain", "--domain"),
			Entry("example.com", "example.com"),
			Entry("", ""),
		)
	})

	Describe("isHelpFlag", func() {
		DescribeTable("should return true for help indicators",
			func(s string) {
				Expect(isHelpFlag(s)).To(BeTrue())
			},
			Entry("--help", "--help"),
			Entry("-h", "-h"),
			Entry("help", kubebuilderSubcommandHelp),
		)

		It("should return false for non-help strings", func() {
			Expect(isHelpFlag("init")).To(BeFalse())
			Expect(isHelpFlag("create")).To(BeFalse())
			Expect(isHelpFlag("foo")).To(BeFalse())
		})
	})

	Describe("isCompletionRequest", func() {
		It("should return true for shell completion commands", func() {
			root := &cobra.Command{Use: "root"}
			cmd1 := &cobra.Command{Use: cobra.ShellCompRequestCmd}
			root.AddCommand(cmd1)
			Expect(isCompletionRequest(cmd1)).To(BeTrue())

			cmd2 := &cobra.Command{Use: cobra.ShellCompNoDescRequestCmd}
			root.AddCommand(cmd2)
			Expect(isCompletionRequest(cmd2)).To(BeTrue())
		})

		It("should return false for regular commands", func() {
			root := &cobra.Command{Use: "root"}
			cmd := &cobra.Command{Use: "init"}
			root.AddCommand(cmd)
			Expect(isCompletionRequest(cmd)).To(BeFalse())
		})
	})

	Describe("subcommandPath", func() {
		It("should return the path from root to the command", func() {
			root := &cobra.Command{Use: kubebuilderCommandName}
			create := &cobra.Command{Use: createSubcommand}
			api := &cobra.Command{Use: apiSubcommand}

			root.AddCommand(create)
			create.AddCommand(api)

			path := subcommandPath(api)
			Expect(path).To(Equal([]string{createSubcommand, apiSubcommand}))
		})

		It("should return empty slice for root command", func() {
			root := &cobra.Command{Use: kubebuilderCommandName}
			path := subcommandPath(root)
			Expect(path).To(BeEmpty())
		})

		It("should return single element for direct child", func() {
			root := &cobra.Command{Use: kubebuilderCommandName}
			init := &cobra.Command{Use: kubebuilderSubcommandInit}
			root.AddCommand(init)

			path := subcommandPath(init)
			Expect(path).To(Equal([]string{kubebuilderSubcommandInit}))
		})
	})

	Describe("getShortKey", func() {
		DescribeTable("should shorten plugin keys",
			func(fullKey, expected string) {
				Expect(getShortKey(fullKey)).To(Equal(expected))
			},
			Entry("standard kubebuilder plugin", "go.kubebuilder.io/v4", "go/v4"),
			Entry("kustomize plugin", "kustomize.common.kubebuilder.io/v2", "kustomize/v2"),
			Entry("deploy-image plugin", "deploy-image.go.kubebuilder.io/v1-alpha", "deploy-image/v1-alpha"),
			Entry("external plugin with domain", "foo.example.com/v1", "foo.example/v1"),
			Entry("plugin without version", "go.kubebuilder.io", "go"),
			Entry("simple external plugin", "example.com/v1", "example.com/v1"),
		)
	})

	Describe("getPluginDescription", func() {
		It("should return fallback description", func() {
			Expect(getPluginDescription("foo.example.com/v1")).To(Equal("External or custom plugin"))
		})
	})
})

var _ = Describe("getPluginTableFilteredWithOptions", func() {
	var c *CLI
	var projectVersion config.Version

	BeforeEach(func() {
		projectVersion = config.Version{Number: 3}
		c = &CLI{
			plugins: make(map[string]plugin.Plugin),
		}
	})

	When("no plugins are registered", func() {
		It("should return no plugins message", func() {
			Expect(c.getPluginTableFiltered(nil)).To(Equal("No plugins available for this subcommand"))
		})
	})

	When("plugins are registered", func() {
		BeforeEach(func() {
			c.plugins = makeMapFor(
				newMockPlugin("go.kubebuilder.io", "v4", projectVersion),
				newMockPlugin("kustomize.common.kubebuilder.io", "v2", projectVersion),
			)
		})

		It("should return formatted plugin table", func() {
			table := c.getPluginTableFiltered(nil)
			Expect(table).To(ContainSubstring("KEY"))
			Expect(table).To(ContainSubstring("DESCRIPTION"))
			Expect(table).To(ContainSubstring("go/v4"))
			Expect(table).To(ContainSubstring("kustomize/v2"))
		})

		It("should exclude deprecated plugins", func() {
			c.plugins = makeMapFor(
				newMockPlugin("go.kubebuilder.io", "v4", projectVersion),
				newMockDeprecatedPlugin("old.kubebuilder.io", "v3", "use go/v4 instead", projectVersion),
			)

			table := c.getPluginTableFiltered(nil)
			Expect(table).To(ContainSubstring("go/v4"))
			Expect(table).NotTo(ContainSubstring("old/v3"))
		})

		It("should exclude base.go plugin to avoid duplication", func() {
			c.plugins = makeMapFor(
				newMockPlugin("go.kubebuilder.io", "v4", projectVersion),
				newMockPlugin("base.go.kubebuilder.io", "v4", projectVersion),
			)

			table := c.getPluginTableFiltered(nil)
			Expect(table).To(ContainSubstring("go/v4"))
			Expect(table).NotTo(ContainSubstring("base.go"))
		})

		It("should filter by predicate", func() {
			c.plugins = makeMapFor(
				newMockPlugin("go.kubebuilder.io", "v4", projectVersion),
				newMockPlugin("kustomize.common.kubebuilder.io", "v2", projectVersion),
			)

			table := c.getPluginTableFiltered(func(p plugin.Plugin) bool {
				return p.Name() == "go.kubebuilder.io"
			})
			Expect(table).To(ContainSubstring("go/v4"))
			Expect(table).NotTo(ContainSubstring("kustomize"))
		})

		It("should exclude default scaffold plugins for subcommands", func() {
			table := c.getPluginTableFilteredForSubcommand(func(_ plugin.Plugin) bool {
				return true
			})
			Expect(table).NotTo(ContainSubstring("go/v4"))
			Expect(table).NotTo(ContainSubstring("kustomize"))
		})
	})

	When("plugins implement Describable", func() {
		It("should use plugin description", func() {
			describablePlugin := &mockDescribablePlugin{
				name:     "custom.kubebuilder.io",
				version:  plugin.Version{Number: 1},
				desc:     "Custom plugin for testing",
				projVers: []config.Version{projectVersion},
			}
			c.plugins = makeMapFor(describablePlugin)

			table := c.getPluginTableFiltered(nil)
			Expect(table).To(ContainSubstring("Custom plugin for testing"))
		})
	})
})

var _ = Describe("newRootCmd", func() {
	var c *CLI
	var projectVersion config.Version

	BeforeEach(func() {
		projectVersion = config.Version{Number: 3}
	})

	When("CLI is properly configured", func() {
		BeforeEach(func() {
			var err error
			c, err = New(
				WithPlugins(&golangv4.Plugin{}),
				WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
				WithDefaultProjectVersion(projectVersion),
				WithVersion("test-version"),
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create root command with correct use", func() {
			cmd := c.newRootCmd()
			Expect(cmd.Use).To(Equal(c.commandName))
		})

		It("should have plugins flag", func() {
			cmd := c.newRootCmd()
			flag := cmd.PersistentFlags().Lookup(pluginsFlag)
			Expect(flag).NotTo(BeNil())
		})

		It("should have project-version flag", func() {
			cmd := c.newRootCmd()
			flag := cmd.Flags().Lookup(projectVersionFlag)
			Expect(flag).NotTo(BeNil())
		})

		It("should allow unknown flags", func() {
			cmd := c.newRootCmd()
			Expect(cmd.FParseErrWhitelist.UnknownFlags).To(BeTrue())
		})

		It("should include plugin table in examples", func() {
			cmd := c.newRootCmd()
			Expect(cmd.Example).To(ContainSubstring("Available plugins"))
		})

		It("should show default plugin in examples", func() {
			cmd := c.newRootCmd()
			Expect(cmd.Example).To(ContainSubstring("Default plugin"))
			Expect(cmd.Example).To(ContainSubstring(pluginGoKubebuilderV4))
		})
	})
})

type mockDescribablePlugin struct {
	name     string
	version  plugin.Version
	desc     string
	projVers []config.Version
}

func (p *mockDescribablePlugin) Name() string                               { return p.name }
func (p *mockDescribablePlugin) Version() plugin.Version                    { return p.version }
func (p *mockDescribablePlugin) SupportedProjectVersions() []config.Version { return p.projVers }
func (p *mockDescribablePlugin) Description() string                        { return p.desc }
