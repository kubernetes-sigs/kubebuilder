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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Version", func() {
	var (
		c *CLI
	)

	BeforeEach(func() {
		c = &CLI{}
	})

	Context("newVersionCmd", func() {
		It("Test the version", func() {
			cmd := c.newVersionCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).To(ContainSubstring("version"))
			Expect(cmd.Use).NotTo(Equal(""))
			Expect(cmd.Short).NotTo(Equal(""))
			Expect(cmd.Short).To(ContainSubstring("Print the"))
			Expect(cmd.Long).NotTo(Equal(""))
			Expect(cmd.Long).To(ContainSubstring("Print the"))
			Expect(cmd.Example).NotTo(Equal(""))
			Expect(cmd.Example).To(ContainSubstring("version"))
		})
	})
})
