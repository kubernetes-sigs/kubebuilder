/*
Copyright 2025 The Kubernetes Authors.

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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("cmd_helpers", func() {
	Context("error types", func() {
		It("noResolvedPluginError should return correct message", func() {
			err := noResolvedPluginError{}
			Expect(err.Error()).To(ContainSubstring("no resolved plugin"))
			Expect(err.Error()).To(ContainSubstring("verify the project version and plugins"))
		})

		It("noAvailablePluginError should return correct message with subcommand", func() {
			err := noAvailablePluginError{subcommand: "init"}
			Expect(err.Error()).To(ContainSubstring("init"))
			Expect(err.Error()).To(ContainSubstring("do not provide any"))
		})
	})

	Context("cmdErr", func() {
		It("should update command with error information", func() {
			cmd := &cobra.Command{
				Long: "Original description",
				RunE: func(*cobra.Command, []string) error {
					return nil
				},
			}
			testError := errors.New("test error")

			cmdErr(cmd, testError)

			Expect(cmd.Long).To(ContainSubstring("Original description"))
			Expect(cmd.Long).To(ContainSubstring("test error"))
			Expect(cmd.RunE).NotTo(BeNil())

			// Execute the modified RunE to verify it returns the error
			err := cmd.RunE(cmd, []string{})
			Expect(err).To(Equal(testError))
		})
	})

	Context("errCmdFunc", func() {
		It("should return a function that returns the provided error", func() {
			testError := errors.New("test error")
			runE := errCmdFunc(testError)

			err := runE(nil, nil)
			Expect(err).To(Equal(testError))
		})
	})
})
