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

	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

var _ = Describe("CLI", func() {

	var (
		c               CLI
		err             error
		pluginNameA     = "go.example.com"
		pluginNameB     = "go.test.com"
		projectVersions = []string{config.Version2, config.Version3Alpha}
		pluginAV1       = makeAllPlugin(pluginNameA, "v1.0", projectVersions...)
		pluginAV2       = makeAllPlugin(pluginNameA, "v2.0", projectVersions...)
		pluginBV1       = makeAllPlugin(pluginNameB, "v1.0", projectVersions...)
		pluginBV2       = makeAllPlugin(pluginNameB, "v2.0", projectVersions...)
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
				Expect(err).To(MatchError(`broken pre-set plugins: two plugins have the same key: "go.example.com/v1.0"`))
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
				Expect(err).To(MatchError(errAmbiguousPlugin{"foo", "no names match"}))
			})
		})

	})

})

func setPluginsFlag(key string) {
	os.Args = append(os.Args, "init", "--"+pluginsFlag, key)
}
