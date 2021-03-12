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
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
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

// nolint:unparam
func setProjectVersionFlag(value string) {
	setFlag(projectVersionFlag, value)
}

func setPluginsFlag(value string) {
	setFlag(pluginsFlag, value)
}

func hasSubCommand(c *CLI, name string) bool {
	for _, subcommand := range c.cmd.Commands() {
		if subcommand.Name() == name {
			return true
		}
	}
	return false
}

var _ = Describe("CLI", func() {

	Context("getInfoFromFlags", func() {
		var (
			projectVersion string
			plugins        []string
			err            error
			c              *CLI
		)

		// Save os.Args and restore it for every test
		var args []string
		BeforeEach(func() {
			c = &CLI{}
			c.cmd = c.newRootCmd()
			args = os.Args
		})
		AfterEach(func() { os.Args = args })

		When("no flag is set", func() {
			It("should succeed", func() {
				projectVersion, plugins, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(""))
				Expect(len(plugins)).To(Equal(0))
			})
		})

		When(fmt.Sprintf("--%s flag is set", projectVersionFlag), func() {
			It("should succeed", func() {
				setProjectVersionFlag("2")
				projectVersion, plugins, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal("2"))
				Expect(len(plugins)).To(Equal(0))
			})
		})

		When(fmt.Sprintf("--%s flag is set", pluginsFlag), func() {
			It("should succeed using one plugin key", func() {
				setPluginsFlag("go/v1")
				projectVersion, plugins, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(""))
				Expect(plugins).To(Equal([]string{"go/v1"}))
			})

			It("should succeed using more than one plugin key", func() {
				setPluginsFlag("go/v1,example/v2,test/v1")
				projectVersion, plugins, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(""))
				Expect(plugins).To(Equal([]string{"go/v1", "example/v2", "test/v1"}))
			})

			It("should succeed using more than one plugin key with spaces", func() {
				setPluginsFlag("go/v1 , example/v2 , test/v1")
				projectVersion, plugins, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(""))
				Expect(plugins).To(Equal([]string{"go/v1", "example/v2", "test/v1"}))
			})
		})

		When(fmt.Sprintf("--%s and --%s flags are set", projectVersionFlag, pluginsFlag), func() {
			It("should succeed using one plugin key", func() {
				setProjectVersionFlag("2")
				setPluginsFlag("go/v1")
				projectVersion, plugins, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal("2"))
				Expect(plugins).To(Equal([]string{"go/v1"}))
			})

			It("should succeed using more than one plugin keys", func() {
				setProjectVersionFlag("2")
				setPluginsFlag("go/v1,example/v2,test/v1")
				projectVersion, plugins, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal("2"))
				Expect(plugins).To(Equal([]string{"go/v1", "example/v2", "test/v1"}))
			})

			It("should succeed using more than one plugin keys with spaces", func() {
				setProjectVersionFlag("2")
				setPluginsFlag("go/v1 , example/v2 , test/v1")
				projectVersion, plugins, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal("2"))
				Expect(plugins).To(Equal([]string{"go/v1", "example/v2", "test/v1"}))
			})
		})

		When("additional flags are set", func() {
			It("should succeed", func() {
				setFlag("extra-flag", "extra-value")
				_, _, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
			})

			// `--help` is not captured by the whitelist, so we need to special case it
			It("should not fail for `--help`", func() {
				setBoolFlag("help")
				_, _, err = c.getInfoFromFlags()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("getInfoFromConfig", func() {
		var (
			projectConfig  config.Config
			projectVersion config.Version
			plugins        []string
			err            error
		)

		When("not having layout field", func() {
			It("should succeed", func() {
				projectConfig = cfgv2.New()
				projectVersion, plugins, err = getInfoFromConfig(projectConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion.Compare(projectConfig.GetVersion())).To(Equal(0))
				Expect(len(plugins)).To(Equal(0))
			})
		})

		When("having layout field", func() {
			It("should succeed", func() {
				projectConfig = cfgv3.New()
				Expect(projectConfig.SetLayout("go.kubebuilder.io/v2")).To(Succeed())
				projectVersion, plugins, err = getInfoFromConfig(projectConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion.Compare(projectConfig.GetVersion())).To(Equal(0))
				Expect(plugins).To(Equal([]string{projectConfig.GetLayout()}))
			})
		})
	})

	Context("CLI.resolveFlagsAndConfigFileConflicts", func() {
		const (
			pluginKey1 = "go.kubebuilder.io/v1"
			pluginKey2 = "go.kubebuilder.io/v2"
			pluginKey3 = "go.kubebuilder.io/v3"
		)
		var (
			c *CLI

			projectVersion config.Version
			plugins        []string
			err            error

			projectVersion1 = config.Version{Number: 1}
			projectVersion2 = config.Version{Number: 2}
			projectVersion3 = config.Version{Number: 3}
		)

		When("having no project version set", func() {
			It("should succeed", func() {
				c = &CLI{}
				projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					config.Version{},
					nil,
					nil,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion.Compare(config.Version{})).To(Equal(0))
			})
		})

		When("having one project version source", func() {
			When("having default project version set", func() {
				It("should succeed", func() {
					c = &CLI{
						defaultProjectVersion: projectVersion1,
					}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						config.Version{},
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion.Compare(projectVersion1)).To(Equal(0))
				})
			})

			When("having project version set from flags", func() {
				It("should succeed", func() {
					c = &CLI{}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						projectVersion1.String(),
						config.Version{},
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion.Compare(projectVersion1)).To(Equal(0))
				})
			})

			When("having project version set from config file", func() {
				It("should succeed", func() {
					c = &CLI{}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						projectVersion1,
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion.Compare(projectVersion1)).To(Equal(0))
				})
			})
		})

		When("having two project version source", func() {
			When("having default project version set and from flags", func() {
				It("should succeed", func() {
					c = &CLI{
						defaultProjectVersion: projectVersion1,
					}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						projectVersion2.String(),
						config.Version{},
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion.Compare(projectVersion2)).To(Equal(0))
				})
			})

			When("having default project version set and from config file", func() {
				It("should succeed", func() {
					c = &CLI{
						defaultProjectVersion: projectVersion1,
					}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						projectVersion2,
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion.Compare(projectVersion2)).To(Equal(0))
				})
			})

			When("having project version set from flags and config file", func() {
				It("should succeed if they are the same", func() {
					c = &CLI{}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						projectVersion1.String(),
						projectVersion1,
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion.Compare(projectVersion1)).To(Equal(0))
				})

				It("should fail if they are different", func() {
					c = &CLI{}
					_, _, err = c.resolveFlagsAndConfigFileConflicts(
						projectVersion1.String(),
						projectVersion2,
						nil,
						nil,
					)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("having three project version sources", func() {
			It("should succeed if project version from flags and config file are the same", func() {
				c = &CLI{
					defaultProjectVersion: projectVersion1,
				}
				projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
					projectVersion2.String(),
					projectVersion2,
					nil,
					nil,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion.Compare(projectVersion2)).To(Equal(0))
			})

			It("should fail if project version from flags and config file are different", func() {
				c = &CLI{
					defaultProjectVersion: projectVersion1,
				}
				_, _, err = c.resolveFlagsAndConfigFileConflicts(
					projectVersion2.String(),
					projectVersion3,
					nil,
					nil,
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("an invalid project version is set", func() {
			It("should fail", func() {
				c = &CLI{}
				projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
					"0",
					config.Version{},
					nil,
					nil,
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("having no plugin keys set", func() {
			It("should succeed", func() {
				c = &CLI{}
				_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					config.Version{},
					nil,
					nil,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(plugins)).To(Equal(0))
			})
		})

		When("having one plugin keys source", func() {
			When("having default plugin keys set", func() {
				It("should succeed", func() {
					c = &CLI{
						defaultProjectVersion: projectVersion1,
						defaultPlugins: map[config.Version][]string{
							projectVersion1: {pluginKey1},
							projectVersion2: {pluginKey2},
						},
					}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						config.Version{},
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey1))
				})
			})

			When("having plugin keys set from flags", func() {
				It("should succeed", func() {
					c = &CLI{}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						config.Version{},
						[]string{pluginKey1},
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey1))
				})
			})

			When("having plugin keys set from config file", func() {
				It("should succeed", func() {
					c = &CLI{}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						config.Version{},
						nil,
						[]string{pluginKey1},
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey1))
				})
			})
		})

		When("having two plugin keys source", func() {
			When("having default plugin keys set and from flags", func() {
				It("should succeed", func() {
					c = &CLI{
						defaultPlugins: map[config.Version][]string{
							{}: {pluginKey1},
						},
					}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						config.Version{},
						[]string{pluginKey2},
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey2))
				})
			})

			When("having default plugin keys set and from config file", func() {
				It("should succeed", func() {
					c = &CLI{
						defaultPlugins: map[config.Version][]string{
							{}: {pluginKey1},
						},
					}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						config.Version{},
						nil,
						[]string{pluginKey2},
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey2))
				})
			})

			When("having plugin keys set from flags and config file", func() {
				It("should succeed if they are the same", func() {
					c = &CLI{}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						config.Version{},
						[]string{pluginKey1},
						[]string{pluginKey1},
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey1))
				})

				It("should fail if they are different", func() {
					c = &CLI{}
					_, _, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						config.Version{},
						[]string{pluginKey1},
						[]string{pluginKey2},
					)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("having three plugin keys sources", func() {
			It("should succeed if plugin keys from flags and config file are the same", func() {
				c = &CLI{
					defaultPlugins: map[config.Version][]string{
						{}: {pluginKey1},
					},
				}
				_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					config.Version{},
					[]string{pluginKey2},
					[]string{pluginKey2},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(plugins)).To(Equal(1))
				Expect(plugins[0]).To(Equal(pluginKey2))
			})

			It("should fail if plugin keys from flags and config file are different", func() {
				c = &CLI{
					defaultPlugins: map[config.Version][]string{
						{}: {pluginKey1},
					},
				}
				_, _, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					config.Version{},
					[]string{pluginKey2},
					[]string{pluginKey3},
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("an invalid plugin key is set", func() {
			It("should fail", func() {
				c = &CLI{}
				_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					config.Version{},
					[]string{"A"},
					nil,
				)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	// NOTE: only flag info can be tested with CLI.getInfo as the config file doesn't exist,
	//       previous tests ensure that the info from config files is read properly and that
	//       conflicts are solved appropriately.
	Context("CLI.getInfo", func() {
		It("should set project version and plugin keys", func() {
			projectVersion := config.Version{Number: 2}
			pluginKeys := []string{"go.kubebuilder.io/v2"}
			c := &CLI{
				defaultProjectVersion: projectVersion,
				defaultPlugins: map[config.Version][]string{
					projectVersion: pluginKeys,
				},
				fs: machinery.Filesystem{FS: afero.NewMemMapFs()},
			}
			c.cmd = c.newRootCmd()
			Expect(c.getInfo()).To(Succeed())
			Expect(c.projectVersion.Compare(projectVersion)).To(Equal(0))
			Expect(c.pluginKeys).To(Equal(pluginKeys))
		})
	})

	Context("CLI.resolve", func() {
		var (
			c *CLI

			projectVersion = config.Version{Number: 2}

			pluginKeys = []string{
				"foo.example.com/v1",
				"bar.example.com/v1",
				"baz.example.com/v1",
				"foo.kubebuilder.io/v1",
				"foo.kubebuilder.io/v2",
				"bar.kubebuilder.io/v1",
				"bar.kubebuilder.io/v2",
			}
		)

		plugins := makeMockPluginsFor(projectVersion, pluginKeys...)
		plugins = append(plugins, newMockPlugin("invalid.kubebuilder.io", "v1"))
		pluginMap := makeMapFor(plugins...)

		for key, qualified := range map[string]string{
			"foo.example.com/v1": "foo.example.com/v1",
			"foo.example.com":    "foo.example.com/v1",
			"baz":                "baz.example.com/v1",
			"foo/v2":             "foo.kubebuilder.io/v2",
		} {
			key, qualified := key, qualified
			It(fmt.Sprintf("should resolve %q", key), func() {
				c = &CLI{
					plugins:        pluginMap,
					projectVersion: projectVersion,
					pluginKeys:     []string{key},
				}
				Expect(c.resolve()).To(Succeed())
				Expect(len(c.resolvedPlugins)).To(Equal(1))
				Expect(plugin.KeyFor(c.resolvedPlugins[0])).To(Equal(qualified))
			})
		}

		for _, key := range []string{
			"foo.kubebuilder.io",
			"foo/v1",
			"foo",
			"blah",
			"foo.example.com/v2",
			"foo/v3",
			"foo.example.com/v3",
			"invalid.kubebuilder.io/v1",
		} {
			key := key
			It(fmt.Sprintf("should not resolve %q", key), func() {
				c = &CLI{
					plugins:        pluginMap,
					projectVersion: projectVersion,
					pluginKeys:     []string{key},
				}
				Expect(c.resolve()).NotTo(Succeed())
			})
		}
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
				c, err = New(WithVersion(version))
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c, "version")).To(BeTrue())
			})
		})

		When("enabling completion", func() {
			It("should create a valid CLI", func() {
				c, err = New(WithCompletion())
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c, "completion")).To(BeTrue())
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
				Expect(c.Run()).NotTo(Succeed())
			})
		})

		When("providing extra commands", func() {
			var extraCommand *cobra.Command

			It("should create a valid CLI for non-conflicting ones", func() {
				extraCommand = &cobra.Command{Use: "extra"}
				c, err = New(WithExtraCommands(extraCommand))
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c, extraCommand.Use)).To(BeTrue())
			})

			It("should return an error for conflicting ones", func() {
				extraCommand = &cobra.Command{Use: "init"}
				_, err = New(WithExtraCommands(extraCommand))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing deprecated plugins", func() {
			It("should succeed and print the deprecation notice", func() {
				const (
					deprecationWarning = "DEPRECATED"
				)
				var (
					projectVersion   = config.Version{Number: 2}
					deprecatedPlugin = newMockDeprecatedPlugin("deprecated", "v1", deprecationWarning, projectVersion)
				)

				// Overwrite stdout to read the output and reset it afterwards
				r, w, _ := os.Pipe()
				temp := os.Stdout
				defer func() {
					os.Stdout = temp
				}()
				os.Stdout = w

				c, err = New(
					WithDefaultProjectVersion(projectVersion),
					WithDefaultPlugins(projectVersion, deprecatedPlugin),
					WithPlugins(deprecatedPlugin),
				)
				_ = w.Close()
				Expect(err).NotTo(HaveOccurred())
				printed, _ := ioutil.ReadAll(r)
				Expect(string(printed)).To(Equal(
					fmt.Sprintf(noticeColor, fmt.Sprintf(deprecationFmt, deprecationWarning))))
			})
		})
	})

})
