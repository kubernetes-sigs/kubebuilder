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

package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	goPluginV4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
)

func makeMockPluginsFor(projectVersion config.Version, pluginKeys ...string) []plugin.Plugin {
	plugins := make([]plugin.Plugin, 0, len(pluginKeys))
	for _, key := range pluginKeys {
		n, v := plugin.SplitKey(key)
		plugins = append(plugins, newMockPlugin(n, v, projectVersion))
	}
	return plugins
}

func makeMapFor(plugins ...plugin.Plugin) map[string]plugin.Plugin {
	pluginMap := make(map[string]plugin.Plugin, len(plugins))
	for _, p := range plugins {
		pluginMap[plugin.KeyFor(p)] = p
	}
	return pluginMap
}

func setFlag(flag, value string) {
	os.Args = append(os.Args, "subcommand", "--"+flag, value)
}

func setBoolFlag(flag string) {
	os.Args = append(os.Args, "subcommand", "--"+flag)
}

func setProjectVersionFlag(value string) {
	setFlag(projectVersionFlag, value)
}

func setPluginsFlag(value string) {
	setFlag(pluginsFlag, value)
}

func hasSubCommand(cmd *cobra.Command, name string) bool {
	for _, subcommand := range cmd.Commands() {
		if subcommand.Name() == name {
			return true
		}
	}
	return false
}

var _ = Describe("CLI", func() {
	var (
		c              *CLI
		projectVersion = config.Version{Number: 3}
	)

	BeforeEach(func() {
		c = &CLI{
			fs: machinery.Filesystem{FS: afero.NewMemMapFs()},
		}
	})

	Context("buildCmd", func() {
		var projectFile string

		BeforeEach(func() {
			projectFile = `domain: zeusville.com
layout: go.kubebuilder.io/v3
projectName: demo-zeus-operator
repo: github.com/jmrodri/demo-zeus-operator
resources:
- crdVersion: v1
  group: test
  kind: Test
  version: v1
version: 3-alpha
plugins:
  manifests.sdk.operatorframework.io/v2: {}
`
			f, err := c.fs.FS.Create("PROJECT")
			Expect(err).To(Not(HaveOccurred()))

			_, err = f.WriteString(projectFile)
			Expect(err).To(Not(HaveOccurred()))
		})

		When("reading a 3-alpha config", func() {
			It("should succeed and set the projectVersion", func() {
				err := c.buildCmd()
				Expect(err).To(Not(HaveOccurred()))
				Expect(c.projectVersion.Compare(
					config.Version{
						Number: 3,
						Stage:  stage.Stable,
					})).To(Equal(0))
			})
			It("should fail when stable is not registered ", func() {
				// overwrite project file with fake 4-alpha
				f, err := c.fs.FS.OpenFile("PROJECT", os.O_WRONLY, 0)
				Expect(err).To(Not(HaveOccurred()))
				_, err = f.WriteString(strings.ReplaceAll(projectFile, "3-alpha", "4-alpha"))
				Expect(err).To(Not(HaveOccurred()))

				// buildCmd should return an error
				err = c.buildCmd()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	// TODO: test CLI.getInfoFromConfigFile using a mock filesystem

	Context("getInfoFromConfig", func() {
		When("having a single plugin in the layout field", func() {
			It("should succeed", func() {
				pluginChain := []string{"go.kubebuilder.io/v4"}
				projectConfig := cfgv3.New()
				Expect(projectConfig.SetPluginChain(pluginChain)).To(Succeed())

				Expect(c.getInfoFromConfig(projectConfig)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginChain))
				Expect(c.projectVersion.Compare(projectConfig.GetVersion())).To(Equal(0))
			})
		})

		When("having multiple plugins in the layout field", func() {
			It("should succeed", func() {
				pluginChain := []string{"go.kubebuilder.io/v2", "deploy-image.go.kubebuilder.io/v1-alpha"}

				projectConfig := cfgv3.New()
				Expect(projectConfig.SetPluginChain(pluginChain)).To(Succeed())

				Expect(c.getInfoFromConfig(projectConfig)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginChain))
				Expect(c.projectVersion.Compare(projectConfig.GetVersion())).To(Equal(0))
			})
		})

		When("having invalid plugin keys in the layout field", func() {
			It("should fail", func() {
				pluginChain := []string{"_/v1"}

				projectConfig := cfgv3.New()
				Expect(projectConfig.SetPluginChain(pluginChain)).To(Succeed())

				Expect(c.getInfoFromConfig(projectConfig)).NotTo(Succeed())
			})
		})
	})

	Context("getInfoFromFlags", func() {
		// Save os.Args and restore it for every test
		var args []string
		BeforeEach(func() {
			c.cmd = c.newRootCmd()

			args = os.Args
		})
		AfterEach(func() {
			os.Args = args
		})

		When("no flag is set", func() {
			It("should succeed", func() {
				Expect(c.getInfoFromFlags(false)).To(Succeed())
				Expect(c.pluginKeys).To(BeEmpty())
				Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
			})
		})

		When(fmt.Sprintf("--%s flag is set", pluginsFlag), func() {
			It("should succeed using one plugin key", func() {
				pluginKeys := []string{"go/v1"}
				setPluginsFlag(strings.Join(pluginKeys, ","))

				Expect(c.getInfoFromFlags(false)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
			})

			It("should succeed using more than one plugin key", func() {
				pluginKeys := []string{"go/v1", "example/v2", "test/v1"}
				setPluginsFlag(strings.Join(pluginKeys, ","))

				Expect(c.getInfoFromFlags(false)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
			})

			It("should succeed using more than one plugin key with spaces", func() {
				pluginKeys := []string{"go/v1", "example/v2", "test/v1"}
				setPluginsFlag(strings.Join(pluginKeys, ", "))

				Expect(c.getInfoFromFlags(false)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
			})

			It("should fail for an invalid plugin key", func() {
				setPluginsFlag("_/v1")

				Expect(c.getInfoFromFlags(false)).NotTo(Succeed())
			})
		})

		When(fmt.Sprintf("--%s flag is set", projectVersionFlag), func() {
			It("should succeed", func() {
				setProjectVersionFlag(projectVersion.String())

				Expect(c.getInfoFromFlags(false)).To(Succeed())
				Expect(c.pluginKeys).To(BeEmpty())
				Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
			})

			It("should fail for an invalid project version", func() {
				setProjectVersionFlag("v_1")

				Expect(c.getInfoFromFlags(false)).NotTo(Succeed())
			})
		})

		When(fmt.Sprintf("--%s and --%s flags are set", pluginsFlag, projectVersionFlag), func() {
			It("should succeed using one plugin key", func() {
				pluginKeys := []string{"go/v1"}
				setPluginsFlag(strings.Join(pluginKeys, ","))
				setProjectVersionFlag(projectVersion.String())

				Expect(c.getInfoFromFlags(false)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
			})

			It("should succeed using more than one plugin key", func() {
				pluginKeys := []string{"go/v1", "example/v2", "test/v1"}
				setPluginsFlag(strings.Join(pluginKeys, ","))
				setProjectVersionFlag(projectVersion.String())

				Expect(c.getInfoFromFlags(false)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
			})

			It("should succeed using more than one plugin key with spaces", func() {
				pluginKeys := []string{"go/v1", "example/v2", "test/v1"}
				setPluginsFlag(strings.Join(pluginKeys, ", "))
				setProjectVersionFlag(projectVersion.String())

				Expect(c.getInfoFromFlags(false)).To(Succeed())
				Expect(c.pluginKeys).To(Equal(pluginKeys))
				Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
			})
		})

		When("additional flags are set", func() {
			It("should succeed", func() {
				setFlag("extra-flag", "extra-value")

				Expect(c.getInfoFromFlags(false)).To(Succeed())
			})

			// `--help` is not captured by the allowlist, so we need to special case it
			It("should not fail for `--help`", func() {
				setBoolFlag("help")

				Expect(c.getInfoFromFlags(false)).To(Succeed())
			})
		})
	})

	Context("getInfoFromDefaults", func() {
		pluginKeys := []string{"go.kubebuilder.io/v2"}

		It("should be a no-op if already have plugin keys", func() {
			c.pluginKeys = pluginKeys

			c.getInfoFromDefaults()
			Expect(c.pluginKeys).To(Equal(pluginKeys))
			Expect(c.projectVersion.Compare(config.Version{})).To(Equal(0))
		})

		It("should succeed if default plugins for project version are set", func() {
			c.projectVersion = projectVersion
			c.defaultPlugins = map[config.Version][]string{projectVersion: pluginKeys}

			c.getInfoFromDefaults()
			Expect(c.pluginKeys).To(Equal(pluginKeys))
			Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
		})

		It("should succeed if default plugins for default project version are set", func() {
			c.defaultPlugins = map[config.Version][]string{projectVersion: pluginKeys}
			c.defaultProjectVersion = projectVersion

			c.getInfoFromDefaults()
			Expect(c.pluginKeys).To(Equal(pluginKeys))
			Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
		})

		It("should succeed if default plugins for only a single project version are set", func() {
			c.defaultPlugins = map[config.Version][]string{projectVersion: pluginKeys}

			c.getInfoFromDefaults()
			Expect(c.pluginKeys).To(Equal(pluginKeys))
			Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
		})
	})

	Context("resolvePlugins", func() {
		pluginKeys := []string{
			"foo.example.com/v1",
			"bar.example.com/v1",
			"baz.example.com/v1",
			"foo.kubebuilder.io/v1",
			"foo.kubebuilder.io/v2",
			"bar.kubebuilder.io/v1",
			"bar.kubebuilder.io/v2",
		}

		plugins := makeMockPluginsFor(projectVersion, pluginKeys...)
		plugins = append(plugins,
			newMockPlugin("invalid.kubebuilder.io", "v1"),
			newMockPlugin("only1.kubebuilder.io", "v1",
				config.Version{Number: 1}),
			newMockPlugin("only2.kubebuilder.io", "v1",
				config.Version{Number: 2}),
			newMockPlugin("1and2.kubebuilder.io", "v1",
				config.Version{Number: 1}, config.Version{Number: 2}),
			newMockPlugin("2and3.kubebuilder.io", "v1",
				config.Version{Number: 2}, config.Version{Number: 3}),
			newMockPlugin("1-2and3.kubebuilder.io", "v1",
				config.Version{Number: 1}, config.Version{Number: 2}, config.Version{Number: 3}),
		)
		pluginMap := makeMapFor(plugins...)

		BeforeEach(func() {
			c.plugins = pluginMap
		})

		DescribeTable("should resolve",
			func(key, qualified string) {
				c.pluginKeys = []string{key}
				c.projectVersion = projectVersion

				Expect(c.resolvePlugins()).To(Succeed())
				Expect(c.resolvedPlugins).To(HaveLen(1))
				Expect(plugin.KeyFor(c.resolvedPlugins[0])).To(Equal(qualified))
			},
			Entry("fully qualified plugin", "foo.example.com/v1", "foo.example.com/v1"),
			Entry("plugin without version", "foo.example.com", "foo.example.com/v1"),
			Entry("shortname without version", "baz", "baz.example.com/v1"),
			Entry("shortname with version", "foo/v2", "foo.kubebuilder.io/v2"),
		)

		DescribeTable("should not resolve",
			func(key string) {
				c.pluginKeys = []string{key}
				c.projectVersion = projectVersion

				Expect(c.resolvePlugins()).NotTo(Succeed())
			},
			Entry("for an ambiguous version", "foo.kubebuilder.io"),
			Entry("for an ambiguous name", "foo/v1"),
			Entry("for an ambiguous name and version", "foo"),
			Entry("for a non-existent name", "blah"),
			Entry("for a non-existent version", "foo.example.com/v2"),
			Entry("for a non-existent version", "foo/v3"),
			Entry("for a non-existent version", "foo.example.com/v3"),
			Entry("for a plugin that doesn't support the project version", "invalid.kubebuilder.io/v1"),
		)

		It("should succeed if only one common project version is found", func() {
			c.pluginKeys = []string{"1and2", "2and3"}

			Expect(c.resolvePlugins()).To(Succeed())
			Expect(c.projectVersion.Compare(config.Version{Number: 2})).To(Equal(0))
		})

		It("should fail if no common project version is found", func() {
			c.pluginKeys = []string{"only1", "only2"}

			Expect(c.resolvePlugins()).NotTo(Succeed())
		})

		It("should fail if more than one common project versions are found", func() {
			c.pluginKeys = []string{"1and2", "1-2and3"}

			Expect(c.resolvePlugins()).NotTo(Succeed())
		})

		It("should succeed if more than one common project versions are found and one is the default", func() {
			c.pluginKeys = []string{"2and3", "1-2and3"}
			c.defaultProjectVersion = projectVersion

			Expect(c.resolvePlugins()).To(Succeed())
			Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
		})
	})

	Context("New", func() {
		var c *CLI
		var err error

		When("no option is provided", func() {
			It("should create a valid CLI", func() {
				_, err = New()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		// NOTE: Options are extensively tested in their own tests.
		//       The ones tested here ensure better coverage.

		When("providing a version string", func() {
			It("should create a valid CLI", func() {
				const version = "version string"
				c, err = New(
					WithPlugins(&goPluginV4.Plugin{}),
					WithDefaultPlugins(projectVersion, &goPluginV4.Plugin{}),
					WithVersion(version),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c.cmd, "version")).To(BeTrue())

				// Test the version command
				c.cmd.SetArgs([]string{"version"})
				// Overwrite stdout to read the output and reset it afterwards
				r, w, _ := os.Pipe()
				temp := os.Stdout
				defer func() {
					os.Stdout = temp
				}()
				os.Stdout = w
				Expect(c.cmd.Execute()).Should(Succeed())

				_ = w.Close()

				Expect(err).NotTo(HaveOccurred())
				printed, _ := io.ReadAll(r)
				Expect(string(printed)).To(Equal(
					fmt.Sprintf("%s\n", version)))

			})
		})

		When("enabling completion", func() {
			It("should create a valid CLI", func() {
				c, err = New(
					WithPlugins(&goPluginV4.Plugin{}),
					WithDefaultPlugins(projectVersion, &goPluginV4.Plugin{}),
					WithCompletion(),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c.cmd, "completion")).To(BeTrue())
			})
		})

		When("providing an invalid option", func() {
			It("should return an error", func() {
				// An empty project version is not valid
				_, err = New(WithDefaultProjectVersion(config.Version{}))
				Expect(err).To(HaveOccurred())
			})
		})

		When("being unable to resolve plugins", func() {
			// Save os.Args and restore it for every test
			var args []string
			BeforeEach(func() { args = os.Args })
			AfterEach(func() { os.Args = args })

			It("should return a CLI that returns an error", func() {
				setPluginsFlag("foo")

				c, err = New()
				Expect(err).NotTo(HaveOccurred())

				// Overwrite stderr to read the output and reset it afterwards
				_, w, _ := os.Pipe()
				temp := os.Stderr
				defer func() {
					os.Stderr = temp
					_ = w.Close()
				}()
				os.Stderr = w

				Expect(c.Run()).NotTo(Succeed())
			})
		})

		When("providing extra commands", func() {
			It("should create a valid CLI for non-conflicting ones", func() {
				extraCommand := &cobra.Command{Use: "extra"}
				c, err = New(
					WithPlugins(&goPluginV4.Plugin{}),
					WithDefaultPlugins(projectVersion, &goPluginV4.Plugin{}),
					WithExtraCommands(extraCommand),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c.cmd, extraCommand.Use)).To(BeTrue())
			})

			It("should return an error for conflicting ones", func() {
				extraCommand := &cobra.Command{Use: "init"}
				c, err = New(
					WithPlugins(&goPluginV4.Plugin{}),
					WithDefaultPlugins(projectVersion, &goPluginV4.Plugin{}),
					WithExtraCommands(extraCommand),
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing extra alpha commands", func() {
			It("should create a valid CLI for non-conflicting ones", func() {
				extraAlphaCommand := &cobra.Command{Use: "extra"}
				c, err = New(
					WithPlugins(&goPluginV4.Plugin{}),
					WithDefaultPlugins(projectVersion, &goPluginV4.Plugin{}),
					WithExtraAlphaCommands(extraAlphaCommand),
				)
				Expect(err).NotTo(HaveOccurred())
				var alpha *cobra.Command
				for _, subcmd := range c.cmd.Commands() {
					if subcmd.Name() == alphaCommand {
						alpha = subcmd
						break
					}
				}
				Expect(alpha).NotTo(BeNil())
				Expect(hasSubCommand(alpha, extraAlphaCommand.Use)).To(BeTrue())
			})

			It("should return an error for conflicting ones", func() {
				extraAlphaCommand := &cobra.Command{Use: "extra"}
				_, err = New(
					WithPlugins(&goPluginV4.Plugin{}),
					WithDefaultPlugins(projectVersion, &goPluginV4.Plugin{}),
					WithExtraAlphaCommands(extraAlphaCommand, extraAlphaCommand),
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing deprecated plugins", func() {
			It("should succeed and print the deprecation notice", func() {
				const (
					deprecationWarning = "DEPRECATED"
				)
				deprecatedPlugin := newMockDeprecatedPlugin("deprecated", "v1", deprecationWarning, projectVersion)

				// Overwrite stderr to read the deprecation output and reset it afterwards
				r, w, _ := os.Pipe()
				temp := os.Stderr
				defer func() {
					os.Stderr = temp
				}()
				os.Stderr = w

				c, err = New(
					WithPlugins(deprecatedPlugin),
					WithDefaultPlugins(projectVersion, deprecatedPlugin),
					WithDefaultProjectVersion(projectVersion),
				)

				_ = w.Close()

				Expect(err).NotTo(HaveOccurred())
				printed, _ := io.ReadAll(r)
				Expect(string(printed)).To(Equal(
					fmt.Sprintf(noticeColor, fmt.Sprintf(deprecationFmt, deprecationWarning))))
			})
		})

		When("new succeeds", func() {
			It("should return the underlying command", func() {
				c, err = New()
				Expect(err).NotTo(HaveOccurred())
				Expect(c.Command()).NotTo(BeNil())
				Expect(c.Command()).To(Equal(c.cmd))
			})
		})
	})
})
