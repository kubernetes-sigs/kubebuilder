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
)

// failingWriter fails every write with err.
type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

var _ = Describe("Version", func() {
	var c *CLI

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

		// Tools that embed the CLI capture its output with SetOut.
		It("writes the version to the command output", func() {
			c.version = "v1.2.3"
			cmd := c.newVersionCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{})
			Expect(cmd.Execute()).To(Succeed())
			Expect(out.String()).To(Equal("v1.2.3\n"))
		})

		It("returns the error when the version can't be written", func() {
			writeErr := errors.New("write failed")
			cmd := c.newVersionCmd()
			cmd.SetOut(failingWriter{err: writeErr})
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{})
			Expect(cmd.Execute()).To(MatchError(writeErr))
		})
	})
})
