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
	"errors"
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

var _ = Describe("Discover external plugins", func() {
	Context("when plugin executables exist in the expected plugin directories", func() {
		const (
			filePermissions  os.FileMode = 755
			testPluginScript             = `#!/bin/bash
			echo "This is an external plugin"
			`
		)

		var (
			pluginFilePath string
			pluginFileName string
			pluginPath     string
			f              afero.File
			fs             machinery.Filesystem
			err            error
		)

		BeforeEach(func() {
			fs = machinery.Filesystem{
				FS: afero.NewMemMapFs(),
			}

			pluginPath, err = getPluginsRoot(runtime.GOOS)
			Expect(err).To(BeNil())

			pluginFileName = "externalPlugin.sh"
			pluginFilePath = filepath.Join(pluginPath, "externalPlugin", "v1", pluginFileName)

			err = fs.FS.MkdirAll(filepath.Dir(pluginFilePath), 0o700)
			Expect(err).To(BeNil())

			f, err = fs.FS.Create(pluginFilePath)
			Expect(err).To(BeNil())
			Expect(f).ToNot(BeNil())

			_, err = fs.FS.Stat(pluginFilePath)
			Expect(err).To(BeNil())
		})

		It("should discover the external plugin executable without any errors", func() {
			// test that DiscoverExternalPlugins works if the plugin file is an executable and
			// is found in the expected path
			_, err = f.WriteString(testPluginScript)
			Expect(err).To(Not(HaveOccurred()))

			err = fs.FS.Chmod(pluginFilePath, filePermissions)
			Expect(err).To(Not(HaveOccurred()))

			_, err = fs.FS.Stat(pluginFilePath)
			Expect(err).To(BeNil())

			ps, err := DiscoverExternalPlugins(fs.FS)
			Expect(err).To(BeNil())
			Expect(ps).NotTo(BeNil())
			Expect(len(ps)).To(Equal(1))
			Expect(ps[0].Name()).To(Equal("externalPlugin"))
			Expect(ps[0].Version().Number).To(Equal(1))
		})

		It("should discover multiple external plugins and return the plugins without any errors", func() {
			// set the execute permissions on the first plugin executable
			err = fs.FS.Chmod(pluginFilePath, filePermissions)

			pluginFileName = "myotherexternalPlugin.sh"
			pluginFilePath = filepath.Join(pluginPath, "myotherexternalPlugin", "v1", pluginFileName)

			f, err = fs.FS.Create(pluginFilePath)
			Expect(err).To(BeNil())
			Expect(f).ToNot(BeNil())

			_, err = fs.FS.Stat(pluginFilePath)
			Expect(err).To(BeNil())

			_, err = f.WriteString(testPluginScript)
			Expect(err).To(Not(HaveOccurred()))

			// set the execute permissions on the second plugin executable
			err = fs.FS.Chmod(pluginFilePath, filePermissions)
			Expect(err).To(Not(HaveOccurred()))

			_, err = fs.FS.Stat(pluginFilePath)
			Expect(err).To(BeNil())

			ps, err := DiscoverExternalPlugins(fs.FS)
			Expect(err).To(BeNil())
			Expect(ps).NotTo(BeNil())
			Expect(len(ps)).To(Equal(2))

			Expect(ps[0].Name()).To(Equal("externalPlugin"))
			Expect(ps[1].Name()).To(Equal("myotherexternalPlugin"))
		})

		Context("that are invalid", func() {
			BeforeEach(func() {
				fs = machinery.Filesystem{
					FS: afero.NewMemMapFs(),
				}

				pluginPath, err = getPluginsRoot(runtime.GOOS)
				Expect(err).To(BeNil())
			})

			It("should error if the plugin found is not an executable", func() {
				pluginFileName = "externalPlugin.sh"
				pluginFilePath = filepath.Join(pluginPath, "externalPlugin", "v1", pluginFileName)

				err = fs.FS.MkdirAll(filepath.Dir(pluginFilePath), 0o700)
				Expect(err).To(BeNil())

				f, err := fs.FS.Create(pluginFilePath)
				Expect(err).To(BeNil())
				Expect(f).ToNot(BeNil())

				_, err = fs.FS.Stat(pluginFilePath)
				Expect(err).To(BeNil())

				// set the plugin file permissions to read-only
				err = fs.FS.Chmod(pluginFilePath, 0o444)
				Expect(err).To(Not(HaveOccurred()))

				ps, err := DiscoverExternalPlugins(fs.FS)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("not an executable"))
				Expect(len(ps)).To(Equal(0))
			})

			It("should error if the plugin found has an invalid plugin name", func() {
				pluginFileName = ".sh"
				pluginFilePath = filepath.Join(pluginPath, "externalPlugin", "v1", pluginFileName)

				err = fs.FS.MkdirAll(filepath.Dir(pluginFilePath), 0o700)
				Expect(err).To(BeNil())

				f, err = fs.FS.Create(pluginFilePath)
				Expect(err).To(BeNil())
				Expect(f).ToNot(BeNil())

				ps, err := DiscoverExternalPlugins(fs.FS)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid plugin name found"))
				Expect(len(ps)).To(Equal(0))
			})
		})

		Context("that does not match the plugin root directory name", func() {
			BeforeEach(func() {
				fs = machinery.Filesystem{
					FS: afero.NewMemMapFs(),
				}

				pluginPath, err = getPluginsRoot(runtime.GOOS)
				Expect(err).To(BeNil())
			})

			It("should skip adding the external plugin and not return any errors", func() {
				pluginFileName = "random.sh"
				pluginFilePath = filepath.Join(pluginPath, "externalPlugin", "v1", pluginFileName)

				err = fs.FS.MkdirAll(filepath.Dir(pluginFilePath), 0o700)
				Expect(err).To(BeNil())

				f, err = fs.FS.Create(pluginFilePath)
				Expect(err).To(BeNil())
				Expect(f).ToNot(BeNil())

				err = fs.FS.Chmod(pluginFilePath, filePermissions)
				Expect(err).To(BeNil())

				ps, err := DiscoverExternalPlugins(fs.FS)
				Expect(err).To(BeNil())
				Expect(len(ps)).To(Equal(0))
			})

			It("should fail if pluginsroot is empty", func() {
				errPluginsRoot := errors.New("could not retrieve plugins root")
				retrievePluginsRoot = func(host string) (string, error) {
					return "", errPluginsRoot
				}

				_, err := DiscoverExternalPlugins(fs.FS)
				Expect(err).NotTo(BeNil())

				Expect(err).To(Equal(errPluginsRoot))
			})

			It("should fail for any other host that is not supported", func() {
				_, err := getPluginsRoot("darwin")
				Expect(err).To(BeNil())

				_, err = getPluginsRoot("linux")
				Expect(err).To(BeNil())

				_, err = getPluginsRoot("random")
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("host not supported"))
			})

			It("should skip parsing of directories if plugins root is not a directory", func() {
				retrievePluginsRoot = func(host string) (string, error) {
					return "externalplugin.sh", nil
				}

				_, err := DiscoverExternalPlugins(fs.FS)
				Expect(err).To(BeNil())
			})

			It("should fail for any other host that is not supported", func() {
				_, err := getPluginsRoot("darwin")
				Expect(err).To(BeNil())

				_, err = getPluginsRoot("linux")
				Expect(err).To(BeNil())

				_, err = getPluginsRoot("random")
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("host not supported"))
			})

			It("should return full path to the external plugins without XDG_CONFIG_HOME", func() {
				if _, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
					err = os.Setenv("XDG_CONFIG_HOME", "")
					Expect(err).To(BeNil())
				}

				home := os.Getenv("HOME")

				pluginsRoot, err := getPluginsRoot("darwin")
				Expect(err).To(BeNil())
				expected := filepath.Join(home, "Library", "Application Support", "kubebuilder", "plugins")
				Expect(pluginsRoot).To(Equal(expected))

				pluginsRoot, err = getPluginsRoot("linux")
				Expect(err).To(BeNil())
				expected = filepath.Join(home, ".config", "kubebuilder", "plugins")
				Expect(pluginsRoot).To(Equal(expected))
			})

			It("should return full path to the external plugins with XDG_CONFIG_HOME", func() {
				err = os.Setenv("XDG_CONFIG_HOME", "/some/random/path")
				Expect(err).To(BeNil())

				pluginsRoot, err := getPluginsRoot(runtime.GOOS)
				Expect(err).To(BeNil())
				Expect(pluginsRoot).To(Equal("/some/random/path/kubebuilder/plugins"))
			})

			It("should return error when home directory is set to empty", func() {
				_, ok := os.LookupEnv("XDG_CONFIG_HOME")
				if !ok {
				} else {
					err = os.Setenv("XDG_CONFIG_HOME", "")
					Expect(err).To(BeNil())
				}

				_, ok = os.LookupEnv("HOME")
				if !ok {
				} else {
					err = os.Setenv("HOME", "")
					Expect(err).To(BeNil())
				}

				pluginsroot, err := getPluginsRoot(runtime.GOOS)
				Expect(err).NotTo(BeNil())
				Expect(pluginsroot).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("error retrieving home dir"))
			})
		})
	})

	Context("parsing flags for external plugins", func() {
		It("should only parse flags excluding the `--plugins` flag", func() {
			// change the os.Args for this test and set them back after
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = []string{
				"kubebuilder",
				"init",
				"--plugins",
				"myexternalplugin/v1",
				"--domain",
				"example.com",
				"--binary-flag",
				"--license",
				"apache2",
				"--another-binary",
			}

			args := parseExternalPluginArgs()
			Expect(args).Should(ContainElements(
				"--domain",
				"example.com",
				"--binary-flag",
				"--license",
				"apache2",
				"--another-binary",
			))

			Expect(args).ShouldNot(ContainElements(
				"kubebuilder",
				"init",
				"--plugins",
				"myexternalplugin/v1",
			))
		})
	})
})

var _ = Describe("CLI options", func() {
	const (
		pluginName    = "plugin"
		pluginVersion = "v1"
	)

	var (
		c   *CLI
		err error

		projectVersion = config.Version{Number: 1}

		p   = newMockPlugin(pluginName, pluginVersion, projectVersion)
		np1 = newMockPlugin("Plugin", pluginVersion, projectVersion)
		np2 = mockPlugin{pluginName, plugin.Version{Number: -1}, []config.Version{projectVersion}}
		np3 = newMockPlugin(pluginName, pluginVersion)
		np4 = newMockPlugin(pluginName, pluginVersion, config.Version{})
	)

	Context("WithCommandName", func() {
		It("should use provided command name", func() {
			commandName := "other-command"
			c, err = newCLI(WithCommandName(commandName))
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.commandName).To(Equal(commandName))
		})
	})

	Context("WithVersion", func() {
		It("should use the provided version string", func() {
			version := "Version: 0.0.0"
			c, err = newCLI(WithVersion(version))
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.version).To(Equal(version))
		})
	})

	Context("WithDescription", func() {
		It("should use the provided description string", func() {
			description := "alternative description"
			c, err = newCLI(WithDescription(description))
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.description).To(Equal(description))
		})
	})

	Context("WithPlugins", func() {
		It("should return a valid CLI", func() {
			c, err = newCLI(WithPlugins(p))
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.plugins).To(Equal(map[string]plugin.Plugin{plugin.KeyFor(p): p}))
		})

		When("providing plugins with same keys", func() {
			It("should return an error", func() {
				_, err = newCLI(WithPlugins(p, p))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing plugins with same keys in two steps", func() {
			It("should return an error", func() {
				_, err = newCLI(WithPlugins(p), WithPlugins(p))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing a plugin with an invalid name", func() {
			It("should return an error", func() {
				_, err = newCLI(WithPlugins(np1))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing a plugin with an invalid version", func() {
			It("should return an error", func() {
				_, err = newCLI(WithPlugins(np2))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing a plugin with an empty list of supported versions", func() {
			It("should return an error", func() {
				_, err = newCLI(WithPlugins(np3))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing a plugin with an invalid list of supported versions", func() {
			It("should return an error", func() {
				_, err = newCLI(WithPlugins(np4))
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("WithDefaultPlugins", func() {
		It("should return a valid CLI", func() {
			c, err = newCLI(WithDefaultPlugins(projectVersion, p))
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.defaultPlugins).To(Equal(map[config.Version][]string{projectVersion: {plugin.KeyFor(p)}}))
		})

		When("providing an invalid project version", func() {
			It("should return an error", func() {
				_, err = newCLI(WithDefaultPlugins(config.Version{}, p))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing an empty set of plugins", func() {
			It("should return an error", func() {
				_, err = newCLI(WithDefaultPlugins(projectVersion))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing a plugin with an invalid name", func() {
			It("should return an error", func() {
				_, err = newCLI(WithDefaultPlugins(projectVersion, np1))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing a plugin with an invalid version", func() {
			It("should return an error", func() {
				_, err = newCLI(WithDefaultPlugins(projectVersion, np2))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing a plugin with an empty list of supported versions", func() {
			It("should return an error", func() {
				_, err = newCLI(WithDefaultPlugins(projectVersion, np3))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing a plugin with an invalid list of supported versions", func() {
			It("should return an error", func() {
				_, err = newCLI(WithDefaultPlugins(projectVersion, np4))
				Expect(err).To(HaveOccurred())
			})
		})

		When("providing a default plugin for an unsupported project version", func() {
			It("should return an error", func() {
				_, err = newCLI(WithDefaultPlugins(config.Version{Number: 2}, p))
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("WithDefaultProjectVersion", func() {
		DescribeTable("should return a valid CLI",
			func(projectVersion config.Version) {
				c, err = newCLI(WithDefaultProjectVersion(projectVersion))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.defaultProjectVersion).To(Equal(projectVersion))
			},
			Entry("for version `2`", config.Version{Number: 2}),
			Entry("for version `3-alpha`", config.Version{Number: 3, Stage: stage.Alpha}),
			Entry("for version `3`", config.Version{Number: 3}),
		)

		DescribeTable("should fail",
			func(projectVersion config.Version) {
				_, err = newCLI(WithDefaultProjectVersion(projectVersion))
				Expect(err).To(HaveOccurred())
			},
			Entry("for empty version", config.Version{}),
			Entry("for invalid stage", config.Version{Number: 1, Stage: stage.Stage(27)}),
		)
	})

	Context("WithExtraCommands", func() {
		It("should return a valid CLI with extra commands", func() {
			commandTest := &cobra.Command{
				Use: "example",
			}
			c, err = newCLI(WithExtraCommands(commandTest))
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.extraCommands).NotTo(BeNil())
			Expect(len(c.extraCommands)).To(Equal(1))
			Expect(c.extraCommands[0]).NotTo(BeNil())
			Expect(c.extraCommands[0].Use).To(Equal(commandTest.Use))
		})
	})

	Context("WithExtraAlphaCommands", func() {
		It("should return a valid CLI with extra alpha commands", func() {
			commandTest := &cobra.Command{
				Use: "example",
			}
			c, err = newCLI(WithExtraAlphaCommands(commandTest))
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.extraAlphaCommands).NotTo(BeNil())
			Expect(len(c.extraAlphaCommands)).To(Equal(1))
			Expect(c.extraAlphaCommands[0]).NotTo(BeNil())
			Expect(c.extraAlphaCommands[0].Use).To(Equal(commandTest.Use))
		})
	})

	Context("WithCompletion", func() {
		It("should not add the completion command by default", func() {
			c, err = newCLI()
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.completionCommand).To(BeFalse())
		})

		It("should add the completion command if requested", func() {
			c, err = newCLI(WithCompletion())
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.completionCommand).To(BeTrue())
		})
	})
})
