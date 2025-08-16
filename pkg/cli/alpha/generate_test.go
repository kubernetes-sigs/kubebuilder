/*
Copyright 2023 The Kubernetes Authors.
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

//lint:ignore ST1001 we use dot-imports in tests for brevity

package alpha

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Figuring out ways to test run these tests similar to existing.
// Currently unable to run without this on VSCode. Will remove once done
func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "alpha")
}

var _ = Describe("NewScaffoldCommand", func() {
	When("NewScaffoldCommand", func() {
		It("Testing the NewScaffoldCommand", func() {
			cmd := NewScaffoldCommand()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(Equal(""))
			Expect(cmd.Use).To(ContainSubstring("generate"))
			Expect(cmd.Short).NotTo(Equal(""))
			Expect(cmd.Short).To(ContainSubstring("Re-scaffold a Kubebuilder project from its PROJECT file"))
			Expect(cmd.Example).NotTo(Equal(""))
			Expect(cmd.Example).To(ContainSubstring("kubebuilder alpha generate"))
		})
	})
})
