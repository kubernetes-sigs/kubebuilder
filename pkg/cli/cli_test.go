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

	internalconfig "sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
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

func (mockPlugin) UpdateContext(*plugin.Context) {}
func (mockPlugin) BindFlags(*pflag.FlagSet)      {}
func (mockPlugin) InjectConfig(*config.Config)   {}
func (mockPlugin) Run() error                    { return nil }

func makeBasePlugin(name, version string, projVers ...string) plugin.Base {
	v, err := plugin.ParseVersion(version)
	if err != nil {
		panic(err)
	}
	return mockPlugin{name, v, projVers}
}

func makePluginsForKeys(keys ...string) (plugins []plugin.Base) {
	for _, key := range keys {
		n, v := plugin.SplitKey(key)
		plugins = append(plugins, makeBasePlugin(n, v, internalconfig.DefaultVersion))
	}
	return
}

type mockAllPlugin struct {
	mockPlugin
	mockInitPlugin
	mockCreateAPIPlugin
	mockCreateWebhookPlugin
}

type mockInitPlugin struct{ mockPlugin }
type mockCreateAPIPlugin struct{ mockPlugin }
type mockCreateWebhookPlugin struct{ mockPlugin }

// GetInitPlugin will return the plugin which is responsible for initialized the project
func (p mockInitPlugin) GetInitPlugin() plugin.Init { return p }

// GetCreateAPIPlugin will return the plugin which is responsible for scaffolding APIs for the project
func (p mockCreateAPIPlugin) GetCreateAPIPlugin() plugin.CreateAPI { return p }

// GetCreateWebhookPlugin will return the plugin which is responsible for scaffolding webhooks for the project
func (p mockCreateWebhookPlugin) GetCreateWebhookPlugin() plugin.CreateWebhook { return p }

func makeAllPlugin(name, version string, projectVersions ...string) plugin.Base {
	p := makeBasePlugin(name, version, projectVersions...).(mockPlugin)
	return mockAllPlugin{
		p,
		mockInitPlugin{p},
		mockCreateAPIPlugin{p},
		mockCreateWebhookPlugin{p},
	}
}

func makeSetByProjVer(ps ...plugin.Base) map[string][]plugin.Base {
	set := make(map[string][]plugin.Base)
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
		allPlugins      = []plugin.Base{pluginAV1, pluginAV2, pluginBV1, pluginBV2}
	)

	Describe("New", func() {

		Context("with no plugins specified", func() {
			It("should return a valid CLI", func() {
				By("setting one plugin")
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(pluginAV1))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Base{pluginAV1}))

				By("setting two plugins with different names and versions")
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(pluginAV1, pluginBV2))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginBV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Base{pluginAV1}))

				By("setting two plugins with the same names and different versions")
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(pluginAV1, pluginAV2))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginAV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Base{pluginAV1}))

				By("setting two plugins with different names and the same version")
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(pluginAV1, pluginBV1))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginBV1)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Base{pluginAV1}))
			})

			It("should return an error", func() {
				By("not setting any plugins or default plugins")
				_, err = New()
				Expect(err).To(MatchError(`no plugins for project version "3-alpha"`))

				By("not setting any plugin")
				_, err = New(WithDefaultPlugins(pluginAV1))
				Expect(err).To(MatchError(`no plugins for project version "3-alpha"`))

				By("not setting any default plugins")
				_, err = New(WithPlugins(pluginAV1))
				Expect(err).To(MatchError(`no default plugins for project version "3-alpha"`))

				By("setting two plugins of the same name and version")
				_, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(pluginAV1, pluginAV1))
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
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(pluginAV1, pluginAV2))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginAV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Base{pluginAV1}))

				By(`setting cliPluginKey to "go/v1"`)
				setPluginsFlag("go/v1")
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(pluginAV1, pluginBV2))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginBV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Base{pluginAV1}))

				By(`setting cliPluginKey to "go/v2"`)
				setPluginsFlag("go/v2")
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(pluginAV1, pluginBV2))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(pluginAV1, pluginBV2)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Base{pluginBV2}))

				By(`setting cliPluginKey to "go.test.com/v2"`)
				setPluginsFlag("go.test.com/v2")
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(allPlugins...))
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.(*cli).pluginsFromOptions).To(Equal(makeSetByProjVer(allPlugins...)))
				Expect(c.(*cli).resolvedPlugins).To(Equal([]plugin.Base{pluginBV2}))
			})

			It("should return an error", func() {
				By(`setting cliPluginKey to an non-existent key "foo"`)
				setPluginsFlag("foo")
				_, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(pluginAV1, pluginAV2))
				Expect(err).To(MatchError(errAmbiguousPlugin{
					key: "foo",
					msg: `no names match, possible plugins: ["go.example.com/v1" "go.example.com/v2"]`,
				}))
			})
		})

		Context("with extra commands set", func() {
			It("should work successfully with extra commands", func() {
				setPluginsFlag("go.test.com/v2")
				commandTest := &cobra.Command{
					Use: "example",
				}
				c, err = New(WithDefaultPlugins(pluginAV1), WithPlugins(allPlugins...), WithExtraCommands(commandTest))
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
