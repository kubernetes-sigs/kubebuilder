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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

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
