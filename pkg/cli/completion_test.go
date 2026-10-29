/*
Copyright 2022 The Kubernetes Authors.

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
	"bytes"
	"errors"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Completion", func() {
	var c *CLI

	BeforeEach(func() {
		c = &CLI{}
	})

	When("newBashCmd", func() {
		It("Testing the BashCompletion", func() {
			cmd := c.newBashCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(Equal(""))
			Expect(cmd.Use).To(ContainSubstring("bash"))
			Expect(cmd.Short).NotTo(Equal(""))
			Expect(cmd.Short).To(ContainSubstring("Load bash completions"))
			Expect(cmd.Example).NotTo(Equal(""))
			Expect(cmd.Example).To(ContainSubstring("# To load completion for this session"))
		})
	})

	Context("newZshCmd", func() {
		It("Testing the ZshCompletion", func() {
			cmd := c.newZshCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(Equal(""))
			Expect(cmd.Use).To(ContainSubstring("zsh"))
			Expect(cmd.Short).NotTo(Equal(""))
			Expect(cmd.Short).To(ContainSubstring("Load zsh completions"))
			Expect(cmd.Example).NotTo(Equal(""))
			Expect(cmd.Example).To(ContainSubstring("# If shell completion is not already enabled in your environment"))
		})
	})

	Context("newFishCmd", func() {
		It("Testing the FishCompletion", func() {
			cmd := c.newFishCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(Equal(""))
			Expect(cmd.Use).To(ContainSubstring("fish"))
			Expect(cmd.Short).NotTo(Equal(""))
			Expect(cmd.Short).To(ContainSubstring("Load fish completions"))
			Expect(cmd.Example).NotTo(Equal(""))
			Expect(cmd.Example).To(ContainSubstring("# To load completion for this session, execute:"))
		})
	})

	Context("newPowerShellCmd", func() {
		It("Testing the PowerShellCompletion", func() {
			cmd := c.newPowerShellCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(Equal(""))
			Expect(cmd.Use).To(ContainSubstring("powershell"))
			Expect(cmd.Short).NotTo(Equal(""))
			Expect(cmd.Short).To(ContainSubstring("Load powershell completions"))
		})
	})

	// Tools that embed the CLI capture its output with SetOut.
	DescribeTable("writes the completion script to the command output",
		func(shell, want string) {
			root := &cobra.Command{Use: "kubebuilder"}
			root.AddCommand(c.newCompletionCmd())
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetArgs([]string{"completion", shell})
			Expect(root.Execute()).To(Succeed())
			Expect(out.String()).To(ContainSubstring(want))
		},
		Entry("bash", "bash", "bash completion V2 for kubebuilder"),
		Entry("zsh", "zsh", "zsh completion for kubebuilder"),
		Entry("fish", "fish", "fish completion for kubebuilder"),
		Entry("powershell", "powershell", "powershell completion for kubebuilder"),
	)

	DescribeTable("returns the error when the completion script can't be written",
		func(shell string) {
			writeErr := errors.New("write failed")
			root := &cobra.Command{Use: "kubebuilder"}
			root.AddCommand(c.newCompletionCmd())
			root.SetOut(failingWriter{err: writeErr})
			root.SetErr(io.Discard)
			root.SetArgs([]string{"completion", shell})
			Expect(root.Execute()).To(MatchError(writeErr))
		},
		Entry("bash", "bash"),
		Entry("zsh", "zsh"),
		Entry("fish", "fish"),
		Entry("powershell", "powershell"),
	)
})
