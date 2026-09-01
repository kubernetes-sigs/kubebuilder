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
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ = Describe("newCreateAPICmd", func() {
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
			cmd := c.newCreateAPICmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).To(Equal("api"))
			Expect(cmd.Short).To(ContainSubstring("Scaffold a Kubernetes API"))

			err := cmd.RunE(cmd, []string{})
			Expect(err).To(MatchError(noResolvedPluginError{}))
		})
	})

	When("resolved plugins do not implement CreateAPI", func() {
		BeforeEach(func() {
			c.resolvedPlugins = []plugin.Plugin{
				newMockPlugin("noapi.kubebuilder.io", "v1", projectVersion),
			}
		})

		It("should create a command that fails with no available plugin error", func() {
			cmd := c.newCreateAPICmd()
			Expect(cmd).NotTo(BeNil())

			err := cmd.RunE(cmd, []string{})
			Expect(err).To(MatchError(noAvailablePluginError{"API creation"}))
		})
	})

	When("resolved plugins implement CreateAPI", func() {
		var testPlugin testCreateAPIPlugin

		BeforeEach(func() {
			testPlugin = newTestCreateAPIPlugin("test.kubebuilder.io", plugin.Version{Number: 1})
			c.resolvedPlugins = []plugin.Plugin{testPlugin}
		})

		It("should create a command with proper metadata", func() {
			cmd := c.newCreateAPICmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).To(Equal("api"))
			Expect(cmd.Short).To(ContainSubstring("Scaffold a Kubernetes API"))
		})

		It("should have valid args function that suggests flags", func() {
			cmd := c.newCreateAPICmd()
			Expect(cmd.ValidArgsFunction).NotTo(BeNil())

			completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
			Expect(completions).To(HaveLen(1))
			Expect(completions[0]).To(ContainSubstring("'--'"))
			Expect(directive).To(Equal(cobra.ShellCompDirectiveNoFileComp))
		})

		It("should not show flag hint when args are provided", func() {
			cmd := c.newCreateAPICmd()
			completions, _ := cmd.ValidArgsFunction(cmd, []string{"--group"}, "")
			Expect(completions).To(BeEmpty())
		})

		It("should not show flag hint when already typing a flag", func() {
			cmd := c.newCreateAPICmd()
			completions, _ := cmd.ValidArgsFunction(cmd, []string{}, "--")
			Expect(completions).To(BeEmpty())
		})
	})
})
