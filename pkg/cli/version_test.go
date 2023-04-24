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

var _ = Describe("newVersionCmd", func() {
	var c CLI

	BeforeEach(func() {
		c = CLI{
			version:     "1.0.0",
			commandName: "myapp",
		}
	})

	It("should have the correct command name and usage message", func() {
		cmd := c.newVersionCmd()
		Expect(cmd.Use).To(Equal("version"))
		Expect(cmd.Short).To(Equal("Print the myapp version"))
		Expect(cmd.Long).To(Equal("Print the myapp version"))
		Expect(cmd.Example).To(Equal("myapp version"))
	})
})
