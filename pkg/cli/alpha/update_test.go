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

package alpha

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewUpdateCommand", func() {
	When("NewUpdateCommand", func() {
		It("Testing the NewUpdateCommand", func() {
			cmd := NewUpdateCommand()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).To(ContainSubstring("update"))
			Expect(cmd.Short).NotTo(Equal(""))
			Expect(cmd.Short).To(ContainSubstring("Update your project to a newer version"))

			flags := cmd.Flags()
			Expect(flags.Lookup("from-version")).NotTo(BeNil())
			Expect(flags.Lookup("to-version")).NotTo(BeNil())
			Expect(flags.Lookup("from-branch")).NotTo(BeNil())
			Expect(flags.Lookup("force")).NotTo(BeNil())
			Expect(flags.Lookup("show-commits")).NotTo(BeNil())
			Expect(flags.Lookup("preserve-path")).NotTo(BeNil())
			Expect(flags.Lookup("output-branch")).NotTo(BeNil())
			Expect(flags.Lookup("push")).NotTo(BeNil())
		})
	})
})
