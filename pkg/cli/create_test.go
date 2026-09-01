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
	golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
)

var _ = Describe("newCreateCmd", func() {
	var c *CLI
	var projectVersion config.Version

	BeforeEach(func() {
		projectVersion = config.Version{Number: 3}
	})

	When("CLI is configured with plugins", func() {
		BeforeEach(func() {
			var err error
			c, err = New(
				WithPlugins(&golangv4.Plugin{}),
				WithDefaultPlugins(projectVersion, &golangv4.Plugin{}),
				WithDefaultProjectVersion(projectVersion),
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create command with correct use", func() {
			cmd := c.newCreateCmd()
			Expect(cmd.Use).To(Equal("create"))
		})

		It("should suggest 'new' as alternative", func() {
			cmd := c.newCreateCmd()
			Expect(cmd.SuggestFor).To(ContainElement("new"))
		})

		It("should have short description", func() {
			cmd := c.newCreateCmd()
			Expect(cmd.Short).To(ContainSubstring("Scaffold a Kubernetes API or webhook"))
		})

		It("should have long description with plugin table", func() {
			cmd := c.newCreateCmd()
			Expect(cmd.Long).To(ContainSubstring("create api"))
			Expect(cmd.Long).To(ContainSubstring("create webhook"))
			Expect(cmd.Long).To(ContainSubstring("Available plugins"))
		})

		It("should exclude default scaffold plugins from plugin table", func() {
			cmd := c.newCreateCmd()
			// The plugin table in create command should not show go/v4 or kustomize
			// since they are the default scaffold bundle
			Expect(cmd.Long).NotTo(ContainSubstring("go/v4"))
			Expect(cmd.Long).NotTo(ContainSubstring("kustomize"))
		})
	})

	When("CLI has no plugins", func() {
		BeforeEach(func() {
			c = &CLI{}
		})

		It("should still create the command", func() {
			cmd := c.newCreateCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).To(Equal("create"))
		})

		It("should show no plugins message", func() {
			cmd := c.newCreateCmd()
			Expect(cmd.Long).To(ContainSubstring("No plugins available"))
		})
	})
})
