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
)

var _ = Describe("addAlphaCmd", func() {
	var c *CLI

	BeforeEach(func() {
		c = &CLI{
			cmd: &cobra.Command{Use: kubebuilderCommandName},
		}
	})

	It("should add alpha command to root when alpha commands exist", func() {
		c.addAlphaCmd()
		Expect(hasSubCommand(c.cmd, alphaCommand)).To(BeTrue())
	})

	It("should add alpha command to root when only extra alpha commands exist", func() {
		c.extraAlphaCommands = []*cobra.Command{
			{Use: "custom-alpha", Short: "Custom alpha command"},
		}
		c.addAlphaCmd()
		Expect(hasSubCommand(c.cmd, alphaCommand)).To(BeTrue())
	})

	When("no alpha commands exist", func() {
		BeforeEach(func() {
			original := alphaCommands
			DeferCleanup(func() { alphaCommands = original })
			alphaCommands = []*cobra.Command{}
			c.extraAlphaCommands = []*cobra.Command{}
		})

		It("should not add alpha command to root", func() {
			c.addAlphaCmd()
			Expect(hasSubCommand(c.cmd, alphaCommand)).To(BeFalse())
		})
	})
})

var _ = Describe("addExtraAlphaCommands", func() {
	var c *CLI

	BeforeEach(func() {
		c = &CLI{
			cmd: &cobra.Command{Use: kubebuilderCommandName},
		}
		c.addAlphaCmd()
	})

	It("should add extra alpha commands to the alpha subcommand", func() {
		c.extraAlphaCommands = []*cobra.Command{
			{Use: "extra-alpha", Short: "Extra alpha command"},
		}

		err := c.addExtraAlphaCommands()
		Expect(err).NotTo(HaveOccurred())

		alphaCmd, _ := c.cmd.Commands()[0], false
		for _, cmd := range c.cmd.Commands() {
			if cmd.Name() == alphaCommand {
				alphaCmd = cmd
				break
			}
		}
		Expect(hasSubCommand(alphaCmd, "extra-alpha")).To(BeTrue())
	})

	It("should reject a duplicate alpha command", func() {
		c.extraAlphaCommands = []*cobra.Command{
			{Use: "generate", Short: "Duplicate generate command"},
		}

		err := c.addExtraAlphaCommands()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already exists"))
	})

	It("should return an error when no alpha command exists", func() {
		c = &CLI{
			cmd: &cobra.Command{Use: kubebuilderCommandName},
			extraAlphaCommands: []*cobra.Command{
				{Use: "custom-alpha", Short: "Custom alpha command"},
			},
		}

		err := c.addExtraAlphaCommands()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no \"alpha\" command found"))
	})
})
