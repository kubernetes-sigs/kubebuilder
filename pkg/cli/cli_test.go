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
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"
)

func makeMockPluginsFor(projectVersion string, pluginKeys ...string) []plugin.Plugin {
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

// nolint:unparam
func setProjectVersionFlag(value string) {
	setFlag(projectVersionFlag, value)
}

func setPluginsFlag(value string) {
	setFlag(pluginsFlag, value)
}

func hasSubCommand(c CLI, name string) bool {
	for _, subcommand := range c.(*cli).cmd.Commands() {
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
			c              *cli
		)

		// Save os.Args and restore it for every test
		var args []string
		BeforeEach(func() {
			c = &cli{}
			args = os.Args
		})
		AfterEach(func() { os.Args = args })

		When("no flag is set", func() {
			It("should success", func() {
				projectVersion, plugins = c.getInfoFromFlags()
				Expect(projectVersion).To(Equal(""))
				Expect(len(plugins)).To(Equal(0))
			})
		})

		When(fmt.Sprintf("--%s flag is set", projectVersionFlag), func() {
			It("should success", func() {
				setProjectVersionFlag("2")
				projectVersion, plugins = c.getInfoFromFlags()
				Expect(projectVersion).To(Equal("2"))
				Expect(len(plugins)).To(Equal(0))
			})
		})

		When(fmt.Sprintf("--%s flag is set", pluginsFlag), func() {
			It("should success using one plugin key", func() {
				setPluginsFlag("go/v1")
				projectVersion, plugins = c.getInfoFromFlags()
				Expect(projectVersion).To(Equal(""))
				Expect(plugins).To(Equal([]string{"go/v1"}))
			})

			It("should success using more than one plugin key", func() {
				setPluginsFlag("go/v1,example/v2,test/v1")
				projectVersion, plugins = c.getInfoFromFlags()
				Expect(projectVersion).To(Equal(""))
				Expect(plugins).To(Equal([]string{"go/v1", "example/v2", "test/v1"}))
			})

			It("should success using more than one plugin key with spaces", func() {
				setPluginsFlag("go/v1 , example/v2 , test/v1")
				projectVersion, plugins = c.getInfoFromFlags()
				Expect(projectVersion).To(Equal(""))
				Expect(plugins).To(Equal([]string{"go/v1", "example/v2", "test/v1"}))
			})
		})

		When(fmt.Sprintf("--%s and --%s flags are set", projectVersionFlag, pluginsFlag), func() {
			It("should success using one plugin key", func() {
				setProjectVersionFlag("2")
				setPluginsFlag("go/v1")
				projectVersion, plugins = c.getInfoFromFlags()
				Expect(projectVersion).To(Equal("2"))
				Expect(plugins).To(Equal([]string{"go/v1"}))
			})

			It("should success using more than one plugin keys", func() {
				setProjectVersionFlag("2")
				setPluginsFlag("go/v1,example/v2,test/v1")
				projectVersion, plugins = c.getInfoFromFlags()
				Expect(projectVersion).To(Equal("2"))
				Expect(plugins).To(Equal([]string{"go/v1", "example/v2", "test/v1"}))
			})

			It("should success using more than one plugin keys with spaces", func() {
				setProjectVersionFlag("2")
				setPluginsFlag("go/v1 , example/v2 , test/v1")
				projectVersion, plugins = c.getInfoFromFlags()
				Expect(projectVersion).To(Equal("2"))
				Expect(plugins).To(Equal([]string{"go/v1", "example/v2", "test/v1"}))
			})
		})

		When("additional flags are set", func() {
			It("should not fail", func() {
				setFlag("extra-flag", "extra-value")
				c.getInfoFromFlags()
			})
		})
	})

	Context("getInfoFromConfig", func() {
		var (
			projectConfig  *config.Config
			projectVersion string
			plugins        []string
			err            error
		)

		When("having version field", func() {
			It("should success", func() {
				projectConfig = &config.Config{
					Version: "2",
				}
				projectVersion, plugins, err = getInfoFromConfig(projectConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(projectConfig.Version))
				Expect(len(plugins)).To(Equal(0))
			})
		})

		When("having layout field", func() {
			It("should success", func() {
				projectConfig = &config.Config{
					Layout: "go.kubebuilder.io/v2",
				}
				projectVersion, plugins, err = getInfoFromConfig(projectConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(""))
				Expect(plugins).To(Equal([]string{projectConfig.Layout}))
			})
		})

		When("having both version and layout fields", func() {
			It("should success", func() {
				projectConfig = &config.Config{
					Version: "3-alpha",
					Layout:  "go.kubebuilder.io/v2",
				}
				projectVersion, plugins, err = getInfoFromConfig(projectConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(projectConfig.Version))
				Expect(plugins).To(Equal([]string{projectConfig.Layout}))
			})
		})

		When("not having neither version nor layout fields set", func() {
			It("should success", func() {
				projectConfig = &config.Config{}
				projectVersion, plugins, err = getInfoFromConfig(projectConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(""))
				Expect(len(plugins)).To(Equal(0))
			})
		})
	})

	Context("cli.resolveFlagsAndConfigFileConflicts", func() {
		const (
			projectVersion1 = "1"
			projectVersion2 = "2"
			projectVersion3 = "3"

			pluginKey1 = "go.kubebuilder.io/v1"
			pluginKey2 = "go.kubebuilder.io/v2"
			pluginKey3 = "go.kubebuilder.io/v3"
		)
		var (
			c *cli

			projectVersion string
			plugins        []string
			err            error
		)

		When("having no project version set", func() {
			It("should success", func() {
				c = &cli{}
				projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					"",
					nil,
					nil,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(""))
			})
		})

		When("having one project version source", func() {
			When("having default project version set", func() {
				It("should success", func() {
					c = &cli{
						defaultProjectVersion: projectVersion1,
					}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						"",
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion).To(Equal(projectVersion1))
				})
			})

			When("having project version set from flags", func() {
				It("should success", func() {
					c = &cli{}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						projectVersion1,
						"",
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion).To(Equal(projectVersion1))
				})
			})

			When("having project version set from config file", func() {
				It("should success", func() {
					c = &cli{}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						projectVersion1,
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion).To(Equal(projectVersion1))
				})
			})
		})

		When("having two project version source", func() {
			When("having default project version set and from flags", func() {
				It("should success", func() {
					c = &cli{
						defaultProjectVersion: projectVersion1,
					}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						projectVersion2,
						"",
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion).To(Equal(projectVersion2))
				})
			})

			When("having default project version set and from config file", func() {
				It("should success", func() {
					c = &cli{
						defaultProjectVersion: projectVersion1,
					}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						projectVersion2,
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion).To(Equal(projectVersion2))
				})
			})

			When("having project version set from flags and config file", func() {
				It("should success if they are the same", func() {
					c = &cli{}
					projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
						projectVersion1,
						projectVersion1,
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectVersion).To(Equal(projectVersion1))
				})

				It("should fail if they are different", func() {
					c = &cli{}
					_, _, err = c.resolveFlagsAndConfigFileConflicts(
						projectVersion1,
						projectVersion2,
						nil,
						nil,
					)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("having three project version sources", func() {
			It("should success if project version from flags and config file are the same", func() {
				c = &cli{
					defaultProjectVersion: projectVersion1,
				}
				projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
					projectVersion2,
					projectVersion2,
					nil,
					nil,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectVersion).To(Equal(projectVersion2))
			})

			It("should fail if project version from flags and config file are different", func() {
				c = &cli{
					defaultProjectVersion: projectVersion1,
				}
				_, _, err = c.resolveFlagsAndConfigFileConflicts(
					projectVersion2,
					projectVersion3,
					nil,
					nil,
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("an invalid project version is set", func() {
			It("should fail", func() {
				c = &cli{
					defaultProjectVersion: "v1",
				}
				projectVersion, _, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					"",
					nil,
					nil,
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("having no plugin keys set", func() {
			It("should success", func() {
				c = &cli{}
				_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					"",
					nil,
					nil,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(plugins)).To(Equal(0))
			})
		})

		When("having one plugin keys source", func() {
			When("having default plugin keys set", func() {
				It("should success", func() {
					c = &cli{
						defaultProjectVersion: projectVersion1,
						defaultPlugins: map[string][]string{
							projectVersion1: {pluginKey1},
							projectVersion2: {pluginKey2},
						},
					}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						"",
						nil,
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey1))
				})
			})

			When("having plugin keys set from flags", func() {
				It("should success", func() {
					c = &cli{}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						"",
						[]string{pluginKey1},
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey1))
				})
			})

			When("having plugin keys set from config file", func() {
				It("should success", func() {
					c = &cli{}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						"",
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
				It("should success", func() {
					c = &cli{
						defaultPlugins: map[string][]string{
							"": {pluginKey1},
						},
					}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						"",
						[]string{pluginKey2},
						nil,
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey2))
				})
			})

			When("having default plugin keys set and from config file", func() {
				It("should success", func() {
					c = &cli{
						defaultPlugins: map[string][]string{
							"": {pluginKey1},
						},
					}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						"",
						nil,
						[]string{pluginKey2},
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey2))
				})
			})

			When("having plugin keys set from flags and config file", func() {
				It("should success if they are the same", func() {
					c = &cli{}
					_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						"",
						[]string{pluginKey1},
						[]string{pluginKey1},
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))
					Expect(plugins[0]).To(Equal(pluginKey1))
				})

				It("should fail if they are different", func() {
					c = &cli{}
					_, _, err = c.resolveFlagsAndConfigFileConflicts(
						"",
						"",
						[]string{pluginKey1},
						[]string{pluginKey2},
					)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("having three plugin keys sources", func() {
			It("should success if plugin keys from flags and config file are the same", func() {
				c = &cli{
					defaultPlugins: map[string][]string{
						"": {pluginKey1},
					},
				}
				_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					"",
					[]string{pluginKey2},
					[]string{pluginKey2},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(plugins)).To(Equal(1))
				Expect(plugins[0]).To(Equal(pluginKey2))
			})

			It("should fail if plugin keys from flags and config file are different", func() {
				c = &cli{
					defaultPlugins: map[string][]string{
						"": {pluginKey1},
					},
				}
				_, _, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					"",
					[]string{pluginKey2},
					[]string{pluginKey3},
				)
				Expect(err).To(HaveOccurred())
			})
		})

		When("an invalid plugin key is set", func() {
			It("should fail", func() {
				c = &cli{
					defaultProjectVersion: projectVersion1,
					defaultPlugins: map[string][]string{
						projectVersion1: {"invalid_plugin/v1"},
					},
				}
				_, plugins, err = c.resolveFlagsAndConfigFileConflicts(
					"",
					"",
					nil,
					nil,
				)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	// NOTE: only flag info can be tested with cli.getInfo as the config file doesn't exist,
	//       previous tests ensure that the info from config files is read properly and that
	//       conflicts are solved appropriately.
	Context("cli.getInfo", func() {
		It("should set project version and plugin keys", func() {
			projectVersion := "2"
			pluginKeys := []string{"go.kubebuilder.io/v2"}
			c := &cli{
				defaultProjectVersion: projectVersion,
				defaultPlugins: map[string][]string{
					projectVersion: pluginKeys,
				},
			}
			Expect(c.getInfo()).To(Succeed())
			Expect(c.projectVersion).To(Equal(projectVersion))
			Expect(c.pluginKeys).To(Equal(pluginKeys))
		})
	})

	Context("cli.resolve", func() {
		const projectVersion = "2"
		var (
			c *cli

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
				c = &cli{
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
				c = &cli{
					plugins:        pluginMap,
					projectVersion: projectVersion,
					pluginKeys:     []string{key},
				}
				Expect(c.resolve()).NotTo(Succeed())
			})
		}
	})

	Context("New", func() {
		var c CLI
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
				c, err = New(WithCompletion)
				Expect(err).NotTo(HaveOccurred())
				Expect(hasSubCommand(c, "completion")).To(BeTrue())
			})
		})

		When("providing an invalid option", func() {
			It("should return an error", func() {
				// An empty project version is not valid
				_, err = New(WithDefaultProjectVersion(""))
				Expect(err).To(HaveOccurred())
			})
		})

		When("being unable to resolve plugins", func() {
			// Save os.Args and restore it for every test
			var args []string
			BeforeEach(func() { args = os.Args })
			AfterEach(func() { os.Args = args })

			It("should return an error", func() {
				setPluginsFlag("foo")
				_, err = New()
				Expect(err).To(HaveOccurred())
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
			It("should success and print the deprecation notice", func() {
				const (
					projectVersion     = "2"
					deprecationWarning = "DEPRECATED"
				)
				var deprecatedPlugin = newMockDeprecatedPlugin("deprecated", "v1", deprecationWarning, projectVersion)

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
