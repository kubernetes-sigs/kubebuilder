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

package util

import (
	"bytes"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RunCmd", func() {
	var (
		output *bytes.Buffer
		err    error
	)
	BeforeEach(func() {
		output = &bytes.Buffer{}
	})
	AfterEach(func() {
		output.Reset()
	})
	It("executes the command and redirects output to stdout", func() {
		cmd := exec.Command("echo", "test")
		cmd.Stdout = output
		err = cmd.Run()
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(output.String())).To(Equal("test"))
	})
	It("returns an error if the command fails", func() {
		err = RunCmd("unknown command", "unknowncommand")
		Expect(err).To(HaveOccurred())
	})
})
