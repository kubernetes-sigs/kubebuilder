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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"
)

var _ = Describe("CLI options", func() {

	const (
		pluginName     = "plugin"
		pluginVersion  = "v1"
		projectVersion = "1"
	)

	var (
		c   *cli
		err error

		p   = newMockPlugin(pluginName, pluginVersion, projectVersion)
		np1 = newMockPlugin("Plugin", pluginVersion, projectVersion)
		np2 = mockPlugin{pluginName, plugin.Version{Number: -1, Stage: plugin.StableStage}, []string{projectVersion}}
		np3 = newMockPlugin(pluginName, pluginVersion)
		np4 = newMockPlugin(pluginName, pluginVersion, "a")
	)

	Context("WithRootCommandConfig", func() {
		It("should use the default command name and descriptions", func() {
			c, err = newCLI()
			shortDescription, longDescription, example := buildDefaultDescriptors(defaultCommandName)
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.cmdCfg.CommandName).To(Equal(defaultCommandName))
			Expect(c.cmdCfg.Short).To(Equal(shortDescription))
			Expect(c.cmdCfg.Long).To(Equal(longDescription))
			Expect(c.cmdCfg.Example).To(Equal(example))
		})

		It("should use the provided command name and default descriptions", func() {
			cfg := RootCommandConfig{
				CommandName: "other-command",
			}
			c, err = newCLI(WithRootCommandConfig(cfg))
			shortDescription, longDescription, example := buildDefaultDescriptors(cfg.CommandName)
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.cmdCfg.CommandName).To(Equal(cfg.CommandName))
			Expect(c.cmdCfg.Short).To(Equal(shortDescription))
			Expect(c.cmdCfg.Long).To(Equal(longDescription))
			Expect(c.cmdCfg.Example).To(Equal(example))
		})

		It("should use the provided command name and descriptions", func() {
			cfg := RootCommandConfig{
				CommandName: "other-command",
				Short:       "Short Description",
				Long:        "Longer Description of the command",
				Example:     "Example usage of the command",
			}
			c, err = newCLI(WithRootCommandConfig(cfg))
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.cmdCfg.CommandName).To(Equal(cfg.CommandName))
			Expect(c.cmdCfg.Short).To(Equal(cfg.Short))
			Expect(c.cmdCfg.Long).To(Equal(cfg.Long))
			Expect(c.cmdCfg.Example).To(Equal(cfg.Example))
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

	Context("WithDefaultProjectVersion", func() {
		It("should return a valid CLI", func() {
			defaultProjectVersions := []string{
				"1",
				"2",
				"3-alpha",
			}
			for _, defaultProjectVersion := range defaultProjectVersions {
				By(fmt.Sprintf("using %q", defaultProjectVersion))
				c, err = newCLI(WithDefaultProjectVersion(defaultProjectVersion))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.defaultProjectVersion).To(Equal(defaultProjectVersion))
			}
		})

		It("should return an error", func() {
			defaultProjectVersions := []string{
				"",         // Empty default project version
				"v1",       // 'v' prefix for project version
				"1alpha",   // non-delimited non-stable suffix
				"1.alpha",  // non-stable version delimited by '.'
				"1-alpha1", // number-suffixed non-stable version
			}
			for _, defaultProjectVersion := range defaultProjectVersions {
				By(fmt.Sprintf("using %q", defaultProjectVersion))
				_, err = newCLI(WithDefaultProjectVersion(defaultProjectVersion))
				Expect(err).To(HaveOccurred())
			}
		})
	})

	Context("WithDefaultPlugins", func() {
		It("should return a valid CLI", func() {
			c, err = newCLI(WithDefaultPlugins(projectVersion, p))
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.defaultPlugins).To(Equal(map[string][]string{projectVersion: {plugin.KeyFor(p)}}))
		})

		When("providing an invalid project version", func() {
			It("should return an error", func() {
				_, err = newCLI(WithDefaultPlugins("", p))
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
				_, err = newCLI(WithDefaultPlugins("2", p))
				Expect(err).To(HaveOccurred())
			})
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

	Context("WithCompletion", func() {
		It("should not add the completion command by default", func() {
			c, err = newCLI()
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.completionCommand).To(BeFalse())
		})

		It("should add the completion command if requested", func() {
			c, err = newCLI(WithCompletion)
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.completionCommand).To(BeTrue())
		})
	})

})
