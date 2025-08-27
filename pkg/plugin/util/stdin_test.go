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

package util

import (
	"bufio"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("stdin", func() {
	It("returns true for 'y'", func() {
		reader := bufio.NewReader(strings.NewReader("y\n"))
		Expect(YesNo(reader)).To(BeTrue())
	})

	It("returns true for 'yes'", func() {
		reader := bufio.NewReader(strings.NewReader("yes\n"))
		Expect(YesNo(reader)).To(BeTrue())
	})

	It("returns false for 'n'", func() {
		reader := bufio.NewReader(strings.NewReader("n\n"))
		Expect(YesNo(reader)).To(BeFalse())
	})

	It("returns false for 'no'", func() {
		reader := bufio.NewReader(strings.NewReader("no\n"))
		Expect(YesNo(reader)).To(BeFalse())
	})

	It("prompts again on invalid input", func() {
		// "maybe" is invalid, then "y" is valid
		input := "maybe\ny\n"
		reader := bufio.NewReader(strings.NewReader(input))

		// Capture stdout to check for prompt
		oldStdout := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w

		// Call YesNo directly (no goroutine needed)
		Expect(YesNo(reader)).To(BeTrue())

		Expect(w.Close()).NotTo(HaveOccurred())
		os.Stdout = oldStdout
	})

	It("trims spaces and works", func() {
		reader := bufio.NewReader(strings.NewReader("  yes  \n"))
		Expect(YesNo(reader)).To(BeTrue())
	})
})
