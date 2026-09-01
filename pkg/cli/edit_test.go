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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ = Describe("newEditCmd", func() {
	var c *CLI
	var projectVersion config.Version

	BeforeEach(func() {
		projectVersion = config.Version{Number: 3}
		c = &CLI{}
	})

	When("no plugins are resolved", func() {
		BeforeEach(func() {
			c.resolvedPlugins = []plugin.Plugin{}
		})

		It("should create a command that fails with no resolved plugin error", func() {
			cmd := c.newEditCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).To(Equal("edit"))
			Expect(cmd.Short).To(ContainSubstring("Update the project configuration"))

			err := cmd.RunE(cmd, []string{})
			Expect(err).To(MatchError(noResolvedPluginError{}))
		})
	})

	When("resolved plugins do not implement Edit", func() {
		BeforeEach(func() {
			c.resolvedPlugins = []plugin.Plugin{
				newMockPlugin("noedit.kubebuilder.io", "v1", projectVersion),
			}
		})

		It("should create a command that fails with no available plugin error", func() {
			cmd := c.newEditCmd()
			Expect(cmd).NotTo(BeNil())

			err := cmd.RunE(cmd, []string{})
			Expect(err).To(MatchError(noAvailablePluginError{"edit project"}))
		})
	})

	When("resolved plugins implement Edit", func() {
		var testPlugin testEditPlugin

		BeforeEach(func() {
			testPlugin = testEditPlugin{
				name:        "test.kubebuilder.io",
				version:     plugin.Version{Number: 1},
				projectVers: []config.Version{projectVersion},
			}
			c.resolvedPlugins = []plugin.Plugin{testPlugin}
		})

		It("should create a command with proper metadata", func() {
			cmd := c.newEditCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).To(Equal("edit"))
			Expect(cmd.Short).To(ContainSubstring("Update the project configuration"))
		})
	})
})

type testEditPlugin struct {
	name        string
	version     plugin.Version
	projectVers []config.Version
}

func (p testEditPlugin) Name() string                               { return p.name }
func (p testEditPlugin) Version() plugin.Version                    { return p.version }
func (p testEditPlugin) SupportedProjectVersions() []config.Version { return p.projectVers }
func (p testEditPlugin) GetEditSubcommand() plugin.EditSubcommand {
	return &testEditSubcommand{}
}

type testEditSubcommand struct{}

func (s *testEditSubcommand) Scaffold(_ machinery.Filesystem) error { return nil }
