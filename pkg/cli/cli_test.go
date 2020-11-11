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
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	internalconfig "sigs.k8s.io/kubebuilder/v2/pkg/cli/internal/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"
)

// Test plugin types and constructors.
type mockPlugin struct {
	name            string
	version         plugin.Version
	projectVersions []string
}

func (p mockPlugin) Name() string                       { return p.name }
func (p mockPlugin) Version() plugin.Version            { return p.version }
func (p mockPlugin) SupportedProjectVersions() []string { return p.projectVersions }

func makeBasePlugin(name, version string, projVers ...string) plugin.Plugin {
	v, err := plugin.ParseVersion(version)
	if err != nil {
		panic(err)
	}
	return mockPlugin{name, v, projVers}
}

func makePluginsForKeys(keys ...string) (plugins []plugin.Plugin) {
	for _, key := range keys {
		n, v := plugin.SplitKey(key)
		plugins = append(plugins, makeBasePlugin(n, v, internalconfig.DefaultVersion))
	}
	return
}

type mockSubcommand struct{}

func (mockSubcommand) UpdateContext(*plugin.Context) {}
func (mockSubcommand) BindFlags(*pflag.FlagSet)      {}
func (mockSubcommand) InjectConfig(*config.Config)   {}
func (mockSubcommand) Run() error                    { return nil }

// nolint:maligned
type mockAllPlugin struct {
	mockPlugin
	mockInitPlugin
	mockCreateAPIPlugin
	mockCreateWebhookPlugin
	mockEditPlugin
}

type mockInitPlugin struct{ mockSubcommand }
type mockCreateAPIPlugin struct{ mockSubcommand }
type mockCreateWebhookPlugin struct{ mockSubcommand }
type mockEditPlugin struct{ mockSubcommand }

// GetInitSubcommand implements plugin.Init
func (p mockInitPlugin) GetInitSubcommand() plugin.InitSubcommand { return p }

// GetCreateAPISubcommand implements plugin.CreateAPI
func (p mockCreateAPIPlugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand { return p }

// GetCreateWebhookSubcommand implements plugin.CreateWebhook
func (p mockCreateWebhookPlugin) GetCreateWebhookSubcommand() plugin.CreateWebhookSubcommand {
	return p
}

// GetEditSubcommand implements plugin.Edit
func (p mockEditPlugin) GetEditSubcommand() plugin.EditSubcommand { return p }

func makeAllPlugin(name, version string, projectVersions ...string) plugin.Plugin {
	p := makeBasePlugin(name, version, projectVersions...).(mockPlugin)
	subcommand := mockSubcommand{}
	return mockAllPlugin{
		p,
		mockInitPlugin{subcommand},
		mockCreateAPIPlugin{subcommand},
		mockCreateWebhookPlugin{subcommand},
		mockEditPlugin{subcommand},
	}
}

func makeSetByProjVer(ps ...plugin.Plugin) map[string][]plugin.Plugin {
	set := make(map[string][]plugin.Plugin)
	for _, p := range ps {
		for _, version := range p.SupportedProjectVersions() {
			set[version] = append(set[version], p)
		}
	}
	return set
}

func setPluginsFlag(key string) {
	os.Args = append(os.Args, "init", "--"+pluginsFlag, key)
}

var _ = Describe("CLI", func() {

	var (
		c               CLI
		err             error
		pluginNameA     = "go.example.com"
		pluginNameB     = "go.test.com"
		projectVersions = []string{config.Version2, config.Version3Alpha}
		pluginAV1       = makeAllPlugin(pluginNameA, "v1", projectVersions...)
		pluginAV2       = makeAllPlugin(pluginNameA, "v2", projectVersions...)
		pluginBV1       = makeAllPlugin(pluginNameB, "v1", projectVersions...)
		pluginBV2       = makeAllPlugin(pluginNameB, "v2", projectVersions...)
		allPlugins      = []plugin.Plugin{pluginAV1, pluginAV2, pluginBV1, pluginBV2}
	)

	Describe("New", func() {

		Context("with no plugins specified", func() {
			It("should return a valid CLI", func() {
				By("setting one plugin")
				c, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(pluginAV1),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Plugin{pluginAV1}))

				By("setting two plugins with different names and versions")
				c, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(pluginAV1, pluginBV2),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginBV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Plugin{pluginAV1}))

				By("setting two plugins with the same names and different versions")
				c, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(pluginAV1, pluginAV2),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginAV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Plugin{pluginAV1}))

				By("setting two plugins with different names and the same version")
				c, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(pluginAV1, pluginBV1),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginBV1)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Plugin{pluginAV1}))
			})

			It("should return an error", func() {
				By("not setting any plugins or default plugins")
				_, err = New()
				Expect(err).To(MatchError(`no plugins for project version "3-alpha"`))

				By("not setting any plugin")
				_, err = New(
					WithDefaultPlugins(pluginAV1),
				)
				Expect(err).To(MatchError(`no plugins for project version "3-alpha"`))

				By("not setting any default plugins")
				_, err = New(
					WithPlugins(pluginAV1),
				)
				Expect(err).To(MatchError(`no default plugins for project version "3-alpha"`))

				By("setting two plugins of the same name and version")
				_, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(pluginAV1, pluginAV1),
				)
				Expect(err).To(MatchError(`broken pre-set plugins: two plugins have the same key: "go.example.com/v1"`))
			})
		})

		Context("with --plugins set", func() {

			var (
				args []string
			)

			BeforeEach(func() {
				args = os.Args
			})

			AfterEach(func() {
				os.Args = args
			})

			It("should return a valid CLI", func() {
				By(`setting cliPluginKey to "go"`)
				setPluginsFlag("go")
				c, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(pluginAV1, pluginAV2),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginAV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Plugin{pluginAV1}))

				By(`setting cliPluginKey to "go/v1"`)
				setPluginsFlag("go/v1")
				c, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(pluginAV1, pluginBV2),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginBV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Plugin{pluginAV1}))

				By(`setting cliPluginKey to "go/v2"`)
				setPluginsFlag("go/v2")
				c, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(pluginAV1, pluginBV2),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginBV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Plugin{pluginBV2}))

				By(`setting cliPluginKey to "go.test.com/v2"`)
				setPluginsFlag("go.test.com/v2")
				c, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(allPlugins...),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(allPlugins...)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Plugin{pluginBV2}))
			})

			It("should return an error", func() {
				By(`setting cliPluginKey to an non-existent key "foo"`)
				setPluginsFlag("foo")
				_, err = New(
					WithDefaultPlugins(pluginAV1),
					WithPlugins(pluginAV1, pluginAV2),
				)
				Expect(err).To(MatchError(errAmbiguousPlugin{
					key: "foo",
					msg: `no names match, possible plugins: ["go.example.com/v1" "go.example.com/v2"]`,
				}))
			})
		})

		Context("WithCommandName", func() {
			It("should use the provided command name", func() {
				commandName := "other-command"
				c, err = New(
					WithCommandName(commandName),
					WithDefaultPlugins(pluginAV1),
					WithPlugins(allPlugins...),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).commandName).To(Equal(commandName))
			})
		})

		Context("WithVersion", func() {
			It("should use the provided version string", func() {
				version := "Version: 0.0.0"
				c, err = New(WithVersion(version), WithDefaultPlugins(pluginAV1), WithPlugins(allPlugins...))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).version).To(Equal(version))
			})
		})

		Context("WithDefaultProjectVersion", func() {
			var defaultProjectVersion string

			It("should use the provided default project version", func() {
				By(`using version "2"`)
				defaultProjectVersion = "2"
				c, err = New(
					WithDefaultProjectVersion(defaultProjectVersion),
					WithDefaultPlugins(pluginAV1),
					WithPlugins(allPlugins...),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).defaultProjectVersion).To(Equal(defaultProjectVersion))

				By(`using version "3-alpha"`)
				defaultProjectVersion = "3-alpha"
				c, err = New(
					WithDefaultProjectVersion(defaultProjectVersion),
					WithDefaultPlugins(pluginAV1),
					WithPlugins(allPlugins...),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).defaultProjectVersion).To(Equal(defaultProjectVersion))
			})

			It("should fail for invalid project versions", func() {
				By(`using version "0"`)
				defaultProjectVersion = "0"
				c, err = New(
					WithDefaultProjectVersion(defaultProjectVersion),
					WithDefaultPlugins(pluginAV1),
					WithPlugins(allPlugins...),
				)
				Expect(err).To(HaveOccurred())

				By(`using version "1-gamma"`)
				defaultProjectVersion = "1-gamma"
				c, err = New(
					WithDefaultProjectVersion(defaultProjectVersion),
					WithDefaultPlugins(pluginAV1),
					WithPlugins(allPlugins...),
				)
				Expect(err).To(HaveOccurred())

				By(`using version "1alpha"`)
				defaultProjectVersion = "1alpha"
				c, err = New(
					WithDefaultProjectVersion(defaultProjectVersion),
					WithDefaultPlugins(pluginAV1),
					WithPlugins(allPlugins...),
				)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("WithExtraCommands", func() {
			It("should work successfully with extra commands", func() {
				commandTest := &cobra.Command{
					Use: "example",
				}
				c, err = New(
					WithExtraCommands(commandTest),
					WithDefaultPlugins(pluginAV1),
					WithPlugins(allPlugins...),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).extraCommands[0]).NotTo(BeNil())
				Expect(c.(*cli).extraCommands[0].Use).To(Equal(commandTest.Use))
			})
		})

		Context("WithCompletion", func() {
			It("should add the completion command if requested", func() {
				By("not providing WithCompletion")
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(allPlugins...))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).completionCommand).To(BeFalse())

				By("providing WithCompletion")
				c, err = New(WithCompletion, WithDefaultPlugins(pluginAV1), WithPlugins(allPlugins...))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).completionCommand).To(BeTrue())
			})
		})

	})

})
