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
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
)

var _ = Describe("isHelpFlagArg", func() {
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

var _ = Describe("isHelpFlag", func() {
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

var _ = Describe("isCompletionRequest", func() {
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

var _ = Describe("subcommandPath", func() {
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

var _ = Describe("getShortKey", func() {
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

var _ = Describe("getPluginDescription", func() {
	It("should return fallback description", func() {
		Expect(getPluginDescription("foo.example.com/v1")).To(Equal("External or custom plugin"))
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

		It("should register the plugins flag", func() {
			cmd := c.newRootCmd()
			flag := cmd.PersistentFlags().Lookup(pluginsFlag)
			Expect(flag).NotTo(BeNil())
		})

		It("should register the project-version flag", func() {
			cmd := c.newRootCmd()
			flag := cmd.Flags().Lookup(projectVersionFlag)
			Expect(flag).NotTo(BeNil())
		})

		It("should allow unknown flags to prevent parsing errors during help display", func() {
			cmd := c.newRootCmd()
			Expect(cmd.FParseErrWhitelist.UnknownFlags).To(BeTrue())
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

// The steps an aliased command records when they run.
const (
	stepArgs = "args"
	stepHook = "hook"
	stepBody = "body"
)

// aliasedCommand records which of its steps ran: the argument check, the persistent hook and the
// body.
type aliasedCommand struct {
	cmd *cobra.Command
	ran []string
}

func newAliasedCommand(use, alias string) *aliasedCommand {
	a := &aliasedCommand{}
	a.cmd = &cobra.Command{
		Use:     use,
		Aliases: []string{alias},
		Args: func(*cobra.Command, []string) error {
			a.ran = append(a.ran, stepArgs)

			return nil
		},
		PersistentPreRunE: func(*cobra.Command, []string) error {
			a.ran = append(a.ran, stepHook)

			return nil
		},
		RunE: func(*cobra.Command, []string) error {
			a.ran = append(a.ran, stepBody)

			return nil
		},
	}

	return a
}

// addChild gives the command a subcommand with its own record of its argument check and body. It sets
// no persistent hook, so the parent's runs for it.
func (a *aliasedCommand) addChild(use string) *aliasedCommand {
	child := &aliasedCommand{}
	child.cmd = &cobra.Command{
		Use: use,
		Args: func(*cobra.Command, []string) error {
			child.ran = append(child.ran, stepArgs)

			return nil
		},
		RunE: func(*cobra.Command, []string) error {
			child.ran = append(child.ran, stepBody)

			return nil
		},
	}
	a.cmd.AddCommand(child.cmd)

	return child
}

// aliasShapes are the two tool commands the specs use: a command named version that also answers
// to docs, and a command named docs that also answers to version.
var aliasShapes = [][2]string{
	{kubebuilderSubcommandVersion, docsSubcommand},
	{docsSubcommand, kubebuilderSubcommandVersion},
}

// newToolCLI builds a CLI the way a tool does when it brings its own top-level command and no
// version command of the CLI's, while the program was invoked with programArgs.
func newToolCLI(filesystem machinery.Filesystem, programArgs []string, extra *cobra.Command) *CLI {
	GinkgoHelper()

	setProgramArgs(programArgs...)
	projectVersion := config.Version{Number: 3}
	c, err := New(
		WithPlugins(&golangv4.Plugin{}),
		WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
		WithDefaultProjectVersion(projectVersion),
		WithFilesystem(filesystem),
		WithExtraCommands(extra),
	)
	Expect(err).NotTo(HaveOccurred())
	c.cmd.SetOut(io.Discard)
	c.cmd.SetErr(io.Discard)

	return c
}

// A tool can give its own top-level command an alias. The command cobra resolves decides whether
// it runs without the PROJECT file, whichever of its names the user typed: a tool's version runs
// under its alias too, and a tool's docs that also answers to version is still docs.
var _ = Describe("a command a tool adds with an alias", func() {
	type run func(filesystem machinery.Filesystem, extra *cobra.Command, typed ...string) error

	// throughRun is a tool that runs the CLI with the words it was started with.
	throughRun := func(filesystem machinery.Filesystem, extra *cobra.Command, typed ...string) error {
		return newToolCLI(filesystem, typed, extra).Run()
	}
	// throughExecute is a tool that was started with other words and tells cobra what to run.
	throughExecute := func(filesystem machinery.Filesystem, extra *cobra.Command, typed ...string) error {
		c := newToolCLI(filesystem, []string{createSubcommand, apiSubcommand}, extra)
		c.cmd.SetArgs(typed)

		return c.cmd.Execute()
	}

	DescribeTable("runs a tool's version on an unusable PROJECT whichever name the user types",
		func(execute run, filesystem func() machinery.Filesystem) {
			for _, typed := range []string{kubebuilderSubcommandVersion, docsSubcommand} {
				extra := newAliasedCommand(kubebuilderSubcommandVersion, docsSubcommand)
				Expect(execute(filesystem(), extra.cmd, typed)).To(Succeed(), "version typed as %q", typed)
				Expect(extra.ran).To(Equal([]string{stepArgs, stepHook, stepBody}), "version typed as %q", typed)
			}
		},
		Entry("through Run, with a directory at the path", throughRun, filesystemWithProjectDirectory),
		Entry("through Run, with content that cannot be parsed", throughRun, filesystemWithInvalidProject),
		Entry("through Execute, with a directory at the path", throughExecute, filesystemWithProjectDirectory),
		Entry("through Execute, with content that cannot be parsed", throughExecute, filesystemWithInvalidProject),
	)

	DescribeTable("runs a command under a tool's version on an unusable PROJECT whichever name the user types",
		func(execute run, filesystem func() machinery.Filesystem) {
			for _, typed := range []string{kubebuilderSubcommandVersion, docsSubcommand} {
				extra := newAliasedCommand(kubebuilderSubcommandVersion, docsSubcommand)
				details := extra.addChild("details")
				Expect(execute(filesystem(), extra.cmd, typed, "details")).To(Succeed(),
					"version details typed as %q details", typed)
				Expect(details.ran).To(Equal([]string{stepArgs, stepBody}), "version details typed as %q details", typed)
				Expect(extra.ran).To(Equal([]string{stepHook}), "only the persistent hook of version runs for details")
			}
		},
		Entry("through Run, with a directory at the path", throughRun, filesystemWithProjectDirectory),
		Entry("through Run, with content that cannot be parsed", throughRun, filesystemWithInvalidProject),
		Entry("through Execute, with a directory at the path", throughExecute, filesystemWithProjectDirectory),
		Entry("through Execute, with content that cannot be parsed", throughExecute, filesystemWithInvalidProject),
	)

	DescribeTable("refuses a tool's docs on an unusable PROJECT whichever name the user types",
		func(execute run, filesystem func() machinery.Filesystem) {
			for _, typed := range []string{docsSubcommand, kubebuilderSubcommandVersion} {
				extra := newAliasedCommand(docsSubcommand, kubebuilderSubcommandVersion)
				Expect(execute(filesystem(), extra.cmd, typed)).NotTo(Succeed(), "docs typed as %q", typed)
				Expect(extra.ran).To(BeEmpty(), "docs typed as %q must not run", typed)
			}
		},
		Entry("through Run, with a directory at the path", throughRun, filesystemWithProjectDirectory),
		Entry("through Run, with content that cannot be parsed", throughRun, filesystemWithInvalidProject),
		Entry("through Execute, with a directory at the path", throughExecute, filesystemWithProjectDirectory),
		Entry("through Execute, with content that cannot be parsed", throughExecute, filesystemWithInvalidProject),
	)

	// The program is started with the alias, so the CLI decides from the first word whether to read
	// PROJECT before it builds the command tree. The word names the tool's docs, which needs the
	// configuration, so the tree is built for the plugins the layout names.
	It("runs a tool's docs typed as its alias version on a usable PROJECT whose layout names other plugins", func() {
		projectPlugin := newTestCreateAPIPlugin(projectChainPluginName, plugin.Version{Number: 1})
		typed := []string{kubebuilderSubcommandVersion}

		throughRun := newAliasedCommand(docsSubcommand, kubebuilderSubcommandVersion)
		Expect(buildCLI(filesystemWithProjectChainProject(), typed,
			WithExtraCommands(throughRun.cmd), WithPlugins(projectPlugin)).Run()).To(Succeed())
		Expect(throughRun.ran).To(Equal([]string{stepArgs, stepHook, stepBody}))

		throughExecute := newAliasedCommand(docsSubcommand, kubebuilderSubcommandVersion)
		Expect(executeEmbeddedCLI(filesystemWithProjectChainProject(), typed, typed,
			WithExtraCommands(throughExecute.cmd), WithPlugins(projectPlugin))).To(Succeed())
		Expect(throughExecute.ran).To(Equal([]string{stepArgs, stepHook, stepBody}))
	})

	DescribeTable("runs with no PROJECT file whatever the user types",
		func(execute run) {
			for _, typed := range []string{kubebuilderSubcommandVersion, docsSubcommand} {
				for _, shape := range aliasShapes {
					extra := newAliasedCommand(shape[0], shape[1])
					Expect(execute(filesystemWithoutProject(), extra.cmd, typed)).To(Succeed())
					Expect(extra.ran).To(Equal([]string{stepArgs, stepHook, stepBody}))
				}
			}
		},
		Entry("through Run", throughRun),
		Entry("through Execute", throughExecute),
	)
})
